package su_net

import (
	"context"
	"errors"
	"fmt"
	"go.local/my_util"
	slog "go.local/su_log"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/panjf2000/gnet/v2"
	"go.uber.org/zap"
)

const (
	PING_PONG_INTERVAL      uint32 = 19
	RECONNECT_INTERVAL      uint32 = 5
	DEFAULT_REQUEST_TIMEOUT        = 30 * time.Second
)

type HandleFuncType func(*GNetConn, uint64, proto.Message, proto.Message)
type GNetRawHandler func(*GNetConn, *DataProtocol)
type GNetDispatchMode uint8

const (
	GNetDispatchPool GNetDispatchMode = iota
	GNetDispatchInline
)

// / 业务处理函数结构
type HandlerFuncST struct {
	RQ         proto.Message
	RQPackId   uint32
	RS         proto.Message
	RSPackId   uint32
	HandleFunc HandleFuncType
	RQType     reflect.Type
	RSType     reflect.Type
}

type pendingGNetRequest struct {
	rq        proto.Message
	createdAt time.Time
}

func newProtoType(template proto.Message) (reflect.Type, error) {
	if template == nil {
		return nil, fmt.Errorf("nil proto template")
	}
	t := reflect.TypeOf(template)
	if t.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("proto template must be pointer, got %s", t.Kind())
	}
	elem := t.Elem()
	if _, ok := reflect.New(elem).Interface().(proto.Message); !ok {
		return nil, fmt.Errorf("%s does not implement proto.Message", t.String())
	}
	return elem, nil
}

func newProtoFromType(t reflect.Type) proto.Message {
	if t == nil {
		return nil
	}
	msg, _ := reflect.New(t).Interface().(proto.Message)
	return msg
}

func newProtoMessage(template proto.Message) (proto.Message, error) {
	t, err := newProtoType(template)
	if err != nil {
		return nil, err
	}
	return newProtoFromType(t), nil
}

type GTcpServer struct {
	gnet.BuiltinEventEngine                 ////匿名字段   事件服务
	pool                    *my_util.GoPool ///协程池
	Stat                    int32           /// 服务状态 0 停止 1 初始化 2 启动
	Addr                    string          ////监听地址
	protoAddr               string
	async                   bool // 是否异步处理
	multicore               bool
	dispatchMode            GNetDispatchMode
	connMap                 sync.Map /////ip - 连接映射
	regHandlerMap           sync.Map /////注册处理映射
	rawHandler              GNetRawHandler
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
		gconn.ClearHeartbeat()
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
	rs_dp.Head.PackLen = uint32(24 + len(rs_dp.Data))
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
	atomic.StoreInt32(&ts.Stat, 0)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := gnet.Stop(ctx, ts.protoAddr); err != nil {
		slog.Error("gnet server stop failed", zap.String("addr", ts.protoAddr), zap.Error(err))
	}
	if ts.pool != nil {
		ts.pool.Stop()
	}
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
	ts := &GTcpServer{async: true, multicore: true, dispatchMode: GNetDispatchPool, Addr: a_addr, protoAddr: "tcp://:" + a_addr, Stat: 1}
	ts.pool = my_util.NewGoPool(16, 1024)
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

///////////////////////////////////客户端///////////////////////////////////////

type GTcpClient struct {
	gnet.BuiltinEventEngine          ////匿名字段   事件服务
	*gnet.Client                     //// 客户端
	remote_addr             string   ////远端连接地址
	cfgConnNum              uint8    //// 配置连接数量
	state                   int32    /// 客户端状态 0 停止 1 连接中 2 已连接
	reconnectState          int32    /// 重连状态  0 停用  1 启用
	connMap                 sync.Map /////ip - 连接映射
	connMu                  sync.RWMutex
	connList                []*GNetConn
	pool                    *my_util.GoPool
	regHandlerMap           sync.Map /////注册处理映射
	rawHandler              GNetRawHandler
	dispatchMode            GNetDispatchMode
	pendingRQMap            sync.Map /////route id - 请求映射
	pendingEnabled          int32
	requestTimeout          time.Duration
	sendSeq                 uint64
}

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
		closedConn.ClearHeartbeat()
	}
	if tc.ConnCount() == 0 {
		tc.clearPendingRequests()
	}
	if atomic.LoadInt32(&tc.state) != 0 {
		tc.Reconnect()
	}
	return
}

func (tc *GTcpClient) OnTick() (delay time.Duration, action gnet.Action) {
	slog.Info("client tick, 发送 心跳", zap.Int32("tc.reconnectState", atomic.LoadInt32(&tc.reconnectState)), zap.Int32("tc.state", atomic.LoadInt32(&tc.state)))
	delay = time.Duration(PING_PONG_INTERVAL) * time.Second
	tc.cleanupExpiredPendingRequests()
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

func pongHandler(gnc *GNetConn, rs_dp *DataProtocol) {
	pong, err := PongDecode(rs_dp.Data, rs_dp.Head.PackLen)
	if err != nil {
		slog.Error("pong 解包失败", zap.Error(err))
		return
	}
	slog.Info("pong心跳", zap.String("remote addr", gnc.RemoteAddr),
		zap.Uint64("pong time", pong.SendTime), zap.Uint64("ping time", pong.PingTime))
	_, ok := gnc.PingPongMap.Load(pong.PingTime)
	if ok {
		gnc.PingPongMap.Delete(pong.PingTime)
	} else {
		slog.Error("PingPongMap没有 PingTime key", zap.Uint64("ping time", pong.PingTime))
	}
}

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

func (tc *GTcpClient) ensurePool() *my_util.GoPool {
	if tc == nil {
		return nil
	}
	tc.connMu.Lock()
	defer tc.connMu.Unlock()
	if tc.pool == nil {
		tc.pool = my_util.NewGoPool(16, 1024)
	}
	return tc.pool
}

func (tc *GTcpClient) handleClientPacket(gconn *GNetConn, dp *DataProtocol) {
	if dp.Head.PackId == PONG {
		pongHandler(gconn, dp)
		return
	}
	if tc.rawHandler != nil {
		tc.rawHandler(gconn, dp)
		return
	}
	v, ok := tc.regHandlerMap.Load(dp.Head.PackId)
	if !ok {
		slog.Error("未识别的包ID", zap.String("remote addr", gconn.RemoteAddr),
			zap.Uint32("packid", dp.Head.PackId))
		return
	}
	handleST := v.(*HandlerFuncST)
	rs := newProtoFromType(handleST.RSType)
	if err := proto.Unmarshal(dp.Data, rs); err != nil {
		slog.Error("proto.Unmarshal 失败", zap.Error(err))
		return
	}
	rq := handleST.RQ
	if tc.pendingRequestsEnabled() {
		if pendingRQ, ok := tc.pendingRQMap.LoadAndDelete(dp.Head.RouteId); ok {
			switch pending := pendingRQ.(type) {
			case *pendingGNetRequest:
				rq = pending.rq
			case proto.Message:
				rq = pending
			}
		}
	}
	handleST.HandleFunc(gconn, dp.Head.RouteId, rq, rs)
}

func (tc *GTcpClient) send(a_bytes []byte) (err error) {
	tc.connMu.RLock()
	connCount := len(tc.connList)
	if connCount == 0 {
		tc.connMu.RUnlock()
		return fmt.Errorf("no active tcp client connection")
	}
	target := int(atomic.AddUint64(&tc.sendSeq, 1)-1) % connCount
	gconn := tc.connList[target]
	tc.connMu.RUnlock()
	return gconn.Send(a_bytes)
}

func (tc *GTcpClient) SendError(a_rq_id, a_rs_id uint32, a_msg proto.Message) error {
	return tc.sendProto(a_rq_id, a_rs_id, a_msg, tc.pendingRequestsEnabled())
}

func (tc *GTcpClient) SendNoPending(a_rq_id, a_rs_id uint32, a_msg proto.Message) error {
	return tc.sendProto(a_rq_id, a_rs_id, a_msg, false)
}

func (tc *GTcpClient) sendProto(a_rq_id, a_rs_id uint32, a_msg proto.Message, trackPending bool) error {
	_, ok := tc.regHandlerMap.Load(a_rs_id)
	if ok {
		var rq_dp DataProtocol
		routeID := nextRouteID()
		micro_time := uint64(time.Now().UnixNano() / 1000)
		rq_dp.Head.PackId = a_rq_id
		rq_dp.Head.HeadUuid = micro_time
		rq_dp.Head.RouteId = routeID
		bs, err := proto.Marshal(a_msg)
		if err != nil {
			slog.Error("proto.Marshal 失败", zap.Error(err))
			return err
		}
		rq_dp.Data = bs
		rq_dp.Head.PackLen = uint32(24 + len(rq_dp.Data))
		rq_bytes, err := Encode(&rq_dp)
		if err != nil {
			slog.Error("rq_dp 封包失败", zap.Error(err))
			return err
		}
		if trackPending {
			tc.pendingRQMap.Store(rq_dp.Head.RouteId, &pendingGNetRequest{rq: proto.Clone(a_msg), createdAt: time.Now()})
		}
		if err := tc.send(rq_bytes); err != nil {
			if trackPending {
				tc.pendingRQMap.Delete(rq_dp.Head.RouteId)
			}
			slog.Error("client send failed", zap.Error(err))
			return err
		}
		return nil
	} else {
		err := fmt.Errorf("unregistered response packet id %d", a_rs_id)
		slog.Error("发包未识别的包ID", zap.Uint32("packid", a_rs_id))
		return err
	}
}

func (tc *GTcpClient) SendPacket(dp *DataProtocol) error {
	if dp == nil {
		return fmt.Errorf("nil data protocol")
	}
	bs, err := Encode(dp)
	if err != nil {
		return err
	}
	return tc.SendBytes(bs)
}

func (tc *GTcpClient) SendBytes(bs []byte) error {
	if len(bs) == 0 {
		return nil
	}
	return tc.send(bs)
}

func (tc *GTcpClient) SendContext(ctx context.Context, a_rq_id, a_rs_id uint32, a_msg proto.Message) error {
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	err := tc.SendError(a_rq_id, a_rs_id, a_msg)
	if err == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return err
	}
}

func (tc *GTcpClient) Send(a_rq_id, a_rs_id uint32, a_msg proto.Message) {
	if err := tc.SendError(a_rq_id, a_rs_id, a_msg); err != nil {
		slog.Error("gnet client send failed", zap.Error(err))
	}
}

func (tc *GTcpClient) cleanupExpiredPendingRequests() {
	if tc == nil {
		return
	}
	if !tc.pendingRequestsEnabled() {
		return
	}
	timeout := tc.requestTimeout
	if timeout <= 0 {
		timeout = DEFAULT_REQUEST_TIMEOUT
	}
	now := time.Now()
	tc.pendingRQMap.Range(func(k, v interface{}) bool {
		pending, ok := v.(*pendingGNetRequest)
		if !ok || now.Sub(pending.createdAt) >= timeout {
			tc.pendingRQMap.Delete(k)
			slog.Warn("gnet pending request expired", zap.Any("route_id", k))
		}
		return true
	})
}

func (tc *GTcpClient) clearPendingRequests() {
	if tc == nil {
		return
	}
	tc.pendingRQMap.Range(func(k, v interface{}) bool {
		tc.pendingRQMap.Delete(k)
		return true
	})
}

func (tc *GTcpClient) pendingRequestsEnabled() bool {
	return tc != nil && atomic.LoadInt32(&tc.pendingEnabled) == 1
}

func (tc *GTcpClient) SetPendingRequestsEnabled(enabled bool) {
	if tc == nil {
		return
	}
	if enabled {
		atomic.StoreInt32(&tc.pendingEnabled, 1)
		return
	}
	atomic.StoreInt32(&tc.pendingEnabled, 0)
	tc.clearPendingRequests()
}

func (tc *GTcpClient) Stop() (err error) {
	atomic.StoreInt32(&tc.state, 0)
	tc.clearPendingRequests()
	err = tc.Client.Stop()
	if tc.pool != nil {
		tc.pool.Stop()
	}
	slog.Info("client stop ", zap.Error(err)) /////关闭客户端
	return
}

func (tc *GTcpClient) State() int32 {
	if tc == nil {
		return 0
	}
	return atomic.LoadInt32(&tc.state)
}

func (tc *GTcpClient) ConnCount() int {
	if tc == nil {
		return 0
	}
	tc.connMu.RLock()
	defer tc.connMu.RUnlock()
	return len(tc.connList)
}

func (tc *GTcpClient) Connect() error {
	if atomic.LoadInt32(&tc.state) == 0 {
		return errors.New("client stopped")
	}
	atomic.StoreInt32(&tc.state, 1)
	conn, err := tc.Client.Dial("tcp", tc.remote_addr)
	if err != nil {
		slog.Error("client dial failed", zap.String("addr: ", tc.remote_addr), zap.Error(err))
		return err
	}
	slog.Info("client connect", zap.String("remote addr:", conn.RemoteAddr().String()),
		zap.String("local addr:", conn.LocalAddr().String()))
	return nil
}

func (tc *GTcpClient) Reconnect() {
	if atomic.LoadInt32(&tc.state) == 0 {
		return
	}
	if !atomic.CompareAndSwapInt32(&tc.reconnectState, 0, 1) {
		return
	}
	my_util.DelayRun(RECONNECT_INTERVAL*1000, func() {
		err := tc.Connect()
		if err != nil {
			atomic.StoreInt32(&tc.reconnectState, 0)
			tc.Reconnect()
		}
	})
}

func (tc *GTcpClient) RegisterHandler(a_rq_id uint32, a_rq proto.Message, a_rs_id uint32, a_rs proto.Message, a_hndle HandleFuncType) {
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
	tc.regHandlerMap.Store(a_rs_id, st)
}

func CreateGNetClient(a_addr string, a_conn_num uint8) *GTcpClient {
	tc := &GTcpClient{remote_addr: a_addr, state: 1, cfgConnNum: a_conn_num, dispatchMode: GNetDispatchInline, pendingEnabled: 1, requestTimeout: DEFAULT_REQUEST_TIMEOUT}

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

func CreateClient(a_addr string, a_conn_num uint8) *GTcpClient {
	return CreateGNetClient(a_addr, a_conn_num)
}

func CreateGNetRawClient(a_addr string, a_conn_num uint8, handler GNetRawHandler) *GTcpClient {
	tc := CreateGNetClient(a_addr, a_conn_num)
	if tc != nil {
		tc.rawHandler = handler
	}
	return tc
}

func (tc *GTcpClient) SetRawHandler(handler GNetRawHandler) {
	if tc == nil {
		return
	}
	tc.rawHandler = handler
}

func (tc *GTcpClient) SetDispatchMode(mode GNetDispatchMode) {
	if tc == nil {
		return
	}
	tc.dispatchMode = mode
	if mode == GNetDispatchPool {
		tc.ensurePool()
	}
}
