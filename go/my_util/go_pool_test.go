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

func TestGoPoolStopAndDrainRunsQueuedTasks(t *testing.T) {
	gp := NewGoPool(1, 16)
	block := make(chan struct{})
	var mu sync.Mutex
	ran := make(map[int]bool)

	if ok := gp.SendTask(1, func() { <-block }); !ok {
		t.Fatal("blocking task should be accepted")
	}
	for i := 0; i < 5; i++ {
		i := i
		if ok := gp.SendTask(1, func() {
			mu.Lock()
			ran[i] = true
			mu.Unlock()
		}); !ok {
			t.Fatalf("queued task %d should be accepted", i)
		}
	}

	drained := make(chan bool, 1)
	go func() {
		drained <- gp.StopAndDrain(time.Second)
	}()
	close(block)

	select {
	case ok := <-drained:
		if !ok {
			t.Fatal("StopAndDrain() = false, want true")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for StopAndDrain")
	}

	mu.Lock()
	defer mu.Unlock()
	for i := 0; i < 5; i++ {
		if !ran[i] {
			t.Fatalf("queued task %d was not run", i)
		}
	}
}

func TestGoPoolStopAndDrainTimeout(t *testing.T) {
	gp := NewGoPool(1, 1)
	block := make(chan struct{})
	defer close(block)

	if ok := gp.SendTask(1, func() { <-block }); !ok {
		t.Fatal("blocking task should be accepted")
	}

	start := time.Now()
	if ok := gp.StopAndDrain(20 * time.Millisecond); ok {
		t.Fatal("StopAndDrain() = true, want false for blocked task")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("StopAndDrain took %s, want bounded timeout", elapsed)
	}
}
