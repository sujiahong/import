package su_net

import (
	"sync"
	"testing"
)

func TestDeleteAllSyncMapRemovesEveryKey(t *testing.T) {
	var m sync.Map
	for i := 0; i < 16; i++ {
		m.Store(i, i)
	}

	deleteAllSyncMap(&m)

	count := 0
	m.Range(func(k, v interface{}) bool {
		count++
		return true
	})
	if count != 0 {
		t.Fatalf("map count = %d, want 0", count)
	}
}

func TestDeleteSyncMapValueOnlyDeletesMatchingCurrentValue(t *testing.T) {
	var m sync.Map
	key := "conn"
	oldConn := &TcpConn{}
	newConn := &TcpConn{}
	m.Store(key, newConn)

	if deleted := deleteSyncMapValue(&m, key, oldConn); deleted {
		t.Fatal("deleteSyncMapValue() = true for stale value, want false")
	}
	if current, ok := m.Load(key); !ok || current != newConn {
		t.Fatalf("current value = %v ok=%t, want new conn", current, ok)
	}

	if deleted := deleteSyncMapValue(&m, key, newConn); !deleted {
		t.Fatal("deleteSyncMapValue() = false for current value, want true")
	}
	if _, ok := m.Load(key); ok {
		t.Fatal("key still exists after deleting current value")
	}
}
