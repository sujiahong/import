package my_util

import (
	slog "go/su_log"
	"go.uber.org/zap"
	"sync/atomic"
)
/////go协程池
type GoPool struct {
	func_pool_slice   []chan func()
	coroutine_num     uint32
	cache_num         uint32
	state             int32     ////状态 0 停止 1 运行
}

func NewGoPool(a_go_num, a_cache_num uint32) *GoPool{
	gp := &GoPool{
		coroutine_num: a_go_num,
		cache_num: a_cache_num,
		state: 0,
	}
	gp.func_pool_slice = make([]chan func(), a_go_num)
	var i uint32 = 0
	for i = 0; i < a_go_num; i ++ {
		gp.func_pool_slice[i] = make(chan func(), a_cache_num)
	}
	gp.Start()
	gp.state = 1
	return gp
}

func (gp *GoPool)Start() {
	var i uint32 = 0
	for i = 0; i < gp.coroutine_num; i++ {
		go gp.run(i, gp.func_pool_slice[i])
	}
}

func (gp *GoPool)run(index uint32, a_func_ch chan func()){
	for f := range a_func_ch {
		if atomic.LoadInt32(&gp.state) == 0 {
			slog.Warn("协程池关闭", zap.Any("index=", index))
			return
		}
		f()
	}
	slog.Warn("协程运行结束", zap.Any("index=", index))
}

func (gp *GoPool)Stop() {
	atomic.StoreInt32(&gp.state, 0)
	var i uint32 = 0
	for i = 0; i < gp.coroutine_num; i++ {
		close(gp.func_pool_slice[i])
	}
}

func (gp *GoPool)SendTask(a_shardingid uint64, a_func func()){
	if atomic.LoadInt32(&gp.state) == 0 {
		slog.Warn("协程池关闭了", zap.Any("a_shardingid=", a_shardingid))
		return
	}
	index := a_shardingid % uint64(gp.coroutine_num);
	slog.Info("go routine index", zap.Any("index=", index))
	gp.func_pool_slice[index] <- a_func
}