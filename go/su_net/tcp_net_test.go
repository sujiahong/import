package su_net

import (
	"go.local/su_util"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCreateTcpServerAndClient(t *testing.T) {
	got := make(chan DataProtocol, 1)

	server, err := CreateTcpServer("127.0.0.1:0")
	if err != nil {
		t.Fatalf("CreateTcpServer() error = %v", err)
	}
	defer server.Close()
	if err := server.RegisterRequestResponseHandler(10, 11, func(ctx *HandlerContext, req []byte) error {
		ctx.SetResponse(append([]byte(nil), req...))
		return nil
	}); err != nil {
		t.Fatalf("server RegisterRequestResponseHandler() error = %v", err)
	}

	client, err := CreateTcpClient(server.Addr)
	if err != nil {
		t.Fatalf("CreateTcpClient() error = %v", err)
	}
	defer client.Close()
	if err := client.RegisterOneWayHandler(11, func(ctx *HandlerContext, req []byte) error {
		dp := *ctx.Packet
		dp.Data = append([]byte(nil), req...)
		got <- dp
		return nil
	}); err != nil {
		t.Fatalf("RegisterOneWayHandler() error = %v", err)
	}

	err = client.Send(&DataProtocol{
		Head: Header{
			PackId:   10,
			RouteId:  20,
			HeadUuid: 30,
		},
		Data: []byte("ping"),
	})
	if err != nil {
		t.Fatalf("client Send() error = %v", err)
	}

	select {
	case dp := <-got:
		if dp.Head.PackId != 11 || dp.Head.RouteId != 20 || dp.Head.HeadUuid != 30 {
			t.Fatalf("response head = %+v", dp.Head)
		}
		if string(dp.Data) != "ping" {
			t.Fatalf("response data = %q, want ping", dp.Data)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for tcp response")
	}
}

func TestTcpClientConnectionPoolRoundRobin(t *testing.T) {
	server, err := CreateTcpServer("127.0.0.1:0")
	if err != nil {
		t.Fatalf("CreateTcpServer() error = %v", err)
	}
	defer server.Close()

	var mu sync.Mutex
	seen := make(map[*TcpConn]int)
	typeErr := make(chan any, 1)
	if err := server.RegisterRequestResponseHandler(40, 41, func(ctx *HandlerContext, req []byte) error {
		conn, ok := ctx.Conn.(*TcpConn)
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

	client, err := CreateTcpClient(server.Addr, 3)
	if err != nil {
		t.Fatalf("CreateTcpClient() error = %v", err)
	}
	defer client.Close()
	if client.ConnCount() != 3 {
		t.Fatalf("client ConnCount() = %d, want 3", client.ConnCount())
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if server.ConnCount() == 3 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if server.ConnCount() != 3 {
		t.Fatalf("server ConnCount() = %d, want 3", server.ConnCount())
	}

	got := make(chan struct{}, 6)
	if err := client.RegisterOneWayHandler(41, func(ctx *HandlerContext, req []byte) error {
		got <- struct{}{}
		return nil
	}); err != nil {
		t.Fatalf("client RegisterOneWayHandler() error = %v", err)
	}
	for i := 0; i < 6; i++ {
		if err := client.Send(&DataProtocol{Head: Header{PackId: 40, RouteId: uint64(i + 1)}, Data: []byte("pool")}); err != nil {
			t.Fatalf("client Send(%d) error = %v", i, err)
		}
	}
	for i := 0; i < 6; i++ {
		select {
		case <-got:
		case conn := <-typeErr:
			t.Fatalf("ctx.Conn type = %T, want *TcpConn", conn)
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for tcp pool response")
		}
	}

	mu.Lock()
	seenCount := len(seen)
	mu.Unlock()
	if seenCount != 3 {
		t.Fatalf("server saw %d tcp conns, want 3", seenCount)
	}
}

func TestTcpWriteTimeoutConfigAndUpdate(t *testing.T) {
	gotServerTimeout := make(chan time.Duration, 1)
	server, err := CreateTcpServerWithConfig("127.0.0.1:0", TcpNetConfig{WriteTimeout: 0})
	if err != nil {
		t.Fatalf("CreateTcpServerWithConfig() error = %v", err)
	}
	defer server.Close()
	if err := server.RegisterOneWayHandler(10, func(ctx *HandlerContext, req []byte) error {
		conn, ok := ctx.Conn.(*TcpConn)
		if !ok {
			gotServerTimeout <- -1
			return nil
		}
		gotServerTimeout <- conn.WriteTimeout()
		return nil
	}); err != nil {
		t.Fatalf("server RegisterOneWayHandler() error = %v", err)
	}
	if server.WriteTimeout() != 0 {
		t.Fatalf("server WriteTimeout() = %s, want 0", server.WriteTimeout())
	}

	client, err := CreateTcpClientWithConfig(server.Addr, TcpNetConfig{WriteTimeout: 0})
	if err != nil {
		t.Fatalf("CreateTcpClientWithConfig() error = %v", err)
	}
	defer client.Close()
	if client.WriteTimeout() != 0 {
		t.Fatalf("client WriteTimeout() = %s, want 0", client.WriteTimeout())
	}
	if client.Conn.WriteTimeout() != 0 {
		t.Fatalf("client conn WriteTimeout() = %s, want 0", client.Conn.WriteTimeout())
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if server.ConnCount() == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if server.ConnCount() != 1 {
		t.Fatalf("server ConnCount() = %d, want 1", server.ConnCount())
	}

	server.SetWriteTimeout(123 * time.Millisecond)
	client.SetWriteTimeout(456 * time.Millisecond)
	if client.WriteTimeout() != 456*time.Millisecond {
		t.Fatalf("client WriteTimeout() = %s, want 456ms", client.WriteTimeout())
	}
	if client.Conn.WriteTimeout() != 456*time.Millisecond {
		t.Fatalf("client conn WriteTimeout() = %s, want 456ms", client.Conn.WriteTimeout())
	}

	if err := client.Send(&DataProtocol{Head: Header{PackId: 10, RouteId: 20}, Data: []byte("timeout")}); err != nil {
		t.Fatalf("client Send() error = %v", err)
	}
	select {
	case timeout := <-gotServerTimeout:
		if timeout != 123*time.Millisecond {
			t.Fatalf("server conn WriteTimeout() = %s, want 123ms", timeout)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for tcp server handler")
	}
}

func TestTcpPingPong(t *testing.T) {
	server, err := CreateTcpServer("127.0.0.1:0")
	if err != nil {
		t.Fatalf("CreateTcpServer() error = %v", err)
	}
	defer server.Close()

	client, err := CreateTcpClient(server.Addr)
	if err != nil {
		t.Fatalf("CreateTcpClient() error = %v", err)
	}
	defer client.Close()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if server.ConnCount() == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if server.ConnCount() != 1 {
		t.Fatalf("server ConnCount() = %d, want 1", server.ConnCount())
	}

	if err := client.Conn.Ping(); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}

	for time.Now().Before(deadline) {
		count := 0
		client.Conn.PingPongMap.Range(func(k, v interface{}) bool {
			count++
			return true
		})
		if count == 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timeout waiting for pong")
}

func TestTcpClientHeartbeatStopsOnRemoteClose(t *testing.T) {
	server, err := CreateTcpServer("127.0.0.1:0")
	if err != nil {
		t.Fatalf("CreateTcpServer() error = %v", err)
	}
	client, err := CreateTcpClient(server.Addr)
	if err != nil {
		t.Fatalf("CreateTcpClient() error = %v", err)
	}
	defer client.Close()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if server.ConnCount() == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := server.Close(); err != nil {
		t.Fatalf("server Close() error = %v", err)
	}

	select {
	case <-client.done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for tcp client heartbeat to stop")
	}
}

func TestTcpClientReconnectRestoresSend(t *testing.T) {
	got := make(chan DataProtocol, 1)
	server, err := CreateTcpServer("127.0.0.1:0")
	if err != nil {
		t.Fatalf("CreateTcpServer() error = %v", err)
	}
	defer server.Close()
	if err := server.RegisterRequestResponseHandler(30, 31, func(ctx *HandlerContext, req []byte) error {
		ctx.SetResponse(append([]byte(nil), req...))
		return nil
	}); err != nil {
		t.Fatalf("server RegisterRequestResponseHandler() error = %v", err)
	}

	client, err := CreateTcpClient(server.Addr)
	if err != nil {
		t.Fatalf("CreateTcpClient() error = %v", err)
	}
	defer client.Close()
	if err := client.RegisterOneWayHandler(31, func(ctx *HandlerContext, req []byte) error {
		dp := *ctx.Packet
		dp.Data = append([]byte(nil), req...)
		got <- dp
		return nil
	}); err != nil {
		t.Fatalf("RegisterOneWayHandler() error = %v", err)
	}

	if err := client.Reconnect(); err != nil {
		t.Fatalf("Reconnect() error = %v", err)
	}
	if err := client.Send(&DataProtocol{Head: Header{PackId: 30, RouteId: 31}, Data: []byte("reconnect")}); err != nil {
		t.Fatalf("Send() after reconnect error = %v", err)
	}
	select {
	case dp := <-got:
		if dp.Head.PackId != 31 || string(dp.Data) != "reconnect" {
			t.Fatalf("response = %+v data=%q", dp.Head, dp.Data)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for tcp reconnect response")
	}
}

func TestTcpServerCloseDrainsQueuedPoolTasks(t *testing.T) {
	server, err := CreateTcpServer("127.0.0.1:0")
	if err != nil {
		t.Fatalf("CreateTcpServer() error = %v", err)
	}
	server.pool.Stop()
	server.pool = su_util.NewGoPool(1, 16)

	block := make(chan struct{})
	ran := make(chan struct{})
	if ok := server.pool.SendTask(1, func() { <-block }); !ok {
		t.Fatal("blocking task should be accepted")
	}
	if ok := server.pool.SendTask(1, func() { close(ran) }); !ok {
		t.Fatal("queued task should be accepted")
	}

	closeDone := make(chan error, 1)
	go func() {
		closeDone <- server.Close()
	}()
	close(block)

	select {
	case <-ran:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for queued task to run during Close drain")
	}
	select {
	case err := <-closeDone:
		if err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for server Close")
	}
}

func TestTcpServerConditionalDeleteDoesNotRemoveReplacedConn(t *testing.T) {
	server := &TcpServer{conns: make(map[string]*TcpConn)}
	key := "127.0.0.1:12345"
	oldConn := &TcpConn{}
	newConn := &TcpConn{}

	server.connsMu.Lock()
	server.conns[key] = oldConn
	server.conns[key] = newConn
	if server.conns[key] == oldConn {
		delete(server.conns, key)
	}
	server.connsMu.Unlock()

	server.connsMu.Lock()
	current, ok := server.conns[key]
	server.connsMu.Unlock()
	if !ok {
		t.Fatal("connection key was deleted")
	}
	if current != newConn {
		t.Fatalf("current conn = %p, want %p", current, newConn)
	}
}

func TestTcpConnCloseClearsHeartbeat(t *testing.T) {
	conn := newTcpConn(nil)
	conn.PingPongMap.Store(uint64(1), 1)
	if err := conn.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	count := 0
	conn.PingPongMap.Range(func(k, v interface{}) bool {
		count++
		return true
	})
	if count != 0 {
		t.Fatalf("heartbeat count = %d, want 0", count)
	}
}

func TestTcpConnCheckPongAfterCloseDoesNotPing(t *testing.T) {
	conn := newTcpConn(nil)
	if err := conn.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	conn.CheckPong()
	count := 0
	conn.PingPongMap.Range(func(k, v interface{}) bool {
		count++
		return true
	})
	if count != 0 {
		t.Fatalf("heartbeat count = %d, want 0", count)
	}
	if err := conn.Send(&DataProtocol{}); err == nil {
		t.Fatal("Send() error = nil, want closed connection error")
	}
}

func TestTcpConnPongDecrementsPendingPingCount(t *testing.T) {
	conn := newTcpConn(nil)
	conn.PingPongMap.Store(uint64(7), 1)
	atomic.StoreInt32(&conn.pendingPings, 1)
	data, err := PongEncode(Pong{SendTime: 8, PingTime: 7})
	if err != nil {
		t.Fatalf("PongEncode() error = %v", err)
	}

	handled, err := conn.handleControlPacket(&DataProtocol{
		Head: Header{PackId: PONG, PackLen: HeadLength + uint32(len(data))},
		Data: data,
	})
	if err != nil {
		t.Fatalf("handleControlPacket() error = %v", err)
	}
	if !handled {
		t.Fatal("handled = false, want true")
	}
	if got := atomic.LoadInt32(&conn.pendingPings); got != 0 {
		t.Fatalf("pendingPings = %d, want 0", got)
	}
}

func TestTcpConnRecvControlPacketAfterCloseReturnsNil(t *testing.T) {
	conn := newTcpConn(nil)
	if err := conn.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	pingBytes, err := PingEncode(Ping{SendTime: 1})
	if err != nil {
		t.Fatalf("PingEncode() error = %v", err)
	}
	packet, err := Encode(&DataProtocol{
		Head: Header{PackId: PING, RouteId: 1, HeadUuid: 1},
		Data: pingBytes,
	})
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	if err := conn.recv(packet, nil); err != nil {
		t.Fatalf("recv() error = %v, want nil during close", err)
	}
}
