package my_util

import (
	"sync"
	"testing"
	"time"
)

func TestGoPoolStopSendTaskRaceDoesNotPanic(t *testing.T) {
	for round := 0; round < 100; round++ {
		gp := NewGoPool(4, 16)
		panicCh := make(chan interface{}, 32)
		var wg sync.WaitGroup
		for i := 0; i < 16; i++ {
			wg.Add(1)
			go func(seed int) {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						panicCh <- r
					}
				}()
				for j := 0; j < 100; j++ {
					gp.SendTask(uint64(seed*100+j), func() {})
				}
			}(i)
		}
		time.Sleep(time.Microsecond)
		gp.Stop()
		wg.Wait()
		select {
		case p := <-panicCh:
			t.Fatalf("SendTask panicked after Stop: %v", p)
		default:
		}
	}
}

func TestGoPoolTrySendTaskReportsFullQueue(t *testing.T) {
	gp := NewGoPool(1, 0)
	defer gp.Stop()

	block := make(chan struct{})
	if ok := gp.SendTask(1, func() { <-block }); !ok {
		t.Fatal("first task should be accepted")
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if !gp.TrySendTask(1, func() {}) {
			close(block)
			return
		}
		time.Sleep(time.Millisecond)
	}
	close(block)
	t.Fatal("expected TrySendTask to report full queue")
}
