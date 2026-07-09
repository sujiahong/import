package su_net

import (
	"fmt"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/panjf2000/gnet/v2"
	testpb "go.local/proto/Test"
	"go.local/su_util"
)

func freeTCPPort(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	defer ln.Close()
	_, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatalf("SplitHostPort() error = %v", err)
	}
	if _, err := strconv.Atoi(port); err != nil {
		t.Fatalf("port %q is invalid: %v", port, err)
	}
	return port
}

func waitForState(t *testing.T, name string, state *int32, want int32) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(state) == want {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("%s state = %d, want %d", name, atomic.LoadInt32(state), want)
}

func waitForCondition(t *testing.T, name string, timeout time.Duration, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if ok() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for %s", name)
}

func gnetPendingCount(client *GTcpClient) int {
	count := 0
	client.pendingRQMap.Range(func(k, v interface{}) bool {
		count++
		return true
	})
	return count
}

func TestGNetTcpSimpleClientServer(t *testing.T) {
	port := freeTCPPort(t)
	server := CreateServer(port)
	server.RegisterHandler(10000, &testpb.TestRQ{}, 10001, &testpb.TestRS{}, func(gnc *GNetConn, shardingID uint64, rqMsg proto.Message, rsMsg proto.Message) {
		rq := rqMsg.(*testpb.TestRQ)
		rs := rsMsg.(*testpb.TestRS)
		rs.Test1 = proto.Uint32(rq.GetTest1() + 1)
		rs.Test2 = proto.String(fmt.Sprintf("echo:%s", rq.GetTest2()))
		rs.Test3S = []uint64{shardingID}
	})
	go server.Run()
	defer server.Close()
	waitForState(t, "server", &server.Stat, 2)

	got := make(chan *testpb.TestRS, 1)
	client := CreateClient("127.0.0.1:"+port, 1)
	if client == nil {
		t.Fatal("CreateClient() returned nil")
	}
	defer client.Stop()
	client.RegisterHandler(10000, &testpb.TestRQ{}, 10001, &testpb.TestRS{}, func(gnc *GNetConn, shardingID uint64, rqMsg proto.Message, rsMsg proto.Message) {
		got <- rsMsg.(*testpb.TestRS)
	})
	waitForState(t, "client", &client.state, 2)

	client.Send(10000, 10001, &testpb.TestRQ{
		Test1: proto.Uint32(41),
		Test2: proto.String("hello"),
	})

	select {
	case rs := <-got:
		if rs.GetTest1() != 42 {
			t.Fatalf("Test1 = %d, want 42", rs.GetTest1())
		}
		if rs.GetTest2() != "echo:hello" {
			t.Fatalf("Test2 = %q, want echo:hello", rs.GetTest2())
		}
		if len(rs.GetTest3S()) != 1 || rs.GetTest3S()[0] == 0 {
			t.Fatalf("Test3S = %v, want one route id", rs.GetTest3S())
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for gnet response")
	}
}

func TestGNetTcpRawClientServer(t *testing.T) {
	port := freeTCPPort(t)
	server := CreateGNetRawServer(port, func(gnc *GNetConn, dp *DataProtocol) {
		err := gnc.SendPacket(&DataProtocol{
			Head: Header{
				PackId:   dp.Head.PackId + 1,
				RouteId:  dp.Head.RouteId,
				HeadUuid: dp.Head.HeadUuid,
			},
			Data: append([]byte(nil), dp.Data...),
		})
		if err != nil {
			t.Errorf("server SendPacket() error = %v", err)
		}
	})
	server.SetDispatchMode(GNetDispatchInline)
	go server.Run()
	defer server.Close()
	waitForState(t, "server", &server.Stat, 2)

	got := make(chan DataProtocol, 1)
	client := CreateGNetRawClient("127.0.0.1:"+port, 1, func(gnc *GNetConn, dp *DataProtocol) {
		got <- *dp
	})
	if client == nil {
		t.Fatal("CreateGNetRawClient() returned nil")
	}
	defer client.Stop()
	waitForState(t, "client", &client.state, 2)

	err := client.SendPacket(&DataProtocol{
		Head: Header{PackId: 20000, RouteId: 1, HeadUuid: 2},
		Data: []byte("raw"),
	})
	if err != nil {
		t.Fatalf("client SendPacket() error = %v", err)
	}

	select {
	case dp := <-got:
		if dp.Head.PackId != 20001 || dp.Head.RouteId != 1 || dp.Head.HeadUuid != 2 {
			t.Fatalf("response head = %+v", dp.Head)
		}
		if string(dp.Data) != "raw" {
			t.Fatalf("response data = %q, want raw", dp.Data)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for raw gnet response")
	}
}

func TestGNetClientSendErrorRejectsUnregisteredResponse(t *testing.T) {
	client := &GTcpClient{}
	err := client.SendError(10000, 10001, &testpb.TestRQ{})
	if err == nil {
		t.Fatal("SendError() error = nil, want unregistered response error")
	}
}

func TestGNetClientSendFailureClearsPendingRequest(t *testing.T) {
	client := &GTcpClient{}
	client.RegisterHandler(10000, &testpb.TestRQ{}, 10001, &testpb.TestRS{}, func(gnc *GNetConn, shardingID uint64, rqMsg proto.Message, rsMsg proto.Message) {})

	err := client.SendError(10000, 10001, &testpb.TestRQ{})
	if err == nil {
		t.Fatal("SendError() error = nil, want no active connection error")
	}
	if got := gnetPendingCount(client); got != 0 {
		t.Fatalf("pending count = %d, want 0 after send failure", got)
	}
}

func TestGNetClientCleanupExpiredPendingRequests(t *testing.T) {
	client := &GTcpClient{requestTimeout: time.Millisecond, pendingEnabled: 1}
	client.pendingRQMap.Store(uint64(1), &pendingGNetRequest{
		rq:        &testpb.TestRQ{},
		createdAt: time.Now().Add(-time.Second),
	})
	client.pendingRQMap.Store(uint64(2), &pendingGNetRequest{
		rq:        &testpb.TestRQ{},
		createdAt: time.Now(),
	})

	client.cleanupExpiredPendingRequests()

	if _, ok := client.pendingRQMap.Load(uint64(1)); ok {
		t.Fatal("expired pending request was not removed")
	}
	if _, ok := client.pendingRQMap.Load(uint64(2)); !ok {
		t.Fatal("fresh pending request was removed")
	}
}

func TestGNetClientDisablePendingClearsRequests(t *testing.T) {
	client := &GTcpClient{pendingEnabled: 1}
	client.pendingRQMap.Store(uint64(1), &pendingGNetRequest{
		rq:        &testpb.TestRQ{},
		createdAt: time.Now(),
	})

	client.SetPendingRequestsEnabled(false)

	if client.pendingRequestsEnabled() {
		t.Fatal("pendingRequestsEnabled() = true, want false")
	}
	if got := gnetPendingCount(client); got != 0 {
		t.Fatalf("pending count = %d, want 0", got)
	}
}

func TestGNetClientHandlePacketUsesFreshRequestWhenPendingDisabled(t *testing.T) {
	client := &GTcpClient{pendingEnabled: 0}
	template := &testpb.TestRQ{}
	var seen []uint32
	client.RegisterHandler(10000, template, 10001, &testpb.TestRS{}, func(gnc *GNetConn, shardingID uint64, rqMsg proto.Message, rsMsg proto.Message) {
		rq := rqMsg.(*testpb.TestRQ)
		seen = append(seen, rq.GetTest1())
		rq.Test1 = proto.Uint32(99)
	})
	rsBytes, err := proto.Marshal(&testpb.TestRS{Test1: proto.Uint32(1)})
	if err != nil {
		t.Fatalf("proto.Marshal() error = %v", err)
	}

	client.handleClientPacket(NewGnetConn(nil), &DataProtocol{
		Head: Header{PackId: 10001, RouteId: 1},
		Data: rsBytes,
	})
	client.handleClientPacket(NewGnetConn(nil), &DataProtocol{
		Head: Header{PackId: 10001, RouteId: 2},
		Data: rsBytes,
	})

	if len(seen) != 2 {
		t.Fatalf("handler calls = %d, want 2", len(seen))
	}
	if seen[0] != 0 || seen[1] != 0 {
		t.Fatalf("request values = %v, want fresh zero-value requests", seen)
	}
	if template.GetTest1() != 0 {
		t.Fatalf("template request was mutated: %d", template.GetTest1())
	}
}

func TestGNetPongDecrementsPendingPingCount(t *testing.T) {
	conn := NewGnetConn(nil)
	conn.PingPongMap.Store(uint64(7), 1)
	atomic.StoreInt32(&conn.pendingPings, 1)
	data, err := PongEncode(Pong{SendTime: 8, PingTime: 7})
	if err != nil {
		t.Fatalf("PongEncode() error = %v", err)
	}

	pongHandler(conn, &DataProtocol{
		Head: Header{PackId: PONG, PackLen: HeadLength + uint32(len(data))},
		Data: data,
	})

	if got := atomic.LoadInt32(&conn.pendingPings); got != 0 {
		t.Fatalf("pendingPings = %d, want 0", got)
	}
}

func TestGNetConnCheckPongAfterCloseDoesNotPing(t *testing.T) {
	conn := NewGnetConn(nil)
	conn.PingPongMap.Store(uint64(1), 1)
	atomic.StoreInt32(&conn.pendingPings, 1)

	conn.Close()
	conn.CheckPong()

	if got := atomic.LoadInt32(&conn.pendingPings); got != 0 {
		t.Fatalf("pendingPings = %d, want 0", got)
	}
	if err := conn.Send([]byte("x")); err == nil {
		t.Fatal("Send() error = nil, want closed error")
	}
}

func TestGNetConnMarkClosedPreventsPing(t *testing.T) {
	conn := NewGnetConn(nil)
	conn.PingPongMap.Store(uint64(1), 1)
	atomic.StoreInt32(&conn.pendingPings, 1)

	conn.markClosed()
	conn.CheckPong()

	if got := atomic.LoadInt32(&conn.pendingPings); got != 0 {
		t.Fatalf("pendingPings = %d, want 0", got)
	}
	if err := conn.Send([]byte("x")); err == nil {
		t.Fatal("Send() error = nil, want closed error")
	}
}

func TestGNetConnCloseIsConcurrentSafe(t *testing.T) {
	port := freeTCPPort(t)
	server := CreateGNetRawServer(port, func(gnc *GNetConn, dp *DataProtocol) {})
	go server.Run()
	defer server.Close()
	waitForState(t, "server", &server.Stat, 2)

	client := CreateGNetRawClient("127.0.0.1:"+port, 1, func(gnc *GNetConn, dp *DataProtocol) {})
	if client == nil {
		t.Fatal("CreateGNetRawClient() returned nil")
	}
	defer client.Stop()
	waitForState(t, "client", &client.state, 2)

	var target *GNetConn
	waitForCondition(t, "client conn", 3*time.Second, func() bool {
		client.connMu.RLock()
		defer client.connMu.RUnlock()
		if len(client.connList) == 0 {
			return false
		}
		target = client.connList[0]
		return true
	})

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			target.Close()
		}()
	}
	wg.Wait()

	waitForCondition(t, "client conn closed", 3*time.Second, func() bool {
		return client.ConnCount() == 0
	})
}

func TestGNetClientRemoteCloseClearsPendingRequests(t *testing.T) {
	port := freeTCPPort(t)
	server := CreateServer(port)
	go server.Run()
	defer server.Close()
	waitForState(t, "server", &server.Stat, 2)

	client := CreateClient("127.0.0.1:"+port, 1)
	if client == nil {
		t.Fatal("CreateClient() returned nil")
	}
	defer client.Stop()
	client.RegisterHandler(10000, &testpb.TestRQ{}, 10001, &testpb.TestRS{}, func(gnc *GNetConn, shardingID uint64, rqMsg proto.Message, rsMsg proto.Message) {})
	waitForState(t, "client", &client.state, 2)
	waitForCondition(t, "server connection", 3*time.Second, func() bool {
		return server.ConnCount() == 1
	})

	if err := client.SendError(10000, 10001, &testpb.TestRQ{Test1: proto.Uint32(1)}); err != nil {
		t.Fatalf("SendError() error = %v", err)
	}
	waitForCondition(t, "pending request", time.Second, func() bool {
		return gnetPendingCount(client) == 1
	})

	serverConns := make([]*GNetConn, 0)
	server.connMap.Range(func(k, v interface{}) bool {
		serverConns = append(serverConns, v.(*GNetConn))
		return true
	})
	for _, conn := range serverConns {
		conn.Close()
	}

	waitForCondition(t, "client remote close", 3*time.Second, func() bool {
		return client.ConnCount() == 0
	})
	if got := gnetPendingCount(client); got != 0 {
		t.Fatalf("pending count = %d, want 0 after remote close", got)
	}
}

func TestGNetClientStopIsConcurrentSafe(t *testing.T) {
	port := freeTCPPort(t)
	server := CreateServer(port)
	go server.Run()
	defer server.Close()
	waitForState(t, "server", &server.Stat, 2)

	client := CreateClient("127.0.0.1:"+port, 1)
	if client == nil {
		t.Fatal("CreateClient() returned nil")
	}
	waitForState(t, "client", &client.state, 2)

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := client.Stop(); err != nil {
				t.Errorf("Stop() error = %v", err)
			}
		}()
	}
	wg.Wait()

	if got := client.State(); got != 0 {
		t.Fatalf("client state = %d, want 0", got)
	}
	waitForCondition(t, "client connections closed after Stop", time.Second, func() bool {
		return client.ConnCount() == 0
	})
	if got := atomic.LoadInt32(&client.reconnectState); got != 0 {
		t.Fatalf("reconnectState = %d, want 0 after Stop", got)
	}
}

func TestGNetClientStopDrainsQueuedPoolTasks(t *testing.T) {
	client := &GTcpClient{state: 2, pool: su_util.NewGoPool(1, 16)}
	const tasks = 8
	var ran int32
	for i := 0; i < tasks; i++ {
		if !client.pool.SendTask(uint64(i), func() {
			time.Sleep(5 * time.Millisecond)
			atomic.AddInt32(&ran, 1)
		}) {
			t.Fatalf("SendTask(%d) failed", i)
		}
	}

	if err := client.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if got := atomic.LoadInt32(&ran); got != tasks {
		t.Fatalf("ran tasks = %d, want %d", got, tasks)
	}
}

func TestGNetClientConnectFailureRestoresConnectedState(t *testing.T) {
	port := freeTCPPort(t)
	client := &GTcpClient{remote_addr: "127.0.0.1:" + port, state: 2}
	gclient, err := gnet.NewClient(client)
	if err != nil {
		t.Fatalf("gnet.NewClient() error = %v", err)
	}
	if err := gclient.Start(); err != nil {
		t.Fatalf("gnet client Start() error = %v", err)
	}
	client.Client = gclient
	defer client.Stop()

	if err := client.Connect(); err == nil {
		t.Fatal("Connect() error = nil, want dial failure")
	}
	if got := client.State(); got != 2 {
		t.Fatalf("client state = %d, want 2 after dial failure", got)
	}
}

func TestGNetServerCloseIsConcurrentSafe(t *testing.T) {
	port := freeTCPPort(t)
	server := CreateServer(port)
	server.SetCloseTimeout(time.Second)
	go server.Run()
	waitForState(t, "server", &server.Stat, 2)

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			server.Close()
		}()
	}
	wg.Wait()

	if got := server.State(); got != 0 {
		t.Fatalf("server state = %d, want 0", got)
	}
}
