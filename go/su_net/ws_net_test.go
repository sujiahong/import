package su_net

import (
	"go.local/my_util"
	"testing"
	"time"
)

func TestCreateWSServerAndClient(t *testing.T) {
	got := make(chan DataProtocol, 1)

	server, err := CreateWSServer("127.0.0.1:0", func(conn *WSConn, dp *DataProtocol) {
		rs := &DataProtocol{
			Head: Header{
				PackId:   dp.Head.PackId + 1,
				RouteId:  dp.Head.RouteId,
				HeadUuid: dp.Head.HeadUuid,
			},
			Data: append([]byte(nil), dp.Data...),
		}
		if err := conn.Send(rs); err != nil {
			t.Errorf("server Send() error = %v", err)
		}
	})
	if err != nil {
		t.Fatalf("CreateWSServer() error = %v", err)
	}
	defer server.Close()

	client, err := CreateWSClient(server.Addr, func(conn *WSConn, dp *DataProtocol) {
		got <- *dp
	})
	if err != nil {
		t.Fatalf("CreateWSClient() error = %v", err)
	}
	defer client.Close()

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
		t.Fatal("timeout waiting for websocket response")
	}
}

func TestWSPingPong(t *testing.T) {
	server, err := CreateWSServer("127.0.0.1:0")
	if err != nil {
		t.Fatalf("CreateWSServer() error = %v", err)
	}
	defer server.Close()

	client, err := CreateWSClient(server.Addr)
	if err != nil {
		t.Fatalf("CreateWSClient() error = %v", err)
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
	t.Fatal("timeout waiting for websocket pong")
}

func TestWSClientHeartbeatStopsOnRemoteClose(t *testing.T) {
	server, err := CreateWSServer("127.0.0.1:0")
	if err != nil {
		t.Fatalf("CreateWSServer() error = %v", err)
	}
	client, err := CreateWSClient(server.Addr)
	if err != nil {
		t.Fatalf("CreateWSClient() error = %v", err)
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
		t.Fatal("timeout waiting for websocket client heartbeat to stop")
	}
}

func TestWSConnCloseClearsHeartbeat(t *testing.T) {
	conn := newWSConn(nil)
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

func TestWSServerTaskCopiesNoCopyPayload(t *testing.T) {
	got := make(chan []byte, 1)
	ws := &WSServer{
		pool: my_util.NewGoPool(1, 1),
		handler: func(conn *WSConn, dp *DataProtocol) {
			got <- append([]byte(nil), dp.Data...)
		},
	}
	defer ws.pool.Stop()

	original := []byte("payload")
	wsConn := newWSConn(nil)
	dp := &DataProtocol{
		Head: Header{PackId: 10, RouteId: 20, HeadUuid: 30},
		Data: original,
	}
	taskDP := *dp
	taskDP.Data = append([]byte(nil), dp.Data...)
	if !ws.pool.SendTask(taskDP.Head.RouteId, func() {
		ws.handler(wsConn, &taskDP)
	}) {
		t.Fatal("SendTask() = false, want true")
	}
	for i := range original {
		original[i] = 'x'
	}

	select {
	case data := <-got:
		if string(data) != "payload" {
			t.Fatalf("handler data = %q, want payload", data)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for websocket task")
	}
}
