package su_net

import (
	"testing"
	"time"

	"go.local/su_util"
)

// TestGNetServerDispatchDoesNotBlockWhenPoolFull 验证 dispatch 在 worker 池满时
// 不会阻塞事件循环线程，而是丢弃任务立即返回。
func TestGNetServerDispatchDoesNotBlockWhenPoolFull(t *testing.T) {
	ts := &GTcpServer{
		dispatchMode: GNetDispatchPool,
		pool:         su_util.NewGoPool(1, 1),
		dataHandler:  newTcpNetHandler(),
	}
	defer ts.pool.Stop()

	block := make(chan struct{})
	defer close(block)
	if ok := ts.pool.SendTask(1, func() { <-block }); !ok {
		t.Fatal("blocking task should be accepted")
	}
	if ok := ts.pool.SendTask(1, func() {}); !ok {
		t.Fatal("queued task should be accepted")
	}

	done := make(chan struct{})
	go func() {
		ts.dispatch(&GNetConn{}, &DataProtocol{Head: Header{PackId: 42}, Data: []byte("x")})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("dispatch blocked on full worker pool")
	}
}

// TestGNetServerDispatchCopiesDataForPool 验证 Pool 模式下提交任务前必须复制
// Data，调用方在 dispatch 返回后修改原始数据不应影响已入队任务。
func TestGNetServerDispatchCopiesDataForPool(t *testing.T) {
	ts := &GTcpServer{
		dispatchMode: GNetDispatchPool,
		pool:         su_util.NewGoPool(2, 8),
		dataHandler:  newTcpNetHandler(),
	}
	defer ts.pool.Stop()

	started := make(chan struct{})
	release := make(chan struct{})
	got := make(chan []byte, 1)
	if err := ts.RegisterOneWayHandler(42, func(ctx *HandlerContext, msg []byte) error {
		close(started)
		<-release
		got <- append([]byte(nil), msg...)
		return nil
	}); err != nil {
		t.Fatalf("RegisterOneWayHandler() error = %v", err)
	}

	dp := &DataProtocol{Head: Header{PackId: 42, RouteId: 1}, Data: []byte("original")}
	ts.dispatch(&GNetConn{}, dp)
	<-started
	dp.Data[0] = 'X'
	close(release)

	select {
	case data := <-got:
		if string(data) != "original" {
			t.Fatalf("dispatched data = %q, want original (data must be copied for pool tasks)", data)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for dispatched task")
	}
}
