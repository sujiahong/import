package su_net

import (
	"errors"
	"go.local/su_errors"
	slog "go.local/su_log"
	"go.local/su_util"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// TcpServer 监听 TCP 连接，并将业务包分发到 worker 池处理。
type TcpServer struct {
	Addr         string              // 实际监听地址。
	listener     *net.TCPListener    // TCP listener。
	handler      TcpHandler          // 业务包处理函数。
	conns        map[string]*TcpConn // remote addr 到连接的映射。
	connsMu      sync.Mutex          // 保护 conns。
	closeOnce    sync.Once           // 保证 Close 只执行一次。
	pool         *su_util.GoPool     // 业务包处理 worker 池。
	writeTimeout int64               // 写超时，存储为 time.Duration 的 int64。
}

// CreateTcpServer 使用默认配置创建并启动 TCP server。
func CreateTcpServer(addr string, handlers ...TcpHandler) (*TcpServer, error) {
	return CreateTcpServerWithConfig(addr, DefaultTcpNetConfig(), handlers...)
}

// CreateTcpServerWithConfig 使用指定配置创建并启动 TCP server。
func CreateTcpServerWithConfig(addr string, cfg TcpNetConfig, handlers ...TcpHandler) (*TcpServer, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, su_errors.Wrap(su_errors.CodeInvalidArgument, "resolve tcp addr failed", err)
	}
	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return nil, su_errors.WrapRetryable(su_errors.CodeUnavailable, "listen tcp failed", err)
	}
	server := &TcpServer{
		Addr:         listener.Addr().String(),
		listener:     listener,
		conns:        make(map[string]*TcpConn),
		pool:         su_util.NewGoPool(DEFAULT_POOL_WORKERS, DEFAULT_POOL_QUEUE_SIZE),
		writeTimeout: int64(cfg.WriteTimeout),
	}
	if len(handlers) > 0 {
		server.handler = handlers[0]
	}
	go server.acceptLoop()
	return server, nil
}

// acceptLoop 接受新 TCP 连接，维护连接表并启动每条连接的读循环。
func (ts *TcpServer) acceptLoop() {
	for {
		conn, err := ts.listener.AcceptTCP()
		if err != nil {
			if !errors.Is(err, net.ErrClosed) {
				slog.Error("tcp accept failed", zap.Error(err))
			}
			return
		}
		tcpConn := newTcpConnWithWriteTimeout(conn, ts.WriteTimeout())
		key := conn.RemoteAddr().String()
		ts.connsMu.Lock()
		tcpConn.SetWriteTimeout(ts.WriteTimeout())
		ts.conns[key] = tcpConn
		ts.connsMu.Unlock()
		go func() {
			defer func() {
				ts.connsMu.Lock()
				if ts.conns[key] == tcpConn {
					delete(ts.conns, key)
				}
				ts.connsMu.Unlock()
			}()
			tcpConn.readLoop(func(conn *TcpConn, dp *DataProtocol) {
				if ts.handler == nil {
					return
				}
				taskDP := *dp
				if !ts.pool.SendTask(taskDP.Head.RouteId, func() {
					ts.handler(conn, &taskDP)
				}) {
					slog.Warn("tcp server task dropped", zap.Uint64("route_id", taskDP.Head.RouteId))
				}
			})
		}()
	}
}

// Close 关闭监听、所有连接和 worker 池。
func (ts *TcpServer) Close() error {
	if ts == nil || ts.listener == nil {
		return nil
	}
	var err error
	ts.closeOnce.Do(func() {
		err = ts.listener.Close()
		conns := make([]*TcpConn, 0)
		ts.connsMu.Lock()
		for _, conn := range ts.conns {
			conns = append(conns, conn)
		}
		ts.connsMu.Unlock()
		for _, conn := range conns {
			conn.Close()
		}
		if ts.pool != nil && !ts.pool.StopAndDrain(DEFAULT_CLOSE_TIMEOUT) {
			slog.Warn("tcp server pool drain timeout")
		}
	})
	return err
}

// ConnCount 返回当前存活的 TCP 连接数。
func (ts *TcpServer) ConnCount() int {
	if ts == nil {
		return 0
	}
	ts.connsMu.Lock()
	defer ts.connsMu.Unlock()
	return len(ts.conns)
}

// SetWriteTimeout 更新服务端默认写超时，并同步到当前所有连接。
func (ts *TcpServer) SetWriteTimeout(timeout time.Duration) {
	if ts == nil {
		return
	}
	atomic.StoreInt64(&ts.writeTimeout, int64(timeout))
	conns := make([]*TcpConn, 0)
	ts.connsMu.Lock()
	for _, conn := range ts.conns {
		conns = append(conns, conn)
	}
	ts.connsMu.Unlock()
	for _, conn := range conns {
		conn.SetWriteTimeout(timeout)
	}
}

// WriteTimeout 返回服务端当前默认写超时。
func (ts *TcpServer) WriteTimeout() time.Duration {
	if ts == nil {
		return 0
	}
	return time.Duration(atomic.LoadInt64(&ts.writeTimeout))
}
