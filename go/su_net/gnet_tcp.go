package su_net

import (
	"flag"
	slog "go/su_log"
	"sync"
	"sync/atomic"
	"time"
	"context"
	"go/my_util"

	"github.com/panjf2000/gnet"
	"go.uber.org/zap"
)

const (
	PING_PONG_INTERVAL uint32 = 19
	RECONNECT_INTERVAL uint32 = 5
)

type GTcpServer struct{
	*gnet.EventServer         ////匿名字段   事件服务
	pool *my_util.GoPool      ///协程池
	Stat    int32             /// 服务状态 0 停止 1 初始化 2 启动
	Addr    string            ////监听地址
	async   bool              // 是否异步处理
	multicore bool
	connMap  sync.Map         /////ip - 连接映射
	regHandlerMap sync.Map    /////注册处理映射 
}

func (ts *GTcpServer)OnInitComplete(server gnet.Server)(action gnet.Action){
	slog.Info("server init finish !!!!", zap.Bool("multicore", server.Multicore), 
		zap.String("listen addr", server.Addr.String()), zap.Int("loops", server.NumEventLoop))
	atomic.StoreInt32(&ts.Stat, 2)
	return
}

func (ts *GTcpServer)OnShutdown(server gnet.Server){
	slog.Info("server shutdown !!!!", zap.Bool("multicore", server.Multicore))
	atomic.StoreInt32(&ts.Stat, 0)
}

func (ts *GTcpServer)OnOpened(c gnet.Conn)(out []byte, action gnet.Action){
	slog.Info("new conn ", zap.String("remote addr", c.RemoteAddr().String()), zap.String("local addr", c.LocalAddr().String()))
	gconn := NewGnetConn(c)
	ts.connMap.Store(c.RemoteAddr().String(), gconn)
	return
}

func (ts *GTcpServer)OnClosed(c gnet.Conn, err error)(action gnet.Action){
	slog.Info("close conn", zap.String("remote addr", c.RemoteAddr().String()), zap.String("local addr", c.LocalAddr().String()),
		zap.Error(err))
	if err != nil {
		return
	}
	ts.connMap.Delete(c.RemoteAddr().String())
	return
}

func pingHandler(gnc *GNetConn, rq_dp *DataProtocol){
	var rs_dp DataProtocol
	nano_time := uint64(time.Now().UnixNano())
	rs_dp.Head.PackId = 1001
	rs_dp.Head.HeadUuid = nano_time
	rs_dp.Head.RouteId = nano_time
	ping, err := PingDecode(rq_dp.Data, rq_dp.Head.PackLen)
	if err != nil {
		slog.Error("Ping 解包失败", zap.Error(err))
		return
	}
	slog.Info("Ping心跳", zap.String("remote addr", gnc.Gconn.RemoteAddr().String()),
		zap.Uint64("ping time", ping.SendTime))
	pong := Pong{SendTime: nano_time, PingTime: ping.SendTime}
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

func (ts *GTcpServer)React(frame []byte, c gnet.Conn)(out []byte, action gnet.Action){
	slog.Info("conn recv data", zap.String("remote addr", c.RemoteAddr().String()), zap.String("data:", string(frame)))
	val, ok := ts.connMap.Load(c.RemoteAddr().String())
	if ok {
		gconn := val.(*GNetConn)
		gconn.Recv(frame, func(dp *DataProtocol){
			if (ts.async) {
				ts.pool.SendTask(dp.Head.RouteId, func(){
					if dp.Head.PackId == 1000 {////处理心跳
						pingHandler(gconn, dp)
					}else{
						_, ok := ts.regHandlerMap.Load(dp.Head.PackId)
						if ok {
							slog.Info("包处理", zap.String("remote addr", c.RemoteAddr().String()),
								zap.Uint32("packid",dp.Head.PackId))
							
						}else {
							slog.Error("未识别的包ID", zap.String("remote addr", c.RemoteAddr().String()), 
								zap.Uint32("packid",dp.Head.PackId))
						}
					}
				})
			} else {
				out = frame
			}
		})
	} else {
		slog.Error("未保存的链接", zap.String("remote addr", c.RemoteAddr().String()))
	}
	return
}

func (ts *GTcpServer)Close(){
	atomic.StoreInt32(&ts.Stat, 0)
	ctx, _ := context.WithCancel(context.Background())
	gnet.Stop(ctx, ts.Addr)
	ts.pool.Stop()
}

func (ts *GTcpServer)RegisterHandler() {

}

func CreateServer(a_addr string) *GTcpServer{
	ts := &GTcpServer{async: true, multicore: true, Addr: a_addr, Stat: 1}
	ts.pool = my_util.NewGoPool(16, 1024)
	paddr := "tcp://:"+a_addr
	err := gnet.Serve(ts, paddr, gnet.WithMulticore(true))
	if err != nil {
		slog.Error("create server failed", zap.String("addr: ", paddr), zap.Error(err))
		return nil
	}
	return ts
}

///////////////////////////////////客户端///////////////////////////////////////

type GTcpClient struct{
	*gnet.EventServer            ////匿名字段   事件服务
	*gnet.Client                 //// 客户端
	remote_addr string           ////远端连接地址
	cfgConnNum uint8             //// 配置连接数量
	state      int32			/// 客户端状态 0 停止 1 连接中 2 已连接
	reconnectState int32         /// 重连状态  0 停用  1 启用
	connMap  sync.Map           /////ip - 连接映射
	regHandlerMap sync.Map      /////注册处理映射 
}

func (tc *GTcpClient)OnOpened(c gnet.Conn)(out []byte, action gnet.Action){
	slog.Info("client new conn ", zap.String("remote addr", c.RemoteAddr().String()), 
		zap.String("local addr", c.LocalAddr().String()))
	gconn := NewGnetConn(c)
	tc.connMap.Store(c.LocalAddr().String(), gconn)
	atomic.StoreInt32(&tc.state, 2)
	atomic.StoreInt32(&tc.reconnectState, 0)
	gconn.Ping()
	return
}

func (tc *GTcpClient)OnClosed(c gnet.Conn, err error)(action gnet.Action){
	slog.Info("client close conn", zap.String("remote addr", c.RemoteAddr().String()), zap.Error(err),
		zap.String("local addr", c.LocalAddr().String()))
	if err != nil {
		return
	}
	tc.connMap.Delete(c.LocalAddr().String())
	tc.Reconnect()
	return
}

func (tc *GTcpClient)Tick()(delay time.Duration, action gnet.Action){
	slog.Info("client tick, 发送 心跳", zap.Int32("tc.reconnectState", atomic.LoadInt32(&tc.reconnectState)), zap.Int32("tc.state", atomic.LoadInt32(&tc.state)))
	delay = time.Duration(PING_PONG_INTERVAL)*time.Second
	var count uint8 = 0
	tc.connMap.Range(func(k, v interface{})bool{
		key_str := k.(string)
		gconn := v.(*GNetConn)
		slog.Info("定时连接检查", zap.String("key_str: ", key_str))
		gconn.CheckPong()
		count++
		return true
	})
	if count != tc.cfgConnNum {
		slog.Error("现有连接数量!=配置连接数量", zap.Uint8("count", count), zap.Uint8("tc.cfgConnNum", tc.cfgConnNum))
		if atomic.LoadInt32(&tc.reconnectState) == 0 && atomic.LoadInt32(&tc.state) == 2{
			if tc.cfgConnNum > count {
				n := tc.cfgConnNum - count
				var i uint8
				for i = 0; i < n; i++{
					tc.Connect()
				}
			}
		}
	}
	return
}

func pongHandler(gnc *GNetConn, rs_dp *DataProtocol){
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
	}else {
		slog.Error("PingPongMap没有 PingTime key", zap.Uint64("ping time", pong.PingTime))
	}
}

func (tc *GTcpClient)React(frame []byte, c gnet.Conn)(out []byte, action gnet.Action){
	slog.Info("client recv data", zap.String("data: ", string(frame)))
	val, ok := tc.connMap.Load(c.LocalAddr().String())
	if ok {
		gconn := val.(*GNetConn)
		gconn.Recv(frame, func(dp *DataProtocol){
			if dp.Head.PackId == 1001 {////处理pong心跳
				pongHandler(gconn, dp)
			}else{
				_, ok := tc.regHandlerMap.Load(dp.Head.PackId)
				if ok {
					slog.Info("包处理", zap.String("remote addr", c.RemoteAddr().String()),
						zap.Uint32("packid",dp.Head.PackId))
					
				}else {
					slog.Error("未识别的包ID", zap.String("remote addr", c.RemoteAddr().String()), 
						zap.Uint32("packid",dp.Head.PackId))
				}
			}
		})
	}
	return
}

func (tc *GTcpClient)Send(a_msg []byte) (err error) {
	tc.connMap.Range(func(k, v interface{})bool{
		key_str := k.(string)
		gconn := v.(*GNetConn)
		slog.Info("client send data", zap.String("key_str: ", key_str), zap.Int("data len:", len(a_msg)))
		gconn.Gconn.AsyncWrite(a_msg)
		//atomic.AddInt32(&gconn.state, 0)
		return false
	})
	return
}

func (tc *GTcpClient)Stop()(err error){
	err = tc.Client.Stop()
	slog.Info("client stop ", zap.String("err:", err.Error()))/////关闭客户端
	atomic.StoreInt32(&tc.state, 0)
	return
}

func (tc *GTcpClient)Connect() error{
	conn, err := tc.Client.Dial("tcp", tc.remote_addr)
	if err != nil {
		slog.Error("client dial failed", zap.String("addr: ", tc.remote_addr), zap.Error(err))
		return err
	}
	slog.Info("client connect", zap.String("remote addr:", conn.RemoteAddr().String()),
		zap.String("local addr:", conn.LocalAddr().String()))
	return nil
}

func (tc *GTcpClient)Reconnect(){
	atomic.StoreInt32(&tc.reconnectState, 1)
	my_util.DelayRun(RECONNECT_INTERVAL*1000, func(){
		err := tc.Connect()
		if err != nil{
			tc.Reconnect()
		}
	})
}

func (tc *GTcpClient)RegisterPack(a_packid uint32) {

}

func CreateClient(a_addr string, a_conn_num uint8) *GTcpClient{
	var port int
	flag.IntVar(&port, "port", 9990, "server port")
	tc := &GTcpClient{remote_addr: a_addr, state: 0, cfgConnNum: a_conn_num}

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
	for i = 0; i < a_conn_num; i++{
		tc.Connect()
	}
	return tc
}