package su_util

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestWaitGroupWithTimeoutDoesNotBlockAfterTimeout(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)
	if err := WaitGroupWithTimeout(&wg, 1); err == nil {
		t.Fatal("expected timeout error")
	}
	done := make(chan struct{})
	go func() {
		wg.Done()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("waitgroup goroutine did not finish")
	}
}

func TestGetIncrUUIDConcurrent(t *testing.T) {
	atomic.StoreUint32(&incrUniqId, 0)
	const goroutines = 16
	const perGoroutine = 1000
	seen := sync.Map{}
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < perGoroutine; j++ {
				id := GetIncrUUID()
				if id == 0 {
					t.Errorf("id should not be zero")
				}
				if _, loaded := seen.LoadOrStore(id, struct{}{}); loaded {
					t.Errorf("duplicate id %d", id)
				}
			}
		}()
	}
	wg.Wait()
}

func TestCopyFileETruncatesDestination(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	if err := os.WriteFile(src, []byte("abc"), 0644); err != nil {
		t.Fatalf("write src failed: %v", err)
	}
	if err := os.WriteFile(dst, []byte("abcdef"), 0644); err != nil {
		t.Fatalf("write dst failed: %v", err)
	}
	if err := CopyFileE(src, dst); err != nil {
		t.Fatalf("copy failed: %v", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read dst failed: %v", err)
	}
	if string(got) != "abc" {
		t.Fatalf("dst = %q, want %q", got, "abc")
	}
}

func TestRandRangeConcurrent(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				v := RandRange(1, 3)
				if v < 1 || v > 3 {
					t.Errorf("RandRange = %d, want [1,3]", v)
				}
			}
		}()
	}
	wg.Wait()
}

func TestRandElementsUnorderedNoDuplicates(t *testing.T) {
	input := []uint64{1, 2, 3, 4, 5}
	rest, picked := RandElementsUnordered(input, 3)
	if len(picked) != 3 {
		t.Fatalf("picked len = %d, want 3", len(picked))
	}
	if len(rest) != 2 {
		t.Fatalf("rest len = %d, want 2", len(rest))
	}
	seen := map[uint64]bool{}
	for _, v := range append(rest, picked...) {
		if seen[v] {
			t.Fatalf("duplicate value %d", v)
		}
		seen[v] = true
	}
}
