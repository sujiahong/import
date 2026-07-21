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

// WSClient 管理 WebSocket 客户端连接池、心跳检查和可选自动重连。
type WSClient struct {
	Addr              string         // WebSocket URL。
	Conn              *WSConn        // 当前首个活跃 WebSocket 连接，保留给旧调用方直接访问。
	done              chan struct{}  // 心跳 goroutine 停止信号。
	stopOnce          sync.Once      // 保证心跳停止信号只关闭一次。
	connMu            sync.RWMutex   // 保护 Conn 和 connList。
	connList          []*WSConn      // 当前可用连接列表。
	cfgConnNum        int            // 期望维持的连接数量。
	sendSeq           uint64         // 轮询连接发送的序号。
	closed            int32          // client 是否已关闭，按 atomic 访问。
	reconnectEnabled  int32          // 是否启用自动重连，按 atomic 访问。
	reconnecting      int32          // 是否正在重连，按 atomic 访问。
	reconnectInterval time.Duration  // 自动重连间隔。
	writeTimeout      int64          // 写超时，存储为 time.Duration 的 int64。
	dataHandler       *TcpNetHandler // 业务数据包处理函数。
}

// CreateWSClient 使用默认配置连接 WebSocket 服务端。
func CreateWSClient(addr string, connNum ...int) (*WSClient, error) {
	return CreateWSClientWithConfig(addr, DefaultWSNetConfig(), connNum...)
}

// CreateWSClientWithConfig 使用指定配置连接 WebSocket 服务端。
func CreateWSClientWithConfig(addr string, cfg WSNetConfig, connNum ...int) (*WSClient, error) {
	targetConnNum, err := normalizeConnPoolSize(connNum...)
	if err != nil {
		return nil, err
	}
	url := normalizeWSURL(addr)
	client := &WSClient{
		Addr:              url,
		done:              make(chan struct{}),
		reconnectInterval: time.Duration(RECONNECT_INTERVAL) * time.Second,
		writeTimeout:      int64(cfg.WriteTimeout),
		cfgConnNum:        targetConnNum,
		dataHandler:       newTcpNetHandler(),
	}
	for i := 0; i < targetConnNum; i++ {
		conn, err := client.dialConn()
		if err != nil {
			_ = client.Close()
			return nil, err
		}
		client.addConn(conn)
		client.startReadLoop(conn)
	}
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
			for _, conn := range wc.getConns() {
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

// getConn 并发安全地返回首个当前 WebSocket 连接。
func (wc *WSClient) getConn() *WSConn {
	if wc == nil {
		return nil
	}
	wc.connMu.RLock()
	defer wc.connMu.RUnlock()
	return wc.Conn
}

// getConns 返回当前 WebSocket 连接快照。
func (wc *WSClient) getConns() []*WSConn {
	if wc == nil {
		return nil
	}
	wc.connMu.RLock()
	defer wc.connMu.RUnlock()
	conns := make([]*WSConn, len(wc.connList))
	copy(conns, wc.connList)
	return conns
}

// addConn 将新连接加入连接池，并将客户端写超时同步到新连接。
func (wc *WSClient) addConn(conn *WSConn) {
	wc.connMu.Lock()
	if conn != nil {
		conn.SetWriteTimeout(wc.WriteTimeout())
	}
	wc.connList = append(wc.connList, conn)
	if wc.Conn == nil {
		wc.Conn = conn
	}
	wc.connMu.Unlock()
}

// removeConn 从连接池移除连接，返回连接是否存在。
func (wc *WSClient) removeConn(conn *WSConn) bool {
	if wc == nil || conn == nil {
		return false
	}
	wc.connMu.Lock()
	defer wc.connMu.Unlock()
	removed := false
	for i, item := range wc.connList {
		if item == conn {
			wc.connList = append(wc.connList[:i], wc.connList[i+1:]...)
			removed = true
			break
		}
	}
	if wc.Conn == conn {
		wc.Conn = nil
		if len(wc.connList) > 0 {
			wc.Conn = wc.connList[0]
		}
	}
	return removed
}

// clearConns 清空连接池并返回原连接快照。
func (wc *WSClient) clearConns() []*WSConn {
	if wc == nil {
		return nil
	}
	wc.connMu.Lock()
	conns := make([]*WSConn, len(wc.connList))
	copy(conns, wc.connList)
	wc.connList = nil
	wc.Conn = nil
	wc.connMu.Unlock()
	return conns
}

// startReadLoop 启动连接读循环，并在读循环退出后按配置触发重连或停止心跳。
func (wc *WSClient) startReadLoop(conn *WSConn) {
	if wc == nil || conn == nil {
		return
	}
	go func() {
		conn.readLoop(wc.HandleMessage)
		if !wc.removeConn(conn) {
			return
		}
		if atomic.LoadInt32(&wc.closed) == 1 {
			return
		}
		if atomic.LoadInt32(&wc.reconnectEnabled) == 1 {
			wc.scheduleReconnect()
			return
		}
		if wc.ConnCount() == 0 {
			wc.stopHeartbeat()
		}
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

// Reconnect 立即执行一次互斥的 WebSocket 连接池补齐。
func (wc *WSClient) Reconnect() error {
	if wc == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "websocket client is nil")
	}
	if !atomic.CompareAndSwapInt32(&wc.reconnecting, 0, 1) {
		return su_errors.NewRetryable(su_errors.CodeUnavailable, "websocket client reconnect already running")
	}
	defer atomic.StoreInt32(&wc.reconnecting, 0)
	return wc.ensureConnections()
}

// dialConn 创建一条新 WebSocket 连接。
func (wc *WSClient) dialConn() (*WSConn, error) {
	if atomic.LoadInt32(&wc.closed) == 1 {
		return nil, su_errors.New(su_errors.CodeUnavailable, "websocket client is closed")
	}
	conn, _, err := websocket.DefaultDialer.Dial(wc.Addr, nil)
	if err != nil {
		return nil, su_errors.WrapRetryable(su_errors.CodeUnavailable, "dial websocket failed", err)
	}
	return newWSConnWithWriteTimeout(conn, wc.WriteTimeout()), nil
}

// ensureConnections 补齐连接池到配置连接数。
func (wc *WSClient) ensureConnections() error {
	if wc == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "websocket client is nil")
	}
	if atomic.LoadInt32(&wc.closed) == 1 {
		return su_errors.New(su_errors.CodeUnavailable, "websocket client is closed")
	}
	target := wc.cfgConnNum
	if target <= 0 {
		target = 1
	}
	for wc.ConnCount() < target {
		conn, err := wc.dialConn()
		if err != nil {
			return err
		}
		wc.addConn(conn)
		wc.startReadLoop(conn)
	}
	return nil
}

// scheduleReconnect 后台循环补齐连接池，直到成功、关闭或禁用重连。
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
		target := wc.cfgConnNum
		if target <= 0 {
			target = 1
		}
		for atomic.LoadInt32(&wc.closed) == 0 && atomic.LoadInt32(&wc.reconnectEnabled) == 1 {
			if wc.ConnCount() >= target {
				return
			}
			timer := time.NewTimer(interval)
			select {
			case <-wc.done:
				timer.Stop()
				return
			case <-timer.C:
			}
			if err := wc.ensureConnections(); err != nil {
				slog.Error("websocket client reconnect failed", zap.Error(err))
				continue
			}
		}
	}()
}

// Send 通过连接池轮询发送数据包。
func (wc *WSClient) Send(dp *DataProtocol) error {
	if wc == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "websocket client is nil")
	}
	conn, err := wc.nextConn()
	if err != nil {
		return err
	}
	return conn.Send(dp)
}

// nextConn 按轮询策略选择一条当前活跃连接。
func (wc *WSClient) nextConn() (*WSConn, error) {
	wc.connMu.RLock()
	connCount := len(wc.connList)
	if connCount == 0 {
		wc.connMu.RUnlock()
		return nil, su_errors.NewRetryable(su_errors.CodeUnavailable, "no active websocket client connection")
	}
	target := int(atomic.AddUint64(&wc.sendSeq, 1)-1) % connCount
	conn := wc.connList[target]
	wc.connMu.RUnlock()
	return conn, nil
}

// SetWriteTimeout 更新客户端及连接池所有连接的写超时。
func (wc *WSClient) SetWriteTimeout(timeout time.Duration) {
	if wc == nil {
		return
	}
	atomic.StoreInt64(&wc.writeTimeout, int64(timeout))
	for _, conn := range wc.getConns() {
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

// Close 关闭客户端心跳和连接池所有连接。
func (wc *WSClient) Close() error {
	if wc == nil {
		return nil
	}
	atomic.StoreInt32(&wc.closed, 1)
	wc.stopHeartbeat()
	conns := wc.clearConns()
	var err error
	for _, conn := range conns {
		if closeErr := conn.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}
	return err
}

// ConnCount 返回当前活跃 WebSocket client 连接数量。
func (wc *WSClient) ConnCount() int {
	if wc == nil {
		return 0
	}
	wc.connMu.RLock()
	defer wc.connMu.RUnlock()
	return len(wc.connList)
}

func (wc *WSClient) RegisterManualResponseHandler(rqPackId uint32, rsPackId uint32, handler MessageHandler) error {
	if wc == nil || wc.dataHandler == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "wc or wc.dataHandler is nil")
	}
	return wc.dataHandler.RegisterManualResponseHandler(rqPackId, rsPackId, handler)
}

func (wc *WSClient) RegisterRequestResponseHandler(rqPackId uint32, rsPackId uint32, handler MessageHandler) error {
	if wc == nil || wc.dataHandler == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "wc or wc.dataHandler is nil")
	}
	return wc.dataHandler.RegisterRequestResponseHandler(rqPackId, rsPackId, handler)
}

func (wc *WSClient) RegisterOneWayHandler(packId uint32, handler MessageHandler) error {
	if wc == nil || wc.dataHandler == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "wc or wc.dataHandler is nil")
	}
	return wc.dataHandler.RegisterOneWayHandler(packId, handler)
}

func (wc *WSClient) HandleMessage(conn *WSConn, dp *DataProtocol) {
	if wc == nil || wc.dataHandler == nil || dp == nil {
		slog.Error("websocket client handler unavailable")
		return
	}
	dispatchTcpNetHandler(wc.dataHandler, &HandlerContext{Conn: conn, Packet: dp})
}
