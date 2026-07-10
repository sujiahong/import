package su_net

import (
	"fmt"
	"go.local/su_errors"
	slog "go.local/su_log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// WSClient 管理一个 WebSocket 客户端连接、心跳检查和可选自动重连。
type WSClient struct {
	Addr              string        // WebSocket URL。
	Conn              *WSConn       // 当前活跃 WebSocket 连接。
	handler           WSHandler     // 业务包处理函数。
	done              chan struct{} // 心跳 goroutine 停止信号。
	stopOnce          sync.Once     // 保证心跳停止信号只关闭一次。
	connMu            sync.RWMutex  // 保护 Conn 替换和读取。
	closed            int32         // client 是否已关闭，按 atomic 访问。
	reconnectEnabled  int32         // 是否启用自动重连，按 atomic 访问。
	reconnecting      int32         // 是否正在重连，按 atomic 访问。
	reconnectInterval time.Duration // 自动重连间隔。
	writeTimeout      int64         // 写超时，存储为 time.Duration 的 int64。
}

// CreateWSClient 使用默认配置连接 WebSocket 服务端。
func CreateWSClient(addr string, handlers ...WSHandler) (*WSClient, error) {
	return CreateWSClientWithConfig(addr, DefaultWSNetConfig(), handlers...)
}

// CreateWSClientWithConfig 使用指定配置连接 WebSocket 服务端。
func CreateWSClientWithConfig(addr string, cfg WSNetConfig, handlers ...WSHandler) (*WSClient, error) {
	url := normalizeWSURL(addr)
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, su_errors.WrapRetryable(su_errors.CodeUnavailable, "dial websocket failed", err)
	}
	client := &WSClient{
		Addr:              url,
		Conn:              newWSConnWithWriteTimeout(conn, cfg.WriteTimeout),
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

// normalizeWSURL 将裸 host 地址补全为默认 ws URL。
func normalizeWSURL(addr string) string {
	if strings.HasPrefix(addr, "ws://") || strings.HasPrefix(addr, "wss://") {
		return addr
	}
	return fmt.Sprintf("ws://%s%s", addr, defaultWSPath)
}

// heartbeatLoop 定时检查当前连接的 PONG 响应。
func (wc *WSClient) heartbeatLoop() {
	ticker := time.NewTicker(time.Duration(PING_PONG_INTERVAL) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if conn := wc.getConn(); conn != nil {
				conn.CheckPong()
			}
		case <-wc.done:
			return
		}
	}
}

// stopHeartbeat 停止客户端心跳 goroutine。
func (wc *WSClient) stopHeartbeat() {
	wc.stopOnce.Do(func() {
		close(wc.done)
	})
}

// getConn 并发安全地返回当前 WebSocket 连接。
func (wc *WSClient) getConn() *WSConn {
	if wc == nil {
		return nil
	}
	wc.connMu.RLock()
	defer wc.connMu.RUnlock()
	return wc.Conn
}

// setConn 替换当前 WebSocket 连接，并将客户端写超时同步到新连接。
func (wc *WSClient) setConn(conn *WSConn) *WSConn {
	wc.connMu.Lock()
	if conn != nil {
		conn.SetWriteTimeout(wc.WriteTimeout())
	}
	oldConn := wc.Conn
	wc.Conn = conn
	wc.connMu.Unlock()
	return oldConn
}

// isCurrentConn 判断给定连接是否仍是客户端当前连接。
func (wc *WSClient) isCurrentConn(conn *WSConn) bool {
	wc.connMu.RLock()
	defer wc.connMu.RUnlock()
	return wc.Conn == conn
}

// startReadLoop 启动连接读循环，并在读循环退出后按配置触发重连或停止心跳。
func (wc *WSClient) startReadLoop(conn *WSConn) {
	if wc == nil || conn == nil {
		return
	}
	go func() {
		conn.readLoop(wc.handler)
		if !wc.isCurrentConn(conn) {
			return
		}
		if atomic.LoadInt32(&wc.closed) == 1 {
			return
		}
		if atomic.LoadInt32(&wc.reconnectEnabled) == 1 {
			wc.scheduleReconnect()
			return
		}
		wc.stopHeartbeat()
	}()
}

// EnableReconnect 开启断线自动重连，可选设置重连间隔。
func (wc *WSClient) EnableReconnect(interval ...time.Duration) {
	if wc == nil {
		return
	}
	if len(interval) > 0 && interval[0] > 0 {
		wc.reconnectInterval = interval[0]
	}
	if wc.reconnectInterval <= 0 {
		wc.reconnectInterval = time.Duration(RECONNECT_INTERVAL) * time.Second
	}
	atomic.StoreInt32(&wc.reconnectEnabled, 1)
}

// DisableReconnect 关闭断线自动重连。
func (wc *WSClient) DisableReconnect() {
	if wc == nil {
		return
	}
	atomic.StoreInt32(&wc.reconnectEnabled, 0)
}

// Reconnect 立即执行一次互斥的 WebSocket 重连。
func (wc *WSClient) Reconnect() error {
	if wc == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "websocket client is nil")
	}
	if !atomic.CompareAndSwapInt32(&wc.reconnecting, 0, 1) {
		return su_errors.NewRetryable(su_errors.CodeUnavailable, "websocket client reconnect already running")
	}
	defer atomic.StoreInt32(&wc.reconnecting, 0)
	return wc.reconnectOnce()
}

// reconnectOnce 创建新连接、替换当前连接并启动读循环。
func (wc *WSClient) reconnectOnce() error {
	if atomic.LoadInt32(&wc.closed) == 1 {
		return su_errors.New(su_errors.CodeUnavailable, "websocket client is closed")
	}
	conn, _, err := websocket.DefaultDialer.Dial(wc.Addr, nil)
	if err != nil {
		return su_errors.WrapRetryable(su_errors.CodeUnavailable, "dial websocket failed", err)
	}
	newConn := newWSConnWithWriteTimeout(conn, wc.WriteTimeout())
	oldConn := wc.setConn(newConn)
	if oldConn != nil && oldConn != newConn {
		_ = oldConn.Close()
	}
	wc.startReadLoop(newConn)
	return nil
}

// scheduleReconnect 后台循环重连，直到成功、关闭或禁用重连。
func (wc *WSClient) scheduleReconnect() {
	if wc == nil || !atomic.CompareAndSwapInt32(&wc.reconnecting, 0, 1) {
		return
	}
	go func() {
		defer atomic.StoreInt32(&wc.reconnecting, 0)
		interval := wc.reconnectInterval
		if interval <= 0 {
			interval = time.Duration(RECONNECT_INTERVAL) * time.Second
		}
		for atomic.LoadInt32(&wc.closed) == 0 && atomic.LoadInt32(&wc.reconnectEnabled) == 1 {
			timer := time.NewTimer(interval)
			select {
			case <-wc.done:
				timer.Stop()
				return
			case <-timer.C:
			}
			if err := wc.reconnectOnce(); err != nil {
				slog.Error("websocket client reconnect failed", zap.Error(err))
				continue
			}
			return
		}
	}()
}

// Send 通过当前连接发送数据包。
func (wc *WSClient) Send(dp *DataProtocol) error {
	if wc == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "websocket client is nil")
	}
	conn := wc.getConn()
	if conn == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "websocket client is nil")
	}
	return conn.Send(dp)
}

// SetWriteTimeout 更新客户端及当前连接的写超时。
func (wc *WSClient) SetWriteTimeout(timeout time.Duration) {
	if wc == nil {
		return
	}
	atomic.StoreInt64(&wc.writeTimeout, int64(timeout))
	if conn := wc.getConn(); conn != nil {
		conn.SetWriteTimeout(timeout)
	}
}

// WriteTimeout 返回客户端当前写超时。
func (wc *WSClient) WriteTimeout() time.Duration {
	if wc == nil {
		return 0
	}
	return time.Duration(atomic.LoadInt64(&wc.writeTimeout))
}

// Close 关闭客户端心跳和当前连接。
func (wc *WSClient) Close() error {
	if wc == nil {
		return nil
	}
	atomic.StoreInt32(&wc.closed, 1)
	wc.stopHeartbeat()
	conn := wc.getConn()
	if conn == nil {
		return nil
	}
	return conn.Close()
}
