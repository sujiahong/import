package su_net

import (
	"flag"
	slog "go/su_log"
	"sync"
	"time"

	"github.com/panjf2000/gnet"
	"github.com/panjf2000/gnet/pkg/pool/goroutine"
	"github.com/panjf2000/gnet/pool/goroutine"
	"go.uber.org/zap"
)

type gTcpServer struct{
	*gnet.EventServer         ////匿名字段   事件服务
	pool *goroutine.Pool        ///协程池
	async   bool
	connMap  sync.Map         /////ip - 连接映射
}

func (ts *gTcpServer)OnInitComplete(server gnet.Server)(action gnet.Action){
	slog.Info("server init finish !!!!", zap.Bool("multicore", server.Multicore), 
		zap.String("listen addr", server.Addr.String()), zap.Int("loops", server.NumEventLoop))
	return
}

func (ts *gTcpServer)OnOpened(c gnet.Conn)(out []byte, action gnet.Action){
	slog.Info("new conn ", zap.String("remote addr", c.RemoteAddr().String()))
	ts.connMap.Store(c.RemoteAddr().String(), c)
	return
}

func (ts *gTcpServer)OnClosed(c gnet.Conn, err error)(action gnet.Action){
	slog.Info("close conn", zap.String("remote addr", c.RemoteAddr().String()))
	ts.connMap.Delete(c.RemoteAddr().String())
	return
}

func (ts *gTcpServer)React(frame []byte, c gnet.Conn)(out []byte, action gnet.Action){
	data := append([]byte{}, frame...)
	if (ts.async) {
		_ = ts.pool.Submit(func(){
			time.Sleep(1 * time.Second)
			c.AsyncWrite(data)
		})
		return
	} else {
		out = frame
	}
	return
}

func CreateServer(){

}

type gTcpClient struct{
	*gnet.EventServer            ////匿名字段   事件服务
	wg sync.WaitGroup             
}

func (tc * gTcpClient)React(frame []byte, c gnet.Conn)(out []byte, action gnet.Action){
	return
}

func CreateClient(a_addr string, a_conn_num uint8){
	var port int
	flag.IntVar(&port, "port", 9000, "server port")
	tc := &gTcpClient{wg: sync.WaitGroup{}}

	client, err := gnet.NewClient(tc, gnet.WithTCPNoDelay(gnet.TCPDelay))
	if err != nil {
		slog.Error("create client failed", zap.String("addr: ", a_addr))
		return
	}
	err = client.Start()
	if err != nil {
		slog.Error("client start failed", zap.String("addr: ", a_addr))
		return
	}
	conn, err := client.Dial("tcp", a_addr)
	if err != nil {
		slog.Error("client dial failed", zap.String("addr: ", a_addr))
		return
	}
	slog.Info("client new conn ", zap.String("remote addr:", conn.RemoteAddr().String()),
		zap.String("local addr:", conn.LocalAddr().String()))
	
}