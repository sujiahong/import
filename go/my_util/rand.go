package my_util

import (
	"math/rand"
	"sync"
	"time"
)

var (
	r   = rand.New(rand.NewSource(time.Now().UnixNano()))
	mux sync.Mutex
)

func Int63n(n int64) int64 {
	if n <= 0 {
		return 0
	}
	mux.Lock()
	res := r.Int63n(n)
	mux.Unlock()
	return res
}

func Intn(n int) int {
	if n <= 0 {
		return 0
	}
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

// RandRange returns a concurrency-safe random integer in [min, max].
func RandRange(min, max int64) int64 {
	return SafeRandRange(min, max)
}

func RandShuffle(length int, list []int) {
	if length <= 1 || len(list) <= 1 {
		return
	}
	if length > len(list) {
		length = len(list)
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(length, func(i, j int) {
		list[i], list[j] = list[j], list[i]
	})
}

func RandElementsUnordered(slice []uint64, n int) ([]uint64, []uint64) {
	if n <= 0 {
		return slice, nil
	}
	m := len(slice)
	if n > m {
		n = m
	}
	if n == 0 {
		return slice, nil
	}
	randSlice := make([]uint64, 0, n)
	for i := 0; i < n; i++ {
		j := i + Intn(m-i)
		slice[i], slice[j] = slice[j], slice[i]
		randSlice = append(randSlice, slice[i])
	}
	newSlice := slice[n:]
	return newSlice, randSlice
}
