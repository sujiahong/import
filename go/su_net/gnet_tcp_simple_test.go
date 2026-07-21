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

func TestGNetTcpSimpleClientServer(t *testing.T) {
	port := freeTCPPort(t)
	server := CreateServer(port)
	if err := server.RegisterRequestResponseHandler(10000, 10001, func(ctx *HandlerContext, req []byte) error {
		rq := &testpb.TestRQ{}
		if err := proto.Unmarshal(req, rq); err != nil {
			return err
		}
		rsBytes, err := proto.Marshal(&testpb.TestRS{
			Test1:  proto.Uint32(rq.GetTest1() + 1),
			Test2:  proto.String(fmt.Sprintf("echo:%s", rq.GetTest2())),
			Test3S: []uint64{ctx.Packet.Head.RouteId},
		})
		if err != nil {
			return err
		}
		ctx.SetResponse(rsBytes)
		return nil
	}); err != nil {
		t.Fatalf("server RegisterRequestResponseHandler() error = %v", err)
	}
	go server.Run()
	defer server.Close()
	waitForState(t, "server", &server.Stat, 2)

	got := make(chan *testpb.TestRS, 1)
	client, err := CreateGNetClient("127.0.0.1:"+port, 1)
	if err != nil {
		t.Fatalf("CreateGNetClient() error = %v", err)
	}
	defer client.Stop()
	if err := client.RegisterOneWayHandler(10001, func(ctx *HandlerContext, req []byte) error {
		rs := &testpb.TestRS{}
		if err := proto.Unmarshal(req, rs); err != nil {
			return err
		}
		got <- rs
		return nil
	}); err != nil {
		t.Fatalf("client RegisterOneWayHandler() error = %v", err)
	}
	waitForState(t, "client", &client.state, 2)

	rqBytes, err := proto.Marshal(&testpb.TestRQ{
		Test1: proto.Uint32(41),
		Test2: proto.String("hello"),
	})
	if err != nil {
		t.Fatalf("proto.Marshal() error = %v", err)
	}
	err = client.Send(&DataProtocol{
		Head: Header{PackId: 10000, RouteId: nextRouteID(), HeadUuid: uint64(time.Now().UnixNano() / 1000)},
		Data: rqBytes,
	})
	if err != nil {
		t.Fatalf("client Send() error = %v", err)
	}

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

func TestGNetClientRejectsInvalidConnNum(t *testing.T) {
	if client, err := CreateGNetClient("127.0.0.1:9999", 0); err == nil {
		if client != nil {
			client.Stop()
		}
		t.Fatal("CreateGNetClient() with zero conn num error = nil")
	}
}

func TestGNetClientWithConfigAppliesOptions(t *testing.T) {
	port := freeTCPPort(t)
	server := CreateServer(port)
	go server.Run()
	defer server.Close()
	waitForState(t, "server", &server.Stat, 2)

	client, err := CreateGNetClientWithConfig("127.0.0.1:"+port, GNetTcpConfig{
		DispatchMode:      GNetDispatchPool,
		ReconnectInterval: 25 * time.Millisecond,
	}, 1)
	if err != nil {
		t.Fatalf("CreateGNetClientWithConfig() error = %v", err)
	}
	defer client.Stop()
	waitForState(t, "client", &client.state, 2)

	if client.dispatchMode != GNetDispatchPool {
		t.Fatalf("dispatchMode = %d, want %d", client.dispatchMode, GNetDispatchPool)
	}
	if client.reconnectInterval != 25*time.Millisecond {
		t.Fatalf("reconnectInterval = %s, want 25ms", client.reconnectInterval)
	}
}

func TestGNetClientConnectionPoolRoundRobin(t *testing.T) {
	port := freeTCPPort(t)
	server := CreateServer(port)
	server.SetDispatchMode(GNetDispatchInline)
	var mu sync.Mutex
	seen := make(map[*GNetConn]int)
	typeErr := make(chan any, 1)
	if err := server.RegisterRequestResponseHandler(30000, 30001, func(ctx *HandlerContext, req []byte) error {
		conn, ok := ctx.Conn.(*GNetConn)
		if !ok {
			select {
			case typeErr <- ctx.Conn:
			default:
			}
			ctx.SetResponse(append([]byte(nil), req...))
			return nil
		}
		mu.Lock()
		seen[conn]++
		mu.Unlock()
		ctx.SetResponse(append([]byte(nil), req...))
		return nil
	}); err != nil {
		t.Fatalf("server RegisterRequestResponseHandler() error = %v", err)
	}
	go server.Run()
	defer server.Close()
	waitForState(t, "server", &server.Stat, 2)

	client, err := CreateGNetClient("127.0.0.1:"+port, 3)
	if err != nil {
		t.Fatalf("CreateGNetClient() error = %v", err)
	}
	defer client.Stop()
	waitForCondition(t, "client pool", 3*time.Second, func() bool {
		return client.State() == 2 && client.ConnCount() == 3 && server.ConnCount() == 3
	})

	got := make(chan struct{}, 6)
	if err := client.RegisterOneWayHandler(30001, func(ctx *HandlerContext, req []byte) error {
		got <- struct{}{}
		return nil
	}); err != nil {
		t.Fatalf("client RegisterOneWayHandler() error = %v", err)
	}
	for i := 0; i < 6; i++ {
		if err := client.Send(&DataProtocol{
			Head: Header{PackId: 30000, RouteId: uint64(i + 1), HeadUuid: uint64(i + 1)},
			Data: []byte("pool"),
		}); err != nil {
			t.Fatalf("client Send(%d) error = %v", i, err)
		}
	}
	for i := 0; i < 6; i++ {
		select {
		case <-got:
		case conn := <-typeErr:
			t.Fatalf("ctx.Conn type = %T, want *GNetConn", conn)
		case <-time.After(3 * time.Second):
			t.Fatal("timeout waiting for gnet pool response")
		}
	}

	mu.Lock()
	seenCount := len(seen)
	mu.Unlock()
	if seenCount != 3 {
		t.Fatalf("server saw %d gnet conns, want 3", seenCount)
	}
}

func TestGNetTcpBinaryPayloadClientServer(t *testing.T) {
	port := freeTCPPort(t)
	server := CreateServer(port)
	if err := server.RegisterRequestResponseHandler(20000, 20001, func(ctx *HandlerContext, req []byte) error {
		ctx.SetResponse(append([]byte(nil), req...))
		return nil
	}); err != nil {
		t.Fatalf("server RegisterRequestResponseHandler() error = %v", err)
	}
	server.SetDispatchMode(GNetDispatchInline)
	go server.Run()
	defer server.Close()
	waitForState(t, "server", &server.Stat, 2)

	got := make(chan DataProtocol, 1)
	client, err := CreateGNetClient("127.0.0.1:"+port, 1)
	if err != nil {
		t.Fatalf("CreateGNetClient() error = %v", err)
	}
	defer client.Stop()
	if err := client.RegisterOneWayHandler(20001, func(ctx *HandlerContext, req []byte) error {
		dp := *ctx.Packet
		dp.Data = append([]byte(nil), req...)
		got <- dp
		return nil
	}); err != nil {
		t.Fatalf("client RegisterOneWayHandler() error = %v", err)
	}
	waitForState(t, "client", &client.state, 2)

	err = client.Send(&DataProtocol{
		Head: Header{PackId: 20000, RouteId: 1, HeadUuid: 2},
		Data: []byte("binary"),
	})
	if err != nil {
		t.Fatalf("client Send() error = %v", err)
	}

	select {
	case dp := <-got:
		if dp.Head.PackId != 20001 || dp.Head.RouteId != 1 || dp.Head.HeadUuid != 2 {
			t.Fatalf("response head = %+v", dp.Head)
		}
		if string(dp.Data) != "binary" {
			t.Fatalf("response data = %q, want binary", dp.Data)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for gnet response")
	}
}

func TestGNetClientSendReturnsNoActiveConnection(t *testing.T) {
	client := &GTcpClient{dataHandler: newTcpNetHandler()}
	err := client.Send(&DataProtocol{Head: Header{PackId: 10000}, Data: []byte("payload")})
	if err == nil {
		t.Fatal("Send() error = nil, want no active connection error")
	}
}

func TestGNetClientHandlePacketDispatchesRegisteredHandler(t *testing.T) {
	client := &GTcpClient{dataHandler: newTcpNetHandler()}
	var seen []uint32
	if err := client.RegisterOneWayHandler(10001, func(ctx *HandlerContext, req []byte) error {
		rs := &testpb.TestRS{}
		if err := proto.Unmarshal(req, rs); err != nil {
			return err
		}
		seen = append(seen, rs.GetTest1())
		return nil
	}); err != nil {
		t.Fatalf("RegisterOneWayHandler() error = %v", err)
	}
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
	if seen[0] != 1 || seen[1] != 1 {
		t.Fatalf("response values = %v, want [1 1]", seen)
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
	if err := conn.Send(&DataProtocol{Head: Header{PackId: 1}}); err == nil {
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
	if err := conn.Send(&DataProtocol{Head: Header{PackId: 1}}); err == nil {
		t.Fatal("Send() error = nil, want closed error")
	}
}

func TestGNetConnCloseIsConcurrentSafe(t *testing.T) {
	port := freeTCPPort(t)
	server := CreateServer(port)
	go server.Run()
	defer server.Close()
	waitForState(t, "server", &server.Stat, 2)

	client, err := CreateGNetClient("127.0.0.1:"+port, 1)
	if err != nil {
		t.Fatalf("CreateGNetClient() error = %v", err)
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

func TestGNetClientStopIsConcurrentSafe(t *testing.T) {
	port := freeTCPPort(t)
	server := CreateServer(port)
	go server.Run()
	defer server.Close()
	waitForState(t, "server", &server.Stat, 2)

	client, err := CreateGNetClient("127.0.0.1:"+port, 1)
	if err != nil {
		t.Fatalf("CreateGNetClient() error = %v", err)
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
	if got := atomic.LoadInt32(&client.reconnecting); got != 0 {
		t.Fatalf("reconnecting = %d, want 0 after Stop", got)
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
	client := &GTcpClient{Addr: "127.0.0.1:" + port, state: 2}
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
