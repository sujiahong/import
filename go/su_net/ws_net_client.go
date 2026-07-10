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

type WSClient struct {
	Addr              string
	Conn              *WSConn
	handler           WSHandler
	done              chan struct{}
	stopOnce          sync.Once
	connMu            sync.RWMutex
	closed            int32
	reconnectEnabled  int32
	reconnecting      int32
	reconnectInterval time.Duration
	writeTimeout      int64
}

func CreateWSClient(addr string, handlers ...WSHandler) (*WSClient, error) {
	return CreateWSClientWithConfig(addr, DefaultWSNetConfig(), handlers...)
}

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

func normalizeWSURL(addr string) string {
	if strings.HasPrefix(addr, "ws://") || strings.HasPrefix(addr, "wss://") {
		return addr
	}
	return fmt.Sprintf("ws://%s%s", addr, defaultWSPath)
}

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

func (wc *WSClient) stopHeartbeat() {
	wc.stopOnce.Do(func() {
		close(wc.done)
	})
}

func (wc *WSClient) getConn() *WSConn {
	if wc == nil {
		return nil
	}
	wc.connMu.RLock()
	defer wc.connMu.RUnlock()
	return wc.Conn
}

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

func (wc *WSClient) isCurrentConn(conn *WSConn) bool {
	wc.connMu.RLock()
	defer wc.connMu.RUnlock()
	return wc.Conn == conn
}

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

func (wc *WSClient) DisableReconnect() {
	if wc == nil {
		return
	}
	atomic.StoreInt32(&wc.reconnectEnabled, 0)
}

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

func (wc *WSClient) SetWriteTimeout(timeout time.Duration) {
	if wc == nil {
		return
	}
	atomic.StoreInt64(&wc.writeTimeout, int64(timeout))
	if conn := wc.getConn(); conn != nil {
		conn.SetWriteTimeout(timeout)
	}
}

func (wc *WSClient) WriteTimeout() time.Duration {
	if wc == nil {
		return 0
	}
	return time.Duration(atomic.LoadInt64(&wc.writeTimeout))
}

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
