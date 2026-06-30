package my_util

import (
	slog "go.local/su_log"
	"go.uber.org/zap"
	"sync"
	"sync/atomic"
)

// ///go协程池
type GoPool struct {
	func_pool_slice []chan func()
	coroutine_num   uint32
	cache_num       uint32
	state           int32 ////状态 0 停止 1 运行
	done            chan struct{}
	stopOnce        sync.Once
}

func NewGoPool(a_go_num, a_cache_num uint32) *GoPool {
	if a_go_num == 0 {
		a_go_num = 1
	}
	gp := &GoPool{
		coroutine_num: a_go_num,
		cache_num:     a_cache_num,
		state:         1,
		done:          make(chan struct{}),
	}
	gp.func_pool_slice = make([]chan func(), a_go_num)
	var i uint32 = 0
	for i = 0; i < a_go_num; i++ {
		gp.func_pool_slice[i] = make(chan func(), a_cache_num)
	}
	gp.Start()
	return gp
}

func (gp *GoPool) Start() {
	var i uint32 = 0
	for i = 0; i < gp.coroutine_num; i++ {
		go gp.run(i, gp.func_pool_slice[i])
	}
}

func (gp *GoPool) run(index uint32, a_func_ch chan func()) {
	for {
		select {
		case <-gp.done:
			slog.Warn("协程池关闭", zap.Any("index=", index))
			return
		case f := <-a_func_ch:
			if f != nil {
				f()
			}
		}
	}
}

func (gp *GoPool) Stop() {
	gp.stopOnce.Do(func() {
		atomic.StoreInt32(&gp.state, 0)
		close(gp.done)
	})
}

func (gp *GoPool) SendTask(a_shardingid uint64, a_func func()) bool {
	if gp == nil {
		return false
	}
	if atomic.LoadInt32(&gp.state) == 0 {
		slog.Warn("协程池关闭了", zap.Any("a_shardingid=", a_shardingid))
		return false
	}
	index := a_shardingid % uint64(gp.coroutine_num)
	select {
	case <-gp.done:
		slog.Warn("协程池关闭了", zap.Any("a_shardingid=", a_shardingid))
		return false
	case gp.func_pool_slice[index] <- a_func:
		return true
	}
}

func (gp *GoPool) TrySendTask(a_shardingid uint64, a_func func()) bool {
	if gp == nil {
		return false
	}
	if atomic.LoadInt32(&gp.state) == 0 {
		slog.Warn("协程池关闭了", zap.Any("a_shardingid=", a_shardingid))
		return false
	}
	index := a_shardingid % uint64(gp.coroutine_num)
	select {
	case <-gp.done:
		slog.Warn("协程池关闭了", zap.Any("a_shardingid=", a_shardingid))
		return false
	case gp.func_pool_slice[index] <- a_func:
		return true
	default:
		return false
	}
}
