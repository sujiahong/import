package go_pool


type GoPool struct {
	func_pool_slice   []chan func()
	coroutine_num     uint32
	cache_num         uint32
	state             int     ////状态 0 停止 1 运行
}

func NewGoPool(a_go_num, a_cache_num uint32) *GoPool{
	gp := &GoPool{
		coroutine_num: a_go_num,
		cache_num: a_cache_num,
		state: 0,
	}
	gp.func_pool_slice = make([]chan func(), a_go_num)
	for i := 0; i < a_go_num; i ++ {
		gp.func_pool_slice[i] = make(chan func(), a_cache_num)
	}
	gp.Start()
	gp.state = 1
	return gp
}

func (gp *GoPool)Start() {
	for i := 0; i < gp.coroutine_num; i++ {
		go gp.Run(i, gp.func_pool_slice[i])
	}
}

func (gp *GoPool)Run(index uint32, a_func_ch chan func()){
	for f := range a_func_ch {
		if gp.state == 0 {
			return
		}
		f()
	}
}

func (gp *GoPool)Stop() {
	gp.state = 0
	for i := 0; i < gp.coroutine_num; i++ {
		Close(gp.func_pool_slice[i])
	}
}

func (gp *GoPool)SendTask(a_shardingid uint64, a_func func()){
	if gp.state == 0 {
		return
	}
	index := a_shardingid % gp.coroutine_num;
	gp.func_pool_slice[index] <- a_func
}