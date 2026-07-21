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

// TcpClient 管理 TCP 客户端连接池、心跳检查和可选自动重连。
type TcpClient struct {
	Addr              string         // 远端 TCP 地址。
	Conn              *TcpConn       // 当前首个活跃 TCP 连接，保留给旧调用方直接访问。
	done              chan struct{}  // 心跳 goroutine 停止信号。
	stopOnce          sync.Once      // 保证心跳停止信号只关闭一次。
	connMu            sync.RWMutex   // 保护 Conn 和 connList。
	connList          []*TcpConn     // 当前可用连接列表。
	cfgConnNum        int            // 期望维持的连接数量。
	sendSeq           uint64         // 轮询连接发送的序号。
	closed            int32          // client 是否已关闭，按 atomic 访问。
	reconnectEnabled  int32          // 是否启用自动重连，按 atomic 访问。
	reconnecting      int32          // 是否正在重连，按 atomic 访问。
	reconnectInterval time.Duration  // 自动重连间隔。
	writeTimeout      int64          // 写超时，存储为 time.Duration 的 int64。
	dataHandler       *TcpNetHandler // 业务数据包处理函数。
}

// CreateTcpClient 使用默认配置连接 TCP 服务端。
func CreateTcpClient(addr string, connNum ...int) (*TcpClient, error) {
	return CreateTcpClientWithConfig(addr, DefaultTcpNetConfig(), connNum...)
}

// CreateTcpClientWithConfig 使用指定配置连接 TCP 服务端。
func CreateTcpClientWithConfig(addr string, cfg TcpNetConfig, connNum ...int) (*TcpClient, error) {
	targetConnNum, err := normalizeConnPoolSize(connNum...)
	if err != nil {
		return nil, err
	}
	client := &TcpClient{
		Addr:              addr,
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
			for _, conn := range tc.getConns() {
				conn.CheckPong()
			}
		case <-tc.done:
			return
		}
	}
}

// getConn 并发安全地返回首个当前 TCP 连接。
func (tc *TcpClient) getConn() *TcpConn {
	if tc == nil {
		return nil
	}
	tc.connMu.RLock()
	defer tc.connMu.RUnlock()
	return tc.Conn
}

// getConns 返回当前 TCP 连接快照。
func (tc *TcpClient) getConns() []*TcpConn {
	if tc == nil {
		return nil
	}
	tc.connMu.RLock()
	defer tc.connMu.RUnlock()
	conns := make([]*TcpConn, len(tc.connList))
	copy(conns, tc.connList)
	return conns
}

// addConn 将新连接加入连接池，并将客户端写超时同步到新连接。
func (tc *TcpClient) addConn(conn *TcpConn) {
	tc.connMu.Lock()
	if conn != nil {
		conn.SetWriteTimeout(tc.WriteTimeout())
	}
	tc.connList = append(tc.connList, conn)
	if tc.Conn == nil {
		tc.Conn = conn
	}
	tc.connMu.Unlock()
}

// removeConn 从连接池移除连接，返回连接是否存在。
func (tc *TcpClient) removeConn(conn *TcpConn) bool {
	if tc == nil || conn == nil {
		return false
	}
	tc.connMu.Lock()
	defer tc.connMu.Unlock()
	removed := false
	for i, item := range tc.connList {
		if item == conn {
			tc.connList = append(tc.connList[:i], tc.connList[i+1:]...)
			removed = true
			break
		}
	}
	if tc.Conn == conn {
		tc.Conn = nil
		if len(tc.connList) > 0 {
			tc.Conn = tc.connList[0]
		}
	}
	return removed
}

// clearConns 清空连接池并返回原连接快照。
func (tc *TcpClient) clearConns() []*TcpConn {
	if tc == nil {
		return nil
	}
	tc.connMu.Lock()
	conns := make([]*TcpConn, len(tc.connList))
	copy(conns, tc.connList)
	tc.connList = nil
	tc.Conn = nil
	tc.connMu.Unlock()
	return conns
}

// startReadLoop 启动连接读循环，并在读循环退出后按配置触发重连或停止心跳。
func (tc *TcpClient) startReadLoop(conn *TcpConn) {
	if tc == nil || conn == nil {
		return
	}
	go func() {
		conn.readLoop(tc.HandleMessage)
		if !tc.removeConn(conn) {
			return
		}
		if atomic.LoadInt32(&tc.closed) == 1 {
			return
		}
		if atomic.LoadInt32(&tc.reconnectEnabled) == 1 {
			tc.scheduleReconnect()
			return
		}
		if tc.ConnCount() == 0 {
			tc.stopHeartbeat()
		}
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

// Reconnect 立即执行一次互斥的 TCP 连接池补齐。
func (tc *TcpClient) Reconnect() error {
	if tc == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "tcp client is nil")
	}
	if !atomic.CompareAndSwapInt32(&tc.reconnecting, 0, 1) {
		return su_errors.NewRetryable(su_errors.CodeUnavailable, "tcp client reconnect already running")
	}
	defer atomic.StoreInt32(&tc.reconnecting, 0)
	return tc.ensureConnections()
}

// dialConn 创建一条新 TCP 连接。
func (tc *TcpClient) dialConn() (*TcpConn, error) {
	if atomic.LoadInt32(&tc.closed) == 1 {
		return nil, su_errors.New(su_errors.CodeUnavailable, "tcp client is closed")
	}
	tcpAddr, err := net.ResolveTCPAddr("tcp", tc.Addr)
	if err != nil {
		return nil, su_errors.Wrap(su_errors.CodeInvalidArgument, "resolve tcp addr failed", err)
	}
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return nil, su_errors.WrapRetryable(su_errors.CodeUnavailable, "dial tcp failed", err)
	}
	return newTcpConnWithWriteTimeout(conn, tc.WriteTimeout()), nil
}

// ensureConnections 补齐连接池到配置连接数。
func (tc *TcpClient) ensureConnections() error {
	if tc == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "tcp client is nil")
	}
	if atomic.LoadInt32(&tc.closed) == 1 {
		return su_errors.New(su_errors.CodeUnavailable, "tcp client is closed")
	}
	target := tc.cfgConnNum
	if target <= 0 {
		target = 1
	}
	for tc.ConnCount() < target {
		conn, err := tc.dialConn()
		if err != nil {
			return err
		}
		tc.addConn(conn)
		tc.startReadLoop(conn)
	}
	return nil
}

// scheduleReconnect 后台循环补齐连接池，直到成功、关闭或禁用重连。
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
		target := tc.cfgConnNum
		if target <= 0 {
			target = 1
		}
		for atomic.LoadInt32(&tc.closed) == 0 && atomic.LoadInt32(&tc.reconnectEnabled) == 1 {
			if tc.ConnCount() >= target {
				return
			}
			timer := time.NewTimer(interval)
			select {
			case <-tc.done:
				timer.Stop()
				return
			case <-timer.C:
			}
			if err := tc.ensureConnections(); err != nil {
				slog.Error("tcp client reconnect failed", zap.Error(err))
				continue
			}
		}
	}()
}

// Send 通过连接池轮询发送数据包。
func (tc *TcpClient) Send(dp *DataProtocol) error {
	if tc == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "tcp client is nil")
	}
	conn, err := tc.nextConn()
	if err != nil {
		return err
	}
	return conn.Send(dp)
}

// nextConn 按轮询策略选择一条当前活跃连接。
func (tc *TcpClient) nextConn() (*TcpConn, error) {
	tc.connMu.RLock()
	connCount := len(tc.connList)
	if connCount == 0 {
		tc.connMu.RUnlock()
		return nil, su_errors.NewRetryable(su_errors.CodeUnavailable, "no active tcp client connection")
	}
	target := int(atomic.AddUint64(&tc.sendSeq, 1)-1) % connCount
	conn := tc.connList[target]
	tc.connMu.RUnlock()
	return conn, nil
}

// SetWriteTimeout 更新客户端及连接池所有连接的写超时。
func (tc *TcpClient) SetWriteTimeout(timeout time.Duration) {
	if tc == nil {
		return
	}
	atomic.StoreInt64(&tc.writeTimeout, int64(timeout))
	for _, conn := range tc.getConns() {
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

// Close 关闭客户端心跳和连接池所有连接。
func (tc *TcpClient) Close() error {
	if tc == nil {
		return nil
	}
	atomic.StoreInt32(&tc.closed, 1)
	tc.stopHeartbeat()
	conns := tc.clearConns()
	var err error
	for _, conn := range conns {
		if closeErr := conn.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}
	return err
}

// ConnCount 返回当前活跃 TCP client 连接数量。
func (tc *TcpClient) ConnCount() int {
	if tc == nil {
		return 0
	}
	tc.connMu.RLock()
	defer tc.connMu.RUnlock()
	return len(tc.connList)
}

func (tc *TcpClient) RegisterManualResponseHandler(rqPackId uint32, rsPackId uint32, handler MessageHandler) error {
	if tc == nil || tc.dataHandler == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "tc or tc.dataHandler is nil")
	}
	return tc.dataHandler.RegisterManualResponseHandler(rqPackId, rsPackId, handler)
}

func (tc *TcpClient) RegisterRequestResponseHandler(rqPackId uint32, rsPackId uint32, handler MessageHandler) error {
	if tc == nil || tc.dataHandler == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "tc or tc.dataHandler is nil")
	}
	return tc.dataHandler.RegisterRequestResponseHandler(rqPackId, rsPackId, handler)
}

func (tc *TcpClient) RegisterOneWayHandler(packId uint32, handler MessageHandler) error {
	if tc == nil || tc.dataHandler == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "tc or tc.dataHandler is nil")
	}
	return tc.dataHandler.RegisterOneWayHandler(packId, handler)
}

func (tc *TcpClient) HandleMessage(conn *TcpConn, dp *DataProtocol) {
	if tc == nil || tc.dataHandler == nil || dp == nil {
		slog.Error("tcp client handler unavailable")
		return
	}
	dispatchTcpNetHandler(tc.dataHandler, &HandlerContext{Conn: conn, Packet: dp})
}
