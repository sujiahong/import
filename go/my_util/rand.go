package my_util

import (
	"math/rand"
	"sync"
	"time"
)

var (
	r  = rand.New(rand.NewSource(time.Now().UnixNano()))
	mux sync.Mutex
)

func Int63n(n int64) int64 {
	mux.Lock()
	res := r.Int63n(n)
	mux.Unlock()
	return res
}

func Intn(n int) int {
	mux.Lock()
	res := r.Intn(n)
	mux.Unlock()
	return res
}

func IntRange(floor, ceiling int) int {
	if floor == ceiling {
		return floor
	} else if floor > ceiling {
		floor, ceiling = ceiling, floor
	}
	return Intn(ceiling-floor) + floor
}

func Float64() float64 {
	mux.Lock()
	res := r.Float64()
	mux.Unlock()
	return res
}

func SafeRandRange(min, max int64) int64 {
	if max < min {
		return 0
	}
	mux.Lock()
	res := r.Int63()%(max-min+1) + min
	mux.Unlock()
	return res
}

///////////////////////////////////////////////非协程安全的范围随机
/////[min,max]范围内随机
func RandRange(min, max int64) int64 {
	if max < min {
		return 0
	}
	return rand.Int63()%(max-min+1) + min
}