package su_net

import (
	"context"
	"fmt"
	"go.local/my_util"
	slog "go.local/su_log"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/panjf2000/gnet"
	"go.uber.org/zap"
)

const (
	PING_PONG_INTERVAL uint32 = 19
	RECONNECT_INTERVAL uint32 = 5
)

type HandleFuncType func(*GNetConn, uint64, proto.Message, proto.Message)

// / 业务处理函数结构
type HandlerFuncST struct {
	RQ         proto.Message
	RQPackId   uint32
	RS         proto.Message
	RSPackId   uint32
	HandleFunc HandleFuncType
	NewRQ      func() proto.Message
	NewRS      func() proto.Message
}

func newProtoFactory(template proto.Message) (func() proto.Message, error) {
	if template == nil {
		return nil, fmt.Errorf("nil proto template")
	}
	t := reflect.TypeOf(template)
	if t.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("proto template must be pointer, got %s", t.Kind())
	}
	if _, ok := reflect.New(t.Elem()).Interface().(proto.Message); !ok {
		return nil, fmt.Errorf("%s does not implement proto.Message", t.String())
	}
	return func() proto.Message {
		msg, _ := reflect.New(t.Elem()).Interface().(proto.Message)
		return msg
	}, nil
}

func newProtoMessage(template proto.Message) (proto.Message, error) {
	factory, err := newProtoFactory(template)
	if err != nil {
		return nil, err
	}
	return factory(), nil
}

type GTcpServer struct {
	*gnet.EventServer                 ////匿名字段   事件服务
	pool              *my_util.GoPool ///协程池
	Stat              int32           /// 服务状态 0 停止 1 初始化 2 启动
	Addr              string          ////监听地址
	async             bool            // 是否异步处理
	multicore         bool
	connMap           sync.Map /////ip - 连接映射
	regHandlerMap     sync.Map /////注册处理映射
}

func (ts *GTcpServer) OnInitComplete(server gnet.Server) (action gnet.Action) {
	slog.Info("server init finish !!!!", zap.Bool("multicore", server.Multicore),
		zap.String("listen addr", server.Addr.String()), zap.Int("loops", server.NumEventLoop))
	atomic.StoreInt32(&ts.Stat, 2)
	return
}

func (ts *GTcpServer) OnShutdown(server gnet.Server) {
	slog.Info("server shutdown !!!!", zap.Bool("multicore", server.Multicore))
	atomic.StoreInt32(&ts.Stat, 0)
}

func (ts *GTcpServer) OnOpened(c gnet.Conn) (out []byte, action gnet.Action) {
	slog.Info("new conn ", zap.String("remote addr", c.RemoteAddr().String()), zap.String("local addr", c.LocalAddr().String()))
	gconn := NewGnetConn(c)
	ts.connMap.Store(c.RemoteAddr().String(), gconn)
	return
}

func (ts *GTcpServer) OnClosed(c gnet.Conn, err error) (action gnet.Action) {
	slog.Info("close conn", zap.String("remote addr", c.RemoteAddr().String()), zap.String("local addr", c.LocalAddr().String()),
		zap.Error(err))
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
	slog.Info("Ping心跳", zap.String("remote addr", gnc.Gconn.RemoteAddr().String()),
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
	gnc.Send(rs_bytes)
	return
}

func (ts *GTcpServer) React(frame []byte, c gnet.Conn) (out []byte, action gnet.Action) {
	//slog.Info("server recv data", zap.String("remote addr", c.RemoteAddr().String()), zap.String("data:", string(frame)), zap.Int("data len:", len(frame)))
	val, ok := ts.connMap.Load(c.RemoteAddr().String())
	if ok {
		gconn := val.(*GNetConn)
		gconn.Recv(frame, func(dp *DataProtocol) {
			if ts.async {
				taskDP := *dp
				ts.pool.SendTask(taskDP.Head.RouteId, func() {
					if taskDP.Head.PackId == 1000 { ////处理心跳
						pingHandler(gconn, &taskDP)
					} else {
						v, ok := ts.regHandlerMap.Load(taskDP.Head.PackId)
						if ok {
							handleST := v.(*HandlerFuncST)
							var rs_dp DataProtocol
							micro_time := uint64(time.Now().UnixNano() / 1000)
							rs_dp.Head.PackId = handleST.RSPackId
							rs_dp.Head.HeadUuid = micro_time
							rs_dp.Head.RouteId = taskDP.Head.RouteId
							rq := handleST.NewRQ()
							rs := handleST.NewRS()
							err := proto.Unmarshal(taskDP.Data, rq)
							if err != nil {
								slog.Error("proto.Unmarshal 失败", zap.Error(err))
								return
							}
							handleST.HandleFunc(gconn, taskDP.Head.RouteId, rq, rs)
							bs, err := proto.Marshal(rs)
							if err != nil {
								slog.Error("proto.Marshal 失败", zap.Error(err))
								return
							}
							rs_dp.Data = bs
							rs_dp.Head.PackLen = uint32(24 + len(rs_dp.Data))
							rs_bytes, err := Encode(&rs_dp)
							if err != nil {
								slog.Error("rs_dp 封包失败", zap.Error(err))
								return
							}
							gconn.Send(rs_bytes)
						} else {
							var count uint8 = 0
							ts.regHandlerMap.Range(func(k, v interface{}) bool {
								count++
								return true
							})
							slog.Error("未识别的包ID", zap.String("remote addr", c.RemoteAddr().String()),
								zap.Uint32("packid", taskDP.Head.PackId), zap.Uint8("count", count))
						}
					}
				})
			} else {
				//暂未实现
				out = frame
			}
		})
	} else {
		slog.Error("未保存的链接", zap.String("remote addr", c.RemoteAddr().String()))
	}
	return
}

func (ts *GTcpServer) Close() {
	atomic.StoreInt32(&ts.Stat, 0)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	gnet.Stop(ctx, "tcp://:"+ts.Addr)
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
	newRQ, err := newProtoFactory(a_rq)
	if err != nil {
		slog.Error("new rq proto factory failed", zap.Error(err))
		return
	}
	newRS, err := newProtoFactory(a_rs)
	if err != nil {
		slog.Error("new rs proto factory failed", zap.Error(err))
		return
	}
	st := &HandlerFuncST{RQ: a_rq, RQPackId: a_rq_id, RS: a_rs, RSPackId: a_rs_id, HandleFunc: a_hndle, NewRQ: newRQ, NewRS: newRS}
	ts.regHandlerMap.Store(a_rq_id, st)
}

func (ts *GTcpServer) Run() {
	paddr := "tcp://:" + ts.Addr
	err := gnet.Serve(ts, paddr, gnet.WithMulticore(true))
	if err != nil {
		slog.Error("create server failed", zap.String("addr: ", paddr), zap.Error(err))
		return
	}
}

func CreateGNetServer(a_addr string) *GTcpServer {
	ts := &GTcpServer{async: true, multicore: true, Addr: a_addr, Stat: 1}
	ts.pool = my_util.NewGoPool(16, 1024)
	return ts
}

func CreateServer(a_addr string) *GTcpServer {
	return CreateGNetServer(a_addr)
}

///////////////////////////////////客户端///////////////////////////////////////

type GTcpClient struct {
	*gnet.EventServer          ////匿名字段   事件服务
	*gnet.Client               //// 客户端
	remote_addr       string   ////远端连接地址
	cfgConnNum        uint8    //// 配置连接数量
	state             int32    /// 客户端状态 0 停止 1 连接中 2 已连接
	reconnectState    int32    /// 重连状态  0 停用  1 启用
	connMap           sync.Map /////ip - 连接映射
	connMu            sync.RWMutex
	connList          []*GNetConn
	regHandlerMap     sync.Map /////注册处理映射
	pendingRQMap      sync.Map /////route id - 请求映射
	sendSeq           uint64
}

func (tc *GTcpClient) OnOpened(c gnet.Conn) (out []byte, action gnet.Action) {
	slog.Info("client new conn ", zap.String("remote addr", c.RemoteAddr().String()),
		zap.String("local addr", c.LocalAddr().String()))
	gconn := NewGnetConn(c)
	tc.connMap.Store(c.LocalAddr().String(), gconn)
	tc.connMu.Lock()
	tc.connList = append(tc.connList, gconn)
	tc.connMu.Unlock()
	atomic.StoreInt32(&tc.state, 2)
	atomic.StoreInt32(&tc.reconnectState, 0)
	gconn.Ping()
	return
}

func (tc *GTcpClient) OnClosed(c gnet.Conn, err error) (action gnet.Action) {
	slog.Info("client close conn", zap.String("remote addr", c.RemoteAddr().String()), zap.Error(err),
		zap.String("local addr", c.LocalAddr().String()))
	tc.connMap.Delete(c.LocalAddr().String())
	tc.connMu.Lock()
	for i, conn := range tc.connList {
		if conn.Gconn == c {
			tc.connList = append(tc.connList[:i], tc.connList[i+1:]...)
			break
		}
	}
	tc.connMu.Unlock()
	if atomic.LoadInt32(&tc.state) != 0 {
		tc.Reconnect()
	}
	return
}

func (tc *GTcpClient) Tick() (delay time.Duration, action gnet.Action) {
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

func pongHandler(gnc *GNetConn, rs_dp *DataProtocol) {
	pong, err := PongDecode(rs_dp.Data, rs_dp.Head.PackLen)
	if err != nil {
		slog.Error("pong 解包失败", zap.Error(err))
		return
	}
	slog.Info("pong心跳", zap.String("remote addr", gnc.Gconn.RemoteAddr().String()),
		zap.Uint64("pong time", pong.SendTime), zap.Uint64("ping time", pong.PingTime))
	_, ok := gnc.PingPongMap.Load(pong.PingTime)
	if ok {
		gnc.PingPongMap.Delete(pong.PingTime)
	} else {
		slog.Error("PingPongMap没有 PingTime key", zap.Uint64("ping time", pong.PingTime))
	}
}

func (tc *GTcpClient) React(frame []byte, c gnet.Conn) (out []byte, action gnet.Action) {
	val, ok := tc.connMap.Load(c.LocalAddr().String())
	if ok {
		gconn := val.(*GNetConn)
		gconn.Recv(frame, func(dp *DataProtocol) {
			if dp.Head.PackId == 1001 { ////处理pong心跳
				pongHandler(gconn, dp)
			} else {
				v, ok := tc.regHandlerMap.Load(dp.Head.PackId)
				if ok {
					handleST := v.(*HandlerFuncST)
					rs := handleST.NewRS()
					err := proto.Unmarshal(dp.Data, rs)
					if err != nil {
						slog.Error("proto.Unmarshal 失败", zap.Error(err))
						return
					}
					rq := handleST.RQ
					if pendingRQ, ok := tc.pendingRQMap.LoadAndDelete(dp.Head.RouteId); ok {
						if msg, ok := pendingRQ.(proto.Message); ok {
							rq = msg
						}
					}
					handleST.HandleFunc(gconn, dp.Head.RouteId, rq, rs)
				} else {
					slog.Error("未识别的包ID", zap.String("remote addr", c.RemoteAddr().String()),
						zap.Uint32("packid", dp.Head.PackId))
				}
			}
		})
	}
	return
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
	gconn.Send(a_bytes)
	return
}

func (tc *GTcpClient) Send(a_rq_id, a_rs_id uint32, a_msg proto.Message) {
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
			return
		}
		rq_dp.Data = bs
		rq_dp.Head.PackLen = uint32(24 + len(rq_dp.Data))
		rq_bytes, err := Encode(&rq_dp)
		if err != nil {
			slog.Error("rq_dp 封包失败", zap.Error(err))
			return
		}
		tc.pendingRQMap.Store(rq_dp.Head.RouteId, proto.Clone(a_msg))
		if err := tc.send(rq_bytes); err != nil {
			tc.pendingRQMap.Delete(rq_dp.Head.RouteId)
			slog.Error("client send failed", zap.Error(err))
		}
	} else {
		slog.Error("发包未识别的包ID", zap.Uint32("packid", a_rs_id))
	}
	return
}

func (tc *GTcpClient) Stop() (err error) {
	atomic.StoreInt32(&tc.state, 0)
	err = tc.Client.Stop()
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
		return fmt.Errorf("client stopped")
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
	newRQ, err := newProtoFactory(a_rq)
	if err != nil {
		slog.Error("new rq proto factory failed", zap.Error(err))
		return
	}
	newRS, err := newProtoFactory(a_rs)
	if err != nil {
		slog.Error("new rs proto factory failed", zap.Error(err))
		return
	}
	st := &HandlerFuncST{RQ: a_rq, RQPackId: a_rq_id, RS: a_rs, RSPackId: a_rs_id, HandleFunc: a_hndle, NewRQ: newRQ, NewRS: newRS}
	tc.regHandlerMap.Store(a_rs_id, st)
}

func CreateGNetClient(a_addr string, a_conn_num uint8) *GTcpClient {
	tc := &GTcpClient{remote_addr: a_addr, state: 1, cfgConnNum: a_conn_num}

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
