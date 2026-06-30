package su_net

import (
	"sync/atomic"
	"time"
)

var routeIDSeq uint64 = uint64(time.Now().UnixNano())

func nextRouteID() uint64 {
	return atomic.AddUint64(&routeIDSeq, 1)
}
