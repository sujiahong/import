package su_net

import (
	"flag"
	slog "go/su_log"
	"sync"
	"sync/atomic"
	"time"
	"go/my_util"

	"github.com/panjf2000/gnet"
	"go.uber.org/zap"
)

type RecvHandler func() error

type GTcpServer struct{
	*gnet.EventServer         ////匿名字段   事件服务
	pool *my_util.GoPool      ///协程池
	Stat    int32             /// 服务状态 0 停止 1 初始化 2 启动
	Addr    string            ////监听地址
	async   bool              // 是否异步处理
	multicore bool
	connMap  sync.Map         /////ip - 连接映射
	handlerMap sync.Map
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
	gconn := &GNetConn{Gconn: c, RecvData: make([]byte, 0, 8192)}
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

func (ts *GTcpServer)React(frame []byte, c gnet.Conn)(out []byte, action gnet.Action){
	slog.Info("conn recv data", zap.String("remote addr", c.RemoteAddr().String()), zap.String("data:", string(frame)))
	val, ok := ts.connMap.Load(c.RemoteAddr().String())
	if ok {
		gconn := val.(*GNetConn)
		gconn.RecvData = append(gconn.RecvData, frame...)
		var dp DataProtocol
		var ret_err error
		gconn.RecvData, dp, ret_err = Decode(gconn.RecvData)
		rs_dp := dp
		if (ts.async) {
			ts.pool.SendTask(dp.Head.RouteId, func(){
				if dp.Head.PackId == 1000 {////处理心跳
					rs_dp.Head.PackId = 1001
					nano_time := uint64(time.Now().UnixNano())
					rs_dp.Head.HeadUuid = nano_time
					ping, err := PingDecode(rs_dp.Data, rs_dp.Head.PackLen)
					if err != nil {
						return
					}
					slog.Info("心跳", zap.String("remote addr", c.RemoteAddr().String()),
						zap.Uint64("ping time", ping.Send_time))
					pong := Pong{Send_time: nano_time}
					rs_dp.Data, ret_err = PongEncode(pong)
					if ret_err != nil {
						return
					}
					rs_bytes, err := Encode(rs_dp)
					if err != nil {
						return
					}
					c.AsyncWrite(rs_bytes)
				}else{
					_, ok := ts.handlerMap.Load(dp.Head.PackId)
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
	} else {
		slog.Error("未保存的链接", zap.String("remote addr", c.RemoteAddr().String()))
	}
	return
}

func (ts *GTcpServer)Close(){
	atomic.StoreInt32(&ts.Stat, 0)
	//gnet.Stop()
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
	remote_addr string           ////远端地址
	state      int32			/// 客户端状态 0 停止 1 连接中 2 已连接
	connMap  sync.Map         /////ip - 连接映射
}

func (tc *GTcpClient)OnOpened(c gnet.Conn)(out []byte, action gnet.Action){
	slog.Info("client new conn ", zap.String("remote addr", c.RemoteAddr().String()), 
		zap.String("local addr", c.LocalAddr().String()))
	gconn := &GNetConn{Gconn: c, RecvData: make([]byte, 0, 8192)}
	tc.connMap.Store(c.LocalAddr().String(), gconn)
	atomic.StoreInt32(&tc.state, 2)
	return
}

func (tc *GTcpClient)OnClosed(c gnet.Conn, err error)(action gnet.Action){
	slog.Info("client close conn", zap.String("remote addr", c.RemoteAddr().String()), zap.Error(err),
		zap.String("local addr", c.LocalAddr().String()))
	if err != nil {
		return
	}
	tc.connMap.Delete(c.LocalAddr().String())
	return
}

func (tc *GTcpClient)React(frame []byte, c gnet.Conn)(out []byte, action gnet.Action){
	slog.Info("client recv data", zap.String("data: ", string(frame)))
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

func (tc *GTcpClient)RegisterHandler() {

}

func CreateClient(a_addr string, a_conn_num uint8) *GTcpClient{
	var port int
	flag.IntVar(&port, "port", 9990, "server port")
	tc := &GTcpClient{remote_addr: a_addr, state: 0}

	client, err := gnet.NewClient(tc, gnet.WithTCPNoDelay(gnet.TCPDelay), gnet.WithTCPKeepAlive(30*time.Second))
	if err != nil {
		slog.Error("create client failed", zap.String("addr: ", a_addr))
		return nil
	}
	err = client.Start()
	if err != nil {
		slog.Error("client start failed", zap.String("addr: ", a_addr))
		return nil
	}
	var i uint8
	for i = 0; i < a_conn_num; i++{
		conn, err := client.Dial("tcp", a_addr)
		if err != nil {
			slog.Error("client dial failed", zap.String("addr: ", a_addr))
			return nil
		}
		slog.Info("client new conn ", zap.String("remote addr:", conn.RemoteAddr().String()),
			zap.String("local addr:", conn.LocalAddr().String()), zap.Uint8("i=", i))
	}
	tc.Client = client
	my_util.IntervalRun(10000, 0, func(){
		nano_time := uint64(time.Now().UnixNano())
		var rq_dp DataProtocol
		var ret_err error
		rq_dp.Head.PackId = 1000
		rq_dp.Head.RouteId = nano_time
		rq_dp.Head.HeadUuid = nano_time
		ping := Ping{Send_time: nano_time}
		rq_dp.Data, ret_err = PingEncode(ping)
		if ret_err != nil {
			return
		}
		rq_dp.Head.PackLen = uint32(24 + len(rq_dp.Data))
		rq_bytes, err := Encode(rq_dp)
		if err != nil {
			return
		}
		tc.Send(rq_bytes)
	})
	return tc
}