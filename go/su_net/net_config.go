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

type TcpNetConfig struct {
	// WriteTimeout controls per-packet write deadlines. A zero or negative
	// value disables SetWriteDeadline on the hot path.
	WriteTimeout time.Duration
}

func DefaultTcpNetConfig() TcpNetConfig {
	return TcpNetConfig{WriteTimeout: DEFAULT_WRITE_TIMEOUT}
}

type WSNetConfig struct {
	// WriteTimeout controls per-message write deadlines. A zero or negative
	// value disables SetWriteDeadline on the hot path.
	WriteTimeout time.Duration
}

func DefaultWSNetConfig() WSNetConfig {
	return WSNetConfig{WriteTimeout: DEFAULT_WRITE_TIMEOUT}
}
