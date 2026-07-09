package su_util

import (
	slog "go.local/su_log"
	"go.uber.org/zap"
	"sync"
	"sync/atomic"
	"time"
)

// ///go协程池
type GoPool struct {
	func_pool_slice []chan func()
	coroutine_num   uint32
	cache_num       uint32
	state           int32 ////状态 0 停止 1 运行
	drain           int32
	done            chan struct{}
	stopOnce        sync.Once
	wg              sync.WaitGroup
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
		gp.wg.Add(1)
		go gp.run(i, gp.func_pool_slice[i])
	}
}

func (gp *GoPool) run(index uint32, a_func_ch chan func()) {
	defer gp.wg.Done()
	for {
		select {
		case <-gp.done:
			if atomic.LoadInt32(&gp.drain) == 1 {
				for {
					select {
					case f := <-a_func_ch:
						if f != nil {
							f()
						}
					default:
						slog.Warn("协程池关闭", zap.Any("index=", index))
						return
					}
				}
			}
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

func (gp *GoPool) StopAndDrain(timeout time.Duration) bool {
	if gp == nil {
		return true
	}
	gp.stopOnce.Do(func() {
		atomic.StoreInt32(&gp.drain, 1)
		atomic.StoreInt32(&gp.state, 0)
		close(gp.done)
	})
	if timeout <= 0 {
		gp.wg.Wait()
		return true
	}
	done := make(chan struct{})
	go func() {
		gp.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
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
