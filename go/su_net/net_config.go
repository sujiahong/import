package su_net

import "time"
import "go.local/su_errors"

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

// GNetTcpConfig 定义 gnet TCP client 的调度和重连配置。
type GNetTcpConfig struct {
	// DispatchMode controls whether client packet handlers run inline on the
	// gnet event loop or are submitted to the worker pool.
	DispatchMode GNetDispatchMode
	// ReconnectInterval controls retry spacing when EnableReconnect is used.
	ReconnectInterval time.Duration
}

// DefaultGNetTcpConfig 返回 gnet TCP client 默认配置。
func DefaultGNetTcpConfig() GNetTcpConfig {
	return GNetTcpConfig{
		DispatchMode:      GNetDispatchInline,
		ReconnectInterval: time.Duration(RECONNECT_INTERVAL) * time.Second,
	}
}

func normalizeConnPoolSize(connNum ...int) (int, error) {
	if len(connNum) == 0 {
		return 1, nil
	}
	if connNum[0] <= 0 {
		return 0, su_errors.New(su_errors.CodeInvalidArgument, "connection pool size must be > 0")
	}
	return connNum[0], nil
}
