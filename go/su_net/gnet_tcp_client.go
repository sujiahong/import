package su_net

import (
	"go.local/su_errors"
	slog "go.local/su_log"
	"go.local/su_util"
	"sync"
	"sync/atomic"
	"time"

	"github.com/panjf2000/gnet/v2"
	"go.uber.org/zap"
)

// GTcpClient 基于 gnet 管理多条 TCP 客户端连接、心跳、重连和业务包分发。
type GTcpClient struct {
	gnet.BuiltinEventEngine                  // gnet 事件引擎嵌入字段。
	*gnet.Client                             // 底层 gnet client。
	remote_addr             string           // 远端连接地址。
	cfgConnNum              uint8            // 期望维持的连接数量。
	state                   int32            // 客户端状态：0 停止、1 连接中、2 已连接。
	reconnectState          int32            // 重连调度状态：0 未调度、1 已调度。
	connMap                 sync.Map         // 本地地址到 GNetConn 的映射。
	connMu                  sync.RWMutex     // 保护 connList 和 pool 惰性创建。
	connList                []*GNetConn      // 当前可用连接列表。
	pool                    *su_util.GoPool  // 包处理 worker 池。
	dataHandler             *TcpNetHandler   // 业务数据包处理函数。
	dispatchMode            GNetDispatchMode // 包处理分发模式。
	sendSeq                 uint64           // 轮询连接发送的序号。
	stopOnce                sync.Once        // 保证 Stop 只执行一次。
	stopErr                 error            // Stop 返回的底层错误。
}

// OnOpen 是 gnet 连接建立回调，会注册连接并发送初始心跳。
func (tc *GTcpClient) OnOpen(c gnet.Conn) (out []byte, action gnet.Action) {
	slog.Info("client new conn ", zap.String("remote addr", c.RemoteAddr().String()),
		zap.String("local addr", c.LocalAddr().String()))
	gconn := NewGnetConn(c)
	c.SetContext(gconn)
	tc.connMap.Store(c.LocalAddr().String(), gconn)
	tc.connMu.Lock()
	tc.connList = append(tc.connList, gconn)
	tc.connMu.Unlock()
	atomic.StoreInt32(&tc.state, 2)
	atomic.StoreInt32(&tc.reconnectState, 0)
	if err := gconn.Ping(); err != nil {
		slog.Error("client initial ping failed", zap.Error(err))
	}
	return
}

// OnClose 是 gnet 连接关闭回调，会移除连接并在客户端未停止时触发重连。
func (tc *GTcpClient) OnClose(c gnet.Conn, err error) (action gnet.Action) {
	slog.Info("client close conn", zap.String("remote addr", c.RemoteAddr().String()), zap.Error(err),
		zap.String("local addr", c.LocalAddr().String()))
	tc.connMap.Delete(c.LocalAddr().String())
	tc.connMu.Lock()
	closedConn, _ := c.Context().(*GNetConn)
	for i, conn := range tc.connList {
		if conn.Gconn == c {
			tc.connList = append(tc.connList[:i], tc.connList[i+1:]...)
			break
		}
	}
	tc.connMu.Unlock()
	if closedConn != nil {
		closedConn.markClosed()
	}
	if atomic.LoadInt32(&tc.state) != 0 {
		tc.Reconnect()
	}
	return
}

// OnTick 是 gnet 定时回调，用于心跳检查和连接数补齐。
func (tc *GTcpClient) OnTick() (delay time.Duration, action gnet.Action) {
	slog.Info("client tick, 发送 心跳", zap.Int32("tc.reconnectState", atomic.LoadInt32(&tc.reconnectState)), zap.Int32("tc.state", atomic.LoadInt32(&tc.state)))
	delay = time.Duration(PING_PONG_INTERVAL) * time.Second
	var count uint8 = 0
	tc.connMap.Range(func(k, v interface{}) bool {
		key_str := k.(string)
		gconn := v.(*GNetConn)
		slog.Info("定时连接检查", zap.String("key_str: ", key_str))
		gconn.CheckPong()
		count++
		return true
	})
	if count != tc.cfgConnNum {
		slog.Error("现有连接数量!=配置连接数量", zap.Uint8("count", count), zap.Uint8("tc.cfgConnNum", tc.cfgConnNum))
		if atomic.LoadInt32(&tc.reconnectState) == 0 && atomic.LoadInt32(&tc.state) == 2 {
			if tc.cfgConnNum > count {
				n := tc.cfgConnNum - count
				var i uint8
				for i = 0; i < n; i++ {
					tc.Connect()
				}
			}
		}
	}
	return
}

// pongHandler 处理 gnet 客户端收到的 PONG 心跳响应。
func pongHandler(gnc *GNetConn, rs_dp *DataProtocol) {
	pong, err := PongDecode(rs_dp.Data, rs_dp.Head.PackLen)
	if err != nil {
		slog.Error("pong 解包失败", zap.Error(err))
		return
	}
	slog.Info("pong心跳", zap.String("remote addr", gnc.RemoteAddr),
		zap.Uint64("pong time", pong.SendTime), zap.Uint64("ping time", pong.PingTime))
	if _, ok := gnc.PingPongMap.LoadAndDelete(pong.PingTime); ok {
		atomic.AddInt32(&gnc.pendingPings, -1)
	} else {
		slog.Error("PingPongMap没有 PingTime key", zap.Uint64("ping time", pong.PingTime))
	}
}

// OnTraffic 是 gnet 读事件回调，会解析完整包并分发处理。
func (tc *GTcpClient) OnTraffic(c gnet.Conn) (action gnet.Action) {
	frame, err := c.Next(-1)
	if err != nil {
		slog.Error("client read gnet frame failed", zap.Error(err))
		return gnet.Close
	}
	if len(frame) == 0 {
		return
	}
	gconn, ok := c.Context().(*GNetConn)
	if !ok || gconn == nil {
		slog.Error("client missing conn context", zap.String("local addr", c.LocalAddr().String()))
		return
	}
	gconn.Recv(frame, func(dp *DataProtocol) {
		tc.dispatch(gconn, dp)
	})
	return
}

// dispatch 根据配置选择在线处理或提交到 worker 池处理。
func (tc *GTcpClient) dispatch(gconn *GNetConn, dp *DataProtocol) {
	taskDP := *dp
	switch tc.dispatchMode {
	case GNetDispatchPool:
		pool := tc.ensurePool()
		if pool == nil || !pool.SendTask(taskDP.Head.RouteId, func() {
			tc.handleClientPacket(gconn, &taskDP)
		}) {
			slog.Warn("gnet client task dropped", zap.Uint64("route_id", taskDP.Head.RouteId))
		}
	default:
		tc.handleClientPacket(gconn, &taskDP)
	}
}

// ensurePool 惰性创建 gnet 客户端包处理 worker 池。
func (tc *GTcpClient) ensurePool() *su_util.GoPool {
	if tc == nil {
		return nil
	}
	tc.connMu.Lock()
	defer tc.connMu.Unlock()
	if tc.pool == nil {
		tc.pool = su_util.NewGoPool(DEFAULT_POOL_WORKERS, DEFAULT_POOL_QUEUE_SIZE)
	}
	return tc.pool
}

// handleClientPacket 处理客户端收到的 PONG 或已注册响应包。
func (tc *GTcpClient) handleClientPacket(gconn *GNetConn, dp *DataProtocol) {
	if dp.Head.PackId == PONG {
		pongHandler(gconn, dp)
		return
	}
	if tc == nil || tc.dataHandler == nil {
		slog.Error("gnet client handler unavailable")
		return
	}
	dispatchTcpNetHandler(tc.dataHandler, &HandlerContext{Conn: gconn, Packet: dp})
}

// sendBytes 将已编码数据轮询发送到当前可用连接。
func (tc *GTcpClient) sendBytes(aBytes []byte) error {
	tc.connMu.RLock()
	connCount := len(tc.connList)
	if connCount == 0 {
		tc.connMu.RUnlock()
		return su_errors.NewRetryable(su_errors.CodeUnavailable, "no active tcp client connection")
	}
	target := int(atomic.AddUint64(&tc.sendSeq, 1)-1) % connCount
	gconn := tc.connList[target]
	tc.connMu.RUnlock()
	return gconn.SendBytes(aBytes)
}

// Send 通过当前可用连接发送数据包。
func (tc *GTcpClient) Send(dp *DataProtocol) error {
	if tc == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "gnet client is nil")
	}
	bs, err := Encode(dp)
	if err != nil {
		return err
	}
	return tc.SendBytes(bs)
}

// SendBytes 发送已编码数据，空数据会被忽略。
func (tc *GTcpClient) SendBytes(bs []byte) error {
	if tc == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "gnet client is nil")
	}
	if len(bs) == 0 {
		return nil
	}
	return tc.sendBytes(bs)
}

// Stop 停止 gnet client 并关闭 worker 池。
func (tc *GTcpClient) Stop() (err error) {
	if tc == nil {
		return nil
	}
	tc.stopOnce.Do(func() {
		atomic.StoreInt32(&tc.state, 0)
		if tc.Client != nil {
			tc.stopErr = tc.Client.Stop()
		}
		if tc.pool != nil && !tc.pool.StopAndDrain(DEFAULT_CLOSE_TIMEOUT) {
			slog.Warn("gnet client pool drain timeout")
		}
		slog.Info("client stop ", zap.Error(tc.stopErr)) /////关闭客户端
	})
	return tc.stopErr
}

// State 返回客户端状态：0 停止、1 连接中、2 已连接。
func (tc *GTcpClient) State() int32 {
	if tc == nil {
		return 0
	}
	return atomic.LoadInt32(&tc.state)
}

// ConnCount 返回当前活跃 gnet 连接数量。
func (tc *GTcpClient) ConnCount() int {
	if tc == nil {
		return 0
	}
	tc.connMu.RLock()
	defer tc.connMu.RUnlock()
	return len(tc.connList)
}

// Connect 发起一次到远端地址的 gnet TCP 连接。
func (tc *GTcpClient) Connect() error {
	if atomic.LoadInt32(&tc.state) == 0 {
		return su_errors.New(su_errors.CodeUnavailable, "client stopped")
	}
	atomic.StoreInt32(&tc.state, 1)
	conn, err := tc.Client.Dial("tcp", tc.remote_addr)
	if err != nil {
		atomic.CompareAndSwapInt32(&tc.state, 1, 2)
		slog.Error("client dial failed", zap.String("addr: ", tc.remote_addr), zap.Error(err))
		return su_errors.WrapRetryable(su_errors.CodeUnavailable, "client dial failed", err)
	}
	slog.Info("client connect", zap.String("remote addr:", conn.RemoteAddr().String()),
		zap.String("local addr:", conn.LocalAddr().String()))
	return nil
}

// Reconnect 延迟发起重连，并确保同一时间只有一个重连调度在运行。
func (tc *GTcpClient) Reconnect() {
	if atomic.LoadInt32(&tc.state) == 0 {
		return
	}
	if !atomic.CompareAndSwapInt32(&tc.reconnectState, 0, 1) {
		return
	}
	su_util.DelayRun(RECONNECT_INTERVAL*1000, func() {
		err := tc.Connect()
		if err != nil {
			atomic.StoreInt32(&tc.reconnectState, 0)
			tc.Reconnect()
		}
	})
}

func (tc *GTcpClient) RegisterManualResponseHandler(rqPackId uint32, rsPackId uint32, handler MessageHandler) error {
	if tc == nil || tc.dataHandler == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "gnet client or dataHandler is nil")
	}
	return tc.dataHandler.RegisterManualResponseHandler(rqPackId, rsPackId, handler)
}

func (tc *GTcpClient) RegisterRequestResponseHandler(rqPackId uint32, rsPackId uint32, handler MessageHandler) error {
	if tc == nil || tc.dataHandler == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "gnet client or dataHandler is nil")
	}
	return tc.dataHandler.RegisterRequestResponseHandler(rqPackId, rsPackId, handler)
}

func (tc *GTcpClient) RegisterOneWayHandler(packId uint32, handler MessageHandler) error {
	if tc == nil || tc.dataHandler == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "gnet client or dataHandler is nil")
	}
	return tc.dataHandler.RegisterOneWayHandler(packId, handler)
}

// CreateGNetClient 创建 gnet TCP client 并建立指定数量连接。
func CreateGNetClient(a_addr string, a_conn_num uint8) *GTcpClient {
	tc := &GTcpClient{remote_addr: a_addr, state: 1, cfgConnNum: a_conn_num, dispatchMode: GNetDispatchInline, dataHandler: newTcpNetHandler()}

	client, err := gnet.NewClient(tc, gnet.WithTCPNoDelay(gnet.TCPDelay), gnet.WithTCPKeepAlive(30*time.Second), gnet.WithTicker(true))
	if err != nil {
		slog.Error("create client failed", zap.String("addr: ", a_addr))
		return nil
	}
	err = client.Start()
	if err != nil {
		slog.Error("client start failed", zap.String("addr: ", a_addr))
		return nil
	}
	tc.Client = client
	var i uint8
	for i = 0; i < a_conn_num; i++ {
		tc.Connect()
	}
	return tc
}

// CreateClient 是 CreateGNetClient 的兼容别名。
func CreateClient(a_addr string, a_conn_num uint8) *GTcpClient {
	return CreateGNetClient(a_addr, a_conn_num)
}

// SetDispatchMode 设置 gnet 客户端包处理模式。
func (tc *GTcpClient) SetDispatchMode(mode GNetDispatchMode) {
	if tc == nil {
		return
	}
	tc.dispatchMode = mode
	if mode == GNetDispatchPool {
		tc.ensurePool()
	}
}
