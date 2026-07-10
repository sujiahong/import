package su_net

import (
	"context"
	slog "go.local/su_log"
	"go.local/su_util"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/panjf2000/gnet/v2"
	"go.uber.org/zap"
)

type GTcpServer struct {
	gnet.BuiltinEventEngine                 ////匿名字段   事件服务
	pool                    *su_util.GoPool ///协程池
	Stat                    int32           /// 服务状态 0 停止 1 初始化 2 启动
	Addr                    string          ////监听地址
	protoAddr               string
	dispatchMode            GNetDispatchMode
	closeOnce               sync.Once
	connMap                 sync.Map /////ip - 连接映射
	regHandlerMap           sync.Map /////注册处理映射
	rawHandler              GNetRawHandler
	closeTimeout            time.Duration
}

func (ts *GTcpServer) OnBoot(eng gnet.Engine) (action gnet.Action) {
	slog.Info("server init finish !!!!", zap.String("listen addr", ts.protoAddr))
	atomic.StoreInt32(&ts.Stat, 2)
	return
}

func (ts *GTcpServer) OnShutdown(eng gnet.Engine) {
	slog.Info("server shutdown !!!!", zap.String("listen addr", ts.protoAddr))
	atomic.StoreInt32(&ts.Stat, 0)
}

func (ts *GTcpServer) OnOpen(c gnet.Conn) (out []byte, action gnet.Action) {
	slog.Info("new conn ", zap.String("remote addr", c.RemoteAddr().String()), zap.String("local addr", c.LocalAddr().String()))
	gconn := NewGnetConn(c)
	c.SetContext(gconn)
	ts.connMap.Store(c.RemoteAddr().String(), gconn)
	return
}

func (ts *GTcpServer) OnClose(c gnet.Conn, err error) (action gnet.Action) {
	slog.Info("close conn", zap.String("remote addr", c.RemoteAddr().String()), zap.String("local addr", c.LocalAddr().String()),
		zap.Error(err))
	if gconn, ok := c.Context().(*GNetConn); ok {
		gconn.markClosed()
	}
	ts.connMap.Delete(c.RemoteAddr().String())
	return
}

func pingHandler(gnc *GNetConn, rq_dp *DataProtocol) {
	var rs_dp DataProtocol
	micro_time := uint64(time.Now().UnixNano() / 1000)
	rs_dp.Head.PackId = 1001
	rs_dp.Head.HeadUuid = micro_time
	rs_dp.Head.RouteId = micro_time
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
	rs_bytes, err := Encode(&rs_dp)
	if err != nil {
		slog.Error("rs_dp 封包失败", zap.Error(err))
		return
	}
	if err := gnc.Send(rs_bytes); err != nil {
		slog.Error("send pong failed", zap.Error(err))
	}
	return
}

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

func (ts *GTcpServer) handleServerPacket(gconn *GNetConn, dp *DataProtocol) {
	if dp.Head.PackId == PING {
		pingHandler(gconn, dp)
		return
	}
	if ts.rawHandler != nil {
		ts.rawHandler(gconn, dp)
		return
	}
	v, ok := ts.regHandlerMap.Load(dp.Head.PackId)
	if !ok {
		var count uint8
		ts.regHandlerMap.Range(func(k, v interface{}) bool {
			count++
			return true
		})
		slog.Error("未识别的包ID", zap.String("remote addr", gconn.RemoteAddr),
			zap.Uint32("packid", dp.Head.PackId), zap.Uint8("count", count))
		return
	}
	handleST := v.(*HandlerFuncST)
	var rsDP DataProtocol
	microTime := uint64(time.Now().UnixNano() / 1000)
	rsDP.Head.PackId = handleST.RSPackId
	rsDP.Head.HeadUuid = microTime
	rsDP.Head.RouteId = dp.Head.RouteId
	rq := newProtoFromType(handleST.RQType)
	rs := newProtoFromType(handleST.RSType)
	if err := proto.Unmarshal(dp.Data, rq); err != nil {
		slog.Error("proto.Unmarshal 失败", zap.Error(err))
		return
	}
	handleST.HandleFunc(gconn, dp.Head.RouteId, rq, rs)
	bs, err := proto.Marshal(rs)
	if err != nil {
		slog.Error("proto.Marshal 失败", zap.Error(err))
		return
	}
	rsDP.Data = bs
	rsBytes, err := Encode(&rsDP)
	if err != nil {
		slog.Error("rs_dp 封包失败", zap.Error(err))
		return
	}
	if err := gconn.Send(rsBytes); err != nil {
		slog.Error("server send response failed", zap.Error(err))
	}
}

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
		if ts.pool != nil && !ts.pool.StopAndDrain(timeout) {
			slog.Warn("gnet server pool drain timeout")
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		if err := gnet.Stop(ctx, ts.protoAddr); err != nil {
			slog.Error("gnet server stop failed", zap.String("addr", ts.protoAddr), zap.Error(err))
		}
		cancel()
	})
}

func (ts *GTcpServer) State() int32 {
	if ts == nil {
		return 0
	}
	return atomic.LoadInt32(&ts.Stat)
}

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

func (ts *GTcpServer) RegisterHandler(a_rq_id uint32, a_rq proto.Message, a_rs_id uint32, a_rs proto.Message, a_hndle HandleFuncType) {
	rqType, err := newProtoType(a_rq)
	if err != nil {
		slog.Error("new rq proto factory failed", zap.Error(err))
		return
	}
	rsType, err := newProtoType(a_rs)
	if err != nil {
		slog.Error("new rs proto factory failed", zap.Error(err))
		return
	}
	st := &HandlerFuncST{RQ: a_rq, RQPackId: a_rq_id, RS: a_rs, RSPackId: a_rs_id, HandleFunc: a_hndle, RQType: rqType, RSType: rsType}
	ts.regHandlerMap.Store(a_rq_id, st)
}

func (ts *GTcpServer) Run() {
	paddr := ts.protoAddr
	err := gnet.Run(ts, paddr, gnet.WithMulticore(true))
	if err != nil {
		slog.Error("create server failed", zap.String("addr: ", paddr), zap.Error(err))
		return
	}
}

func CreateGNetServer(a_addr string) *GTcpServer {
	ts := &GTcpServer{dispatchMode: GNetDispatchPool, Addr: a_addr, protoAddr: "tcp://:" + a_addr, Stat: 1, closeTimeout: DEFAULT_CLOSE_TIMEOUT}
	ts.pool = su_util.NewGoPool(DEFAULT_POOL_WORKERS, DEFAULT_POOL_QUEUE_SIZE)
	return ts
}

func CreateServer(a_addr string) *GTcpServer {
	return CreateGNetServer(a_addr)
}

func CreateGNetRawServer(a_addr string, handler GNetRawHandler) *GTcpServer {
	ts := CreateGNetServer(a_addr)
	ts.rawHandler = handler
	return ts
}

func (ts *GTcpServer) SetRawHandler(handler GNetRawHandler) {
	if ts == nil {
		return
	}
	ts.rawHandler = handler
}

func (ts *GTcpServer) SetDispatchMode(mode GNetDispatchMode) {
	if ts == nil {
		return
	}
	ts.dispatchMode = mode
}

func (ts *GTcpServer) SetCloseTimeout(timeout time.Duration) {
	if ts == nil {
		return
	}
	ts.closeTimeout = timeout
}
