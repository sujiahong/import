package su_net

import (
	"sync/atomic"
	"time"
)

var routeIDSeq uint64 = uint64(time.Now().UnixNano())

// nextRouteID 返回进程内递增的请求路由 ID。
func nextRouteID() uint64 {
	return atomic.AddUint64(&routeIDSeq, 1)
}
