package su_net

import "time"

const (
	PING_PONG_INTERVAL      uint32 = 19
	RECONNECT_INTERVAL      uint32 = 5
	DEFAULT_REQUEST_TIMEOUT        = 30 * time.Second
	DEFAULT_CLOSE_TIMEOUT          = 3 * time.Second
	DEFAULT_WRITE_TIMEOUT          = 5 * time.Second
	DEFAULT_POOL_WORKERS    uint32 = 16
	DEFAULT_POOL_QUEUE_SIZE uint32 = 1024

	defaultWSPath = "/ws"
)

// TcpNetConfig 定义 TCP client/server 的网络超时配置。
type TcpNetConfig struct {
	// WriteTimeout controls per-packet write deadlines. A zero or negative
	// value disables SetWriteDeadline on the hot path.
	WriteTimeout time.Duration
}

// DefaultTcpNetConfig 返回 TCP 网络默认配置。
func DefaultTcpNetConfig() TcpNetConfig {
	return TcpNetConfig{WriteTimeout: DEFAULT_WRITE_TIMEOUT}
}

// WSNetConfig 定义 WebSocket client/server 的网络超时配置。
type WSNetConfig struct {
	// WriteTimeout controls per-message write deadlines. A zero or negative
	// value disables SetWriteDeadline on the hot path.
	WriteTimeout time.Duration
}

// DefaultWSNetConfig 返回 WebSocket 网络默认配置。
func DefaultWSNetConfig() WSNetConfig {
	return WSNetConfig{WriteTimeout: DEFAULT_WRITE_TIMEOUT}
}
