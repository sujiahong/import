package su_net

import (
	"testing"
	"time"
)

func TestCreateTcpServerAndClient(t *testing.T) {
	got := make(chan DataProtocol, 1)

	server, err := CreateTcpServer("127.0.0.1:0", func(conn *TcpConn, dp *DataProtocol) {
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
		t.Fatalf("CreateTcpServer() error = %v", err)
	}
	defer server.Close()

	client, err := CreateTcpClient(server.Addr, func(conn *TcpConn, dp *DataProtocol) {
		got <- *dp
	})
	if err != nil {
		t.Fatalf("CreateTcpClient() error = %v", err)
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
		t.Fatal("timeout waiting for tcp response")
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
