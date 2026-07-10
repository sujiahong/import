package su_net

import (
	"context"
	"fmt"
	"go.local/su_errors"
	slog "go.local/su_log"
	"go.local/su_util"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/panjf2000/gnet/v2"
	"go.uber.org/zap"
)

// GTcpClient 基于 gnet 管理多条 TCP 客户端连接、心跳、重连和请求响应映射。
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
	pool                    *su_util.GoPool
	regHandlerMap           sync.Map /////注册处理映射
	rawHandler              GNetRawHandler
	dispatchMode            GNetDispatchMode
	pendingRQMap            sync.Map /////route id - 请求映射
	pendingEnabled          int32
	requestTimeout          time.Duration
	sendSeq                 uint64
	stopOnce                sync.Once
	stopErr                 error
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
	if tc.ConnCount() == 0 {
		tc.clearPendingRequests()
	}
	if atomic.LoadInt32(&tc.state) != 0 {
		tc.Reconnect()
	}
	return
}

// OnTick 是 gnet 定时回调，用于心跳检查、pending 请求清理和连接数补齐。
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

// handleClientPacket 处理客户端收到的 PONG、raw 包或已注册响应包。
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
	rq := newProtoFromType(handleST.RQType)
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

// send 将已编码数据轮询发送到当前可用连接。
func (tc *GTcpClient) send(a_bytes []byte) (err error) {
	tc.connMu.RLock()
	connCount := len(tc.connList)
	if connCount == 0 {
		tc.connMu.RUnlock()
		return su_errors.NewRetryable(su_errors.CodeUnavailable, "no active tcp client connection")
	}
	target := int(atomic.AddUint64(&tc.sendSeq, 1)-1) % connCount
	gconn := tc.connList[target]
	tc.connMu.RUnlock()
	return gconn.Send(a_bytes)
}

// SendError 发送 typed 请求，并在开启 pending 时记录请求用于响应回调。
func (tc *GTcpClient) SendError(a_rq_id, a_rs_id uint32, a_msg proto.Message) error {
	return tc.sendProto(a_rq_id, a_rs_id, a_msg, tc.pendingRequestsEnabled())
}

// SendNoPending 发送 typed 请求，但不记录 pending 请求。
func (tc *GTcpClient) SendNoPending(a_rq_id, a_rs_id uint32, a_msg proto.Message) error {
	return tc.sendProto(a_rq_id, a_rs_id, a_msg, false)
}

// sendProto 将 proto 消息编码成协议包并发送，可选记录 pending 请求。
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
		err := su_errors.New(su_errors.CodeNotFound, fmt.Sprintf("unregistered response packet id %d", a_rs_id))
		slog.Error("发包未识别的包ID", zap.Uint32("packid", a_rs_id))
		return err
	}
}

// SendPacket 编码 DataProtocol 后发送到当前连接。
func (tc *GTcpClient) SendPacket(dp *DataProtocol) error {
	if dp == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "nil data protocol")
	}
	bs, err := Encode(dp)
	if err != nil {
		return err
	}
	return tc.SendBytes(bs)
}

// SendBytes 发送已编码数据，空数据会被忽略。
func (tc *GTcpClient) SendBytes(bs []byte) error {
	if len(bs) == 0 {
		return nil
	}
	return tc.send(bs)
}

// SendContext 发送 typed 请求；遇到 retryable 错误时会尝试重连并在 context 内重试一次。
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
	if su_errors.Retryable(err) {
		tc.Reconnect()
		if waitErr := tc.waitForConnection(ctx); waitErr != nil {
			return waitErr
		}
		err = tc.SendError(a_rq_id, a_rs_id, a_msg)
		if err == nil {
			return nil
		}
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return err
	}
}

// waitForConnection 等待客户端恢复到至少一条可用连接。
func (tc *GTcpClient) waitForConnection(ctx context.Context) error {
	if tc == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "gnet client is nil")
	}
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		if atomic.LoadInt32(&tc.state) == 0 {
			return su_errors.New(su_errors.CodeUnavailable, "client stopped")
		}
		if tc.ConnCount() > 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// Send 兼容旧接口发送 typed 请求，错误只记录日志。
func (tc *GTcpClient) Send(a_rq_id, a_rs_id uint32, a_msg proto.Message) {
	if err := tc.SendError(a_rq_id, a_rs_id, a_msg); err != nil {
		slog.Error("gnet client send failed", zap.Error(err))
	}
}

// cleanupExpiredPendingRequests 删除超过请求超时时间的 pending 请求。
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
	expiredKeys := make([]interface{}, 0)
	tc.pendingRQMap.Range(func(k, v interface{}) bool {
		pending, ok := v.(*pendingGNetRequest)
		if !ok || now.Sub(pending.createdAt) >= timeout {
			expiredKeys = append(expiredKeys, k)
		}
		return true
	})
	for _, key := range expiredKeys {
		tc.pendingRQMap.Delete(key)
		slog.Warn("gnet pending request expired", zap.Any("route_id", key))
	}
}

// clearPendingRequests 清空所有等待响应的请求。
func (tc *GTcpClient) clearPendingRequests() {
	if tc == nil {
		return
	}
	deleteAllSyncMap(&tc.pendingRQMap)
}

// pendingRequestsEnabled 返回是否启用 pending 请求记录。
func (tc *GTcpClient) pendingRequestsEnabled() bool {
	return tc != nil && atomic.LoadInt32(&tc.pendingEnabled) == 1
}

// SetPendingRequestsEnabled 开关 pending 请求记录；关闭时会清空现有 pending。
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

// Stop 停止 gnet client、清理 pending 请求并关闭 worker 池。
func (tc *GTcpClient) Stop() (err error) {
	if tc == nil {
		return nil
	}
	tc.stopOnce.Do(func() {
		atomic.StoreInt32(&tc.state, 0)
		tc.clearPendingRequests()
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

// RegisterHandler 注册请求/响应包 ID、proto 模板和处理函数。
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

// CreateGNetClient 创建 gnet TCP client 并建立指定数量连接。
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

// CreateClient 是 CreateGNetClient 的兼容别名。
func CreateClient(a_addr string, a_conn_num uint8) *GTcpClient {
	return CreateGNetClient(a_addr, a_conn_num)
}

// CreateGNetRawClient 创建 raw 模式 gnet TCP client。
func CreateGNetRawClient(a_addr string, a_conn_num uint8, handler GNetRawHandler) *GTcpClient {
	tc := CreateGNetClient(a_addr, a_conn_num)
	if tc != nil {
		tc.rawHandler = handler
	}
	return tc
}

// SetRawHandler 设置 raw 数据包处理函数。
func (tc *GTcpClient) SetRawHandler(handler GNetRawHandler) {
	if tc == nil {
		return
	}
	tc.rawHandler = handler
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
