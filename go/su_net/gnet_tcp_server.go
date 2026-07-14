package su_net

import (
	"context"
	"go.local/su_errors"
	slog "go.local/su_log"
	"go.local/su_util"
	"sync"
	"sync/atomic"
	"time"

	"github.com/panjf2000/gnet/v2"
	"go.uber.org/zap"
)

// GTcpServer 基于 gnet 提供 TCP 服务端连接管理、包分发和请求响应处理。
type GTcpServer struct {
	gnet.BuiltinEventEngine                  // gnet 事件引擎嵌入字段。
	pool                    *su_util.GoPool  // 包处理 worker 池。
	Stat                    int32            // 服务状态：0 停止、1 初始化、2 启动。
	Addr                    string           // 用户传入的监听地址。
	protoAddr               string           // gnet 使用的协议地址。
	dispatchMode            GNetDispatchMode // 包处理分发模式。
	closeOnce               sync.Once        // 保证 Close 只执行一次。
	connMap                 sync.Map         // 远端地址到 GNetConn 的映射。
	closeTimeout            time.Duration    // gnet Stop 和 worker 池排空超时。
	dataHandler             *TcpNetHandler   // 业务数据包处理函数。
}

// OnBoot 是 gnet 服务启动完成回调。
func (ts *GTcpServer) OnBoot(eng gnet.Engine) (action gnet.Action) {
	slog.Info("server init finish !!!!", zap.String("listen addr", ts.protoAddr))
	atomic.StoreInt32(&ts.Stat, 2)
	return
}

// OnShutdown 是 gnet 服务关闭回调。
func (ts *GTcpServer) OnShutdown(eng gnet.Engine) {
	slog.Info("server shutdown !!!!", zap.String("listen addr", ts.protoAddr))
	atomic.StoreInt32(&ts.Stat, 0)
}

// OnOpen 是 gnet 新连接回调，会创建并保存 GNetConn。
func (ts *GTcpServer) OnOpen(c gnet.Conn) (out []byte, action gnet.Action) {
	slog.Info("new conn ", zap.String("remote addr", c.RemoteAddr().String()), zap.String("local addr", c.LocalAddr().String()))
	gconn := NewGnetConn(c)
	c.SetContext(gconn)
	ts.connMap.Store(c.RemoteAddr().String(), gconn)
	return
}

// OnClose 是 gnet 连接关闭回调，会标记连接关闭并从连接表移除。
func (ts *GTcpServer) OnClose(c gnet.Conn, err error) (action gnet.Action) {
	slog.Info("close conn", zap.String("remote addr", c.RemoteAddr().String()), zap.String("local addr", c.LocalAddr().String()),
		zap.Error(err))
	if gconn, ok := c.Context().(*GNetConn); ok {
		gconn.markClosed()
	}
	ts.connMap.Delete(c.RemoteAddr().String())
	return
}

// pingHandler 处理客户端 PING 并返回 PONG。
func pingHandler(gnc *GNetConn, rq_dp *DataProtocol) {
	var rs_dp DataProtocol
	micro_time := uint64(time.Now().UnixNano() / 1000)
	rs_dp.Head.PackId = PONG
	rs_dp.Head.HeadUuid = micro_time
	rs_dp.Head.RouteId = rq_dp.Head.RouteId
	ping, err := PingDecode(rq_dp.Data, rq_dp.Head.PackLen)
	if err != nil {
		slog.Error("Ping 解包失败", zap.Error(err))
		return
	}
	slog.Info("Ping心跳", zap.String("remote addr", gnc.RemoteAddr),
		zap.Uint64("ping time", ping.SendTime))
	pong := Pong{SendTime: micro_time, PingTime: ping.SendTime}
	rs_dp.Data, err = PongEncode(pong)
	if err != nil {
		slog.Error("Pong 封包失败", zap.Error(err))
		return
	}
	if err := gnc.Send(&rs_dp); err != nil {
		slog.Error("send pong failed", zap.Error(err))
	}
	return
}

// OnTraffic 是 gnet 读事件回调，会解析完整包并分发处理。
func (ts *GTcpServer) OnTraffic(c gnet.Conn) (action gnet.Action) {
	frame, err := c.Next(-1)
	if err != nil {
		slog.Error("server read gnet frame failed", zap.Error(err))
		return gnet.Close
	}
	if len(frame) == 0 {
		return
	}
	gconn, ok := c.Context().(*GNetConn)
	if !ok || gconn == nil {
		slog.Error("未保存的链接", zap.String("remote addr", c.RemoteAddr().String()))
		return
	}
	gconn.Recv(frame, func(dp *DataProtocol) {
		ts.dispatch(gconn, dp)
	})
	return
}

// dispatch 根据配置选择在线处理或提交到 worker 池处理。
func (ts *GTcpServer) dispatch(gconn *GNetConn, dp *DataProtocol) {
	taskDP := *dp
	switch ts.dispatchMode {
	case GNetDispatchInline:
		ts.handleServerPacket(gconn, &taskDP)
	default:
		if !ts.pool.SendTask(taskDP.Head.RouteId, func() {
			ts.handleServerPacket(gconn, &taskDP)
		}) {
			slog.Warn("gnet server task dropped", zap.Uint64("route_id", taskDP.Head.RouteId))
		}
	}
}

// handleServerPacket 处理服务端收到的 PING 或已注册请求包。
func (ts *GTcpServer) handleServerPacket(gconn *GNetConn, dp *DataProtocol) {
	if dp.Head.PackId == PING {
		pingHandler(gconn, dp)
		return
	}
	if ts == nil || ts.dataHandler == nil {
		slog.Error("gnet server handler unavailable")
		return
	}
	dispatchTcpNetHandler(ts.dataHandler, &HandlerContext{Conn: gconn, Packet: dp})
}

// Close 停止 gnet server 并等待 worker 池排空。
func (ts *GTcpServer) Close() {
	if ts == nil {
		return
	}
	ts.closeOnce.Do(func() {
		atomic.StoreInt32(&ts.Stat, 0)
		timeout := ts.closeTimeout
		if timeout <= 0 {
			timeout = DEFAULT_CLOSE_TIMEOUT
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		if err := gnet.Stop(ctx, ts.protoAddr); err != nil {
			slog.Error("gnet server stop failed", zap.String("addr", ts.protoAddr), zap.Error(err))
		}
		cancel()
		if ts.pool != nil && !ts.pool.StopAndDrain(timeout) {
			slog.Warn("gnet server pool drain timeout")
		}
	})
}

// State 返回服务端状态：0 停止、1 初始化、2 启动。
func (ts *GTcpServer) State() int32 {
	if ts == nil {
		return 0
	}
	return atomic.LoadInt32(&ts.Stat)
}

// ConnCount 返回当前活跃 gnet 连接数量。
func (ts *GTcpServer) ConnCount() int {
	if ts == nil {
		return 0
	}
	count := 0
	ts.connMap.Range(func(k, v interface{}) bool {
		count++
		return true
	})
	return count
}

func (ts *GTcpServer) RegisterManualResponseHandler(rqPackId uint32, rsPackId uint32, handler MessageHandler) error {
	if ts == nil || ts.dataHandler == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "gnet server or dataHandler is nil")
	}
	return ts.dataHandler.RegisterManualResponseHandler(rqPackId, rsPackId, handler)
}

func (ts *GTcpServer) RegisterRequestResponseHandler(rqPackId uint32, rsPackId uint32, handler MessageHandler) error {
	if ts == nil || ts.dataHandler == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "gnet server or dataHandler is nil")
	}
	return ts.dataHandler.RegisterRequestResponseHandler(rqPackId, rsPackId, handler)
}

func (ts *GTcpServer) RegisterOneWayHandler(packId uint32, handler MessageHandler) error {
	if ts == nil || ts.dataHandler == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "gnet server or dataHandler is nil")
	}
	return ts.dataHandler.RegisterOneWayHandler(packId, handler)
}

// Run 阻塞运行 gnet TCP server。
func (ts *GTcpServer) Run() {
	paddr := ts.protoAddr
	err := gnet.Run(ts, paddr, gnet.WithMulticore(true))
	if err != nil {
		slog.Error("create server failed", zap.String("addr: ", paddr), zap.Error(err))
		return
	}
}

// CreateGNetServer 创建 gnet TCP server，但不立即运行。
func CreateGNetServer(a_addr string) *GTcpServer {
	ts := &GTcpServer{dispatchMode: GNetDispatchPool, Addr: a_addr, protoAddr: "tcp://:" + a_addr, Stat: 1, closeTimeout: DEFAULT_CLOSE_TIMEOUT, dataHandler: newTcpNetHandler()}
	ts.pool = su_util.NewGoPool(DEFAULT_POOL_WORKERS, DEFAULT_POOL_QUEUE_SIZE)
	return ts
}

// CreateServer 是 CreateGNetServer 的兼容别名。
func CreateServer(a_addr string) *GTcpServer {
	return CreateGNetServer(a_addr)
}

// SetDispatchMode 设置 gnet 服务端包处理模式。
func (ts *GTcpServer) SetDispatchMode(mode GNetDispatchMode) {
	if ts == nil {
		return
	}
	ts.dispatchMode = mode
}

// SetCloseTimeout 设置 gnet server 关闭等待超时。
func (ts *GTcpServer) SetCloseTimeout(timeout time.Duration) {
	if ts == nil {
		return
	}
	ts.closeTimeout = timeout
}
