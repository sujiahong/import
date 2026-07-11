package su_net

import (
	"go.local/su_errors"
	slog "go.local/su_log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// TcpClient 管理一个 TCP 客户端连接、心跳检查和可选自动重连。
type TcpClient struct {
	Addr              string        // 远端 TCP 地址。
	Conn              *TcpConn      // 当前活跃 TCP 连接。
	handler           TcpHandler    // 业务包处理函数。
	done              chan struct{} // 心跳 goroutine 停止信号。
	stopOnce          sync.Once     // 保证心跳停止信号只关闭一次。
	connMu            sync.RWMutex  // 保护 Conn 替换和读取。
	closed            int32         // client 是否已关闭，按 atomic 访问。
	reconnectEnabled  int32         // 是否启用自动重连，按 atomic 访问。
	reconnecting      int32         // 是否正在重连，按 atomic 访问。
	reconnectInterval time.Duration // 自动重连间隔。
	writeTimeout      int64         // 写超时，存储为 time.Duration 的 int64。
	dataHandler 	  DataHandler   // 业务数据包处理函数。
}

// CreateTcpClient 使用默认配置连接 TCP 服务端。
func CreateTcpClient(addr string, handlers ...TcpHandler) (*TcpClient, error) {
	return CreateTcpClientWithConfig(addr, DefaultTcpNetConfig(), handlers...)
}

// CreateTcpClientWithConfig 使用指定配置连接 TCP 服务端。
func CreateTcpClientWithConfig(addr string, cfg TcpNetConfig, handlers ...TcpHandler) (*TcpClient, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, su_errors.Wrap(su_errors.CodeInvalidArgument, "resolve tcp addr failed", err)
	}
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return nil, su_errors.WrapRetryable(su_errors.CodeUnavailable, "dial tcp failed", err)
	}
	client := &TcpClient{
		Addr:              addr,
		Conn:              newTcpConnWithWriteTimeout(conn, cfg.WriteTimeout),
		done:              make(chan struct{}),
		reconnectInterval: time.Duration(RECONNECT_INTERVAL) * time.Second,
		writeTimeout:      int64(cfg.WriteTimeout),
	}
	if len(handlers) > 0 {
		client.handler = handlers[0]
	}
	client.startReadLoop(client.Conn)
	go client.heartbeatLoop()
	return client, nil
}

// stopHeartbeat 停止客户端心跳 goroutine。
func (tc *TcpClient) stopHeartbeat() {
	tc.stopOnce.Do(func() {
		close(tc.done)
	})
}

// heartbeatLoop 定时检查当前连接的 PONG 响应。
func (tc *TcpClient) heartbeatLoop() {
	ticker := time.NewTicker(time.Duration(PING_PONG_INTERVAL) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if conn := tc.getConn(); conn != nil {
				conn.CheckPong()
			}
		case <-tc.done:
			return
		}
	}
}

// getConn 并发安全地返回当前 TCP 连接。
func (tc *TcpClient) getConn() *TcpConn {
	if tc == nil {
		return nil
	}
	tc.connMu.RLock()
	defer tc.connMu.RUnlock()
	return tc.Conn
}

// setConn 替换当前 TCP 连接，并将客户端写超时同步到新连接。
func (tc *TcpClient) setConn(conn *TcpConn) *TcpConn {
	tc.connMu.Lock()
	if conn != nil {
		conn.SetWriteTimeout(tc.WriteTimeout())
	}
	oldConn := tc.Conn
	tc.Conn = conn
	tc.connMu.Unlock()
	return oldConn
}

// isCurrentConn 判断给定连接是否仍是客户端当前连接。
func (tc *TcpClient) isCurrentConn(conn *TcpConn) bool {
	tc.connMu.RLock()
	defer tc.connMu.RUnlock()
	return tc.Conn == conn
}

// startReadLoop 启动连接读循环，并在读循环退出后按配置触发重连或停止心跳。
func (tc *TcpClient) startReadLoop(conn *TcpConn) {
	if tc == nil || conn == nil {
		return
	}
	go func() {
		conn.readLoop(tc.handler)
		if !tc.isCurrentConn(conn) {
			return
		}
		if atomic.LoadInt32(&tc.closed) == 1 {
			return
		}
		if atomic.LoadInt32(&tc.reconnectEnabled) == 1 {
			tc.scheduleReconnect()
			return
		}
		tc.stopHeartbeat()
	}()
}

// EnableReconnect 开启断线自动重连，可选设置重连间隔。
func (tc *TcpClient) EnableReconnect(interval ...time.Duration) {
	if tc == nil {
		return
	}
	if len(interval) > 0 && interval[0] > 0 {
		tc.reconnectInterval = interval[0]
	}
	if tc.reconnectInterval <= 0 {
		tc.reconnectInterval = time.Duration(RECONNECT_INTERVAL) * time.Second
	}
	atomic.StoreInt32(&tc.reconnectEnabled, 1)
}

// DisableReconnect 关闭断线自动重连。
func (tc *TcpClient) DisableReconnect() {
	if tc == nil {
		return
	}
	atomic.StoreInt32(&tc.reconnectEnabled, 0)
}

// Reconnect 立即执行一次互斥的 TCP 重连。
func (tc *TcpClient) Reconnect() error {
	if tc == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "tcp client is nil")
	}
	if !atomic.CompareAndSwapInt32(&tc.reconnecting, 0, 1) {
		return su_errors.NewRetryable(su_errors.CodeUnavailable, "tcp client reconnect already running")
	}
	defer atomic.StoreInt32(&tc.reconnecting, 0)
	return tc.reconnectOnce()
}

// reconnectOnce 创建新连接、替换当前连接并启动读循环。
func (tc *TcpClient) reconnectOnce() error {
	if atomic.LoadInt32(&tc.closed) == 1 {
		return su_errors.New(su_errors.CodeUnavailable, "tcp client is closed")
	}
	tcpAddr, err := net.ResolveTCPAddr("tcp", tc.Addr)
	if err != nil {
		return su_errors.Wrap(su_errors.CodeInvalidArgument, "resolve tcp addr failed", err)
	}
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return su_errors.WrapRetryable(su_errors.CodeUnavailable, "dial tcp failed", err)
	}
	newConn := newTcpConnWithWriteTimeout(conn, tc.WriteTimeout())
	oldConn := tc.setConn(newConn)
	if oldConn != nil && oldConn != newConn {
		_ = oldConn.Close()
	}
	tc.startReadLoop(newConn)
	return nil
}

// scheduleReconnect 后台循环重连，直到成功、关闭或禁用重连。
func (tc *TcpClient) scheduleReconnect() {
	if tc == nil || !atomic.CompareAndSwapInt32(&tc.reconnecting, 0, 1) {
		return
	}
	go func() {
		defer atomic.StoreInt32(&tc.reconnecting, 0)
		interval := tc.reconnectInterval
		if interval <= 0 {
			interval = time.Duration(RECONNECT_INTERVAL) * time.Second
		}
		for atomic.LoadInt32(&tc.closed) == 0 && atomic.LoadInt32(&tc.reconnectEnabled) == 1 {
			timer := time.NewTimer(interval)
			select {
			case <-tc.done:
				timer.Stop()
				return
			case <-timer.C:
			}
			if err := tc.reconnectOnce(); err != nil {
				slog.Error("tcp client reconnect failed", zap.Error(err))
				continue
			}
			return
		}
	}()
}

// Send 通过当前连接发送数据包。
func (tc *TcpClient) Send(dp *DataProtocol) error {
	if tc == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "tcp client is nil")
	}
	conn := tc.getConn()
	if conn == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "tcp client is nil")
	}
	return conn.Send(dp)
}

// SetWriteTimeout 更新客户端及当前连接的写超时。
func (tc *TcpClient) SetWriteTimeout(timeout time.Duration) {
	if tc == nil {
		return
	}
	atomic.StoreInt64(&tc.writeTimeout, int64(timeout))
	if conn := tc.getConn(); conn != nil {
		conn.SetWriteTimeout(timeout)
	}
}

// WriteTimeout 返回客户端当前写超时。
func (tc *TcpClient) WriteTimeout() time.Duration {
	if tc == nil {
		return 0
	}
	return time.Duration(atomic.LoadInt64(&tc.writeTimeout))
}

// Close 关闭客户端心跳和当前连接。
func (tc *TcpClient) Close() error {
	if tc == nil {
		return nil
	}
	atomic.StoreInt32(&tc.closed, 1)
	tc.stopHeartbeat()
	conn := tc.getConn()
	if conn == nil {
		return nil
	}
	return conn.Close()
}
