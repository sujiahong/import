package su_net

import (
	"flag"
	slog "go/su_log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/panjf2000/gnet"
	"github.com/panjf2000/gnet/pkg/pool/goroutine"
	"github.com/panjf2000/gnet/pool/goroutine"
	"go.uber.org/zap"
)

type RecvHandler func() error

type GTcpServer struct{
	*gnet.EventServer         ////匿名字段   事件服务
	pool *goroutine.Pool        ///协程池
	Addr    string            ////监听地址
	async   bool
	multicore bool
	connMap  sync.Map         /////ip - 连接映射
}

func (ts *GTcpServer)OnInitComplete(server gnet.Server)(action gnet.Action){
	slog.Info("server init finish !!!!", zap.Bool("multicore", server.Multicore), 
		zap.String("listen addr", server.Addr.String()), zap.Int("loops", server.NumEventLoop))
	return
}

func (ts *GTcpServer)OnOpened(c gnet.Conn)(out []byte, action gnet.Action){
	slog.Info("new conn ", zap.String("remote addr", c.RemoteAddr().String()))
	ts.connMap.Store(c.RemoteAddr().String(), c)
	return
}

func (ts *GTcpServer)OnClosed(c gnet.Conn, err error)(action gnet.Action){
	slog.Info("close conn", zap.String("remote addr", c.RemoteAddr().String()))
	ts.connMap.Delete(c.RemoteAddr().String())
	return
}

func (ts *GTcpServer)React(frame []byte, c gnet.Conn)(out []byte, action gnet.Action){
	slog.Info("server recv data", zap.String("remote addr", c.RemoteAddr().String()), zap.String("data:", string(frame)))
	if (ts.async) {
		data := append([]byte{}, frame...)
		
		_ = ts.pool.Submit(func(){
			//time.Sleep(1 * time.Second)
			c.AsyncWrite(data)
		})
		return
	} else {
		out = frame
	}
	return
}

func (ts *GTcpServer)Close(){

}


func CreateServer(a_addr string) *GTcpServer{
	ts := &GTcpServer{async: true, multicore: true, Addr: a_addr}
	paddr := "tcp://:"+a_addr
	err := gnet.Serve(ts, paddr, gnet.WithMulticore(true))
	if err != nil {
		slog.Error("create server failed", zap.String("addr: ", paddr))
		return nil
	}
	return ts
}

///////////////////////////////////客户端///////////////////////////////////////

type GTcpClient struct{
	*gnet.EventServer            ////匿名字段   事件服务
	*gnet.Client                 //// 客户端
	remote_addr string           ////远端地址
	conn_pool  []*suNetConn       ////连接池
	mtx       sync.Mutex          ///同步锁
}

func (tc *GTcpClient)OnOpened(c gnet.Conn)(out []byte, action gnet.Action){
	slog.Info("client new conn ", zap.String("remote addr", c.RemoteAddr().String()))
	return
}

func (tc *GTcpClient)OnClosed(c gnet.Conn, err error)(action gnet.Action){
	slog.Info("client close conn", zap.String("remote addr", c.RemoteAddr().String()), zap.String("err:", err.Error()),
		zap.String("local addr", c.LocalAddr().String()))
	j := 0
	for _, val := range tc.conn_pool {
		slog.Info("client close conn 打印", zap.String("remote addr", val.Conn.RemoteAddr().String()), zap.String("local addr", val.Conn.LocalAddr().String()))
		if val.Conn.LocalAddr().String() != c.LocalAddr().String() {
			tc.conn_pool[j] = val
			j++
		}
	}
	return
}

func (tc *GTcpClient)React(frame []byte, c gnet.Conn)(out []byte, action gnet.Action){
	slog.Info("client recv data", zap.String("data: ", string(frame)))
	return
}

func (tc *GTcpClient)Send(a_msg []byte) (err error) {
	var c_i int = -1
	tc.mtx.Lock()
	for i, val := range tc.conn_pool {
		if val.state == 0 {////可用
			c_i = i
			val.state = 1
			break
		}
	}
	tc.mtx.Unlock()
	slog.Info("client send data", zap.Int("c_i: ", c_i), zap.Int("data len:", len(a_msg)))
	if c_i > 0 {
		err = tc.conn_pool[c_i].Conn.AsyncWrite(a_msg)
	}
	atomic.AddInt32(&tc.conn_pool[c_i].state, 0)
	return
}

func (tc *GTcpClient)Stop()(err error){
	err = tc.Client.Stop()
	slog.Info("client stop ", zap.String("err:", err.Error()))/////关闭客户端
	return
}

func CreateClient(a_addr string, a_conn_num uint8) *GTcpClient{
	var port int
	flag.IntVar(&port, "port", 9000, "server port")
	tc := &GTcpClient{remote_addr: a_addr, conn_pool: make([]*suNetConn, 0)}

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
		tc.conn_pool = append(tc.conn_pool, &suNetConn{Conn: conn, state: 0})
	}
	tc.Client = client
	time.AfterFunc(10 * time.Second, func() {
		tc.Send([]byte("hello"))
	})
	return tc
}