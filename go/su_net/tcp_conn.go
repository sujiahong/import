package su_net

import (
	slog "go/su_log"
	"sync"
	"sync/atomic"
	"time"
)

////gnet网络连接结构
type GNetConn struct {
	gnet.Conn
	state     int32       /////是否使用 1 使用  0 未使用
	data_cache []byte     ////网络数据缓存
}
