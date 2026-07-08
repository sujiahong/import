package su_net

import (
	"errors"
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

type jitterTCPProxy struct {
	listener  net.Listener
	target    string
	chunkSize int
	delay     time.Duration
	closeOnce sync.Once
	conns     sync.Map
}

func newJitterTCPProxy(t *testing.T, target string, chunkSize int, delay time.Duration) *jitterTCPProxy {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	proxy := &jitterTCPProxy{
		listener:  ln,
		target:    target,
		chunkSize: chunkSize,
		delay:     delay,
	}
	go proxy.acceptLoop(t)
	return proxy
}

func (p *jitterTCPProxy) addr() string {
	return p.listener.Addr().String()
}

func (p *jitterTCPProxy) close() {
	p.closeOnce.Do(func() {
		p.listener.Close()
		p.conns.Range(func(k, v interface{}) bool {
			if conn, ok := v.(net.Conn); ok {
				conn.Close()
			}
			return true
		})
	})
}

func (p *jitterTCPProxy) acceptLoop(t *testing.T) {
	for {
		clientConn, err := p.listener.Accept()
		if err != nil {
			if !errors.Is(err, net.ErrClosed) {
				t.Logf("jitter proxy accept error: %v", err)
			}
			return
		}
		serverConn, err := net.Dial("tcp", p.target)
		if err != nil {
			clientConn.Close()
			t.Logf("jitter proxy dial target error: %v", err)
			continue
		}
		p.conns.Store(clientConn, clientConn)
		p.conns.Store(serverConn, serverConn)
		go p.copyWithJitter(clientConn, serverConn)
		go p.copyWithJitter(serverConn, clientConn)
	}
}

func (p *jitterTCPProxy) copyWithJitter(dst net.Conn, src net.Conn) {
	defer func() {
		dst.Close()
		src.Close()
		p.conns.Delete(dst)
		p.conns.Delete(src)
	}()
	buf := make([]byte, p.chunkSize)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			if p.delay > 0 {
				time.Sleep(p.delay)
			}
			if _, writeErr := dst.Write(buf[:n]); writeErr != nil {
				return
			}
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return
			}
			return
		}
	}
}

func TestTcpNetThroughJitterProxy(t *testing.T) {
	got := make(chan DataProtocol, 32)
	server, err := CreateTcpServer("127.0.0.1:0", func(conn *TcpConn, dp *DataProtocol) {
		if err := conn.Send(&DataProtocol{
			Head: Header{PackId: dp.Head.PackId + 1, RouteId: dp.Head.RouteId, HeadUuid: dp.Head.HeadUuid},
			Data: append([]byte(nil), dp.Data...),
		}); err != nil {
			t.Errorf("server Send() error = %v", err)
		}
	})
	if err != nil {
		t.Fatalf("CreateTcpServer() error = %v", err)
	}
	defer server.Close()

	proxy := newJitterTCPProxy(t, server.Addr, 3, time.Millisecond)
	defer proxy.close()

	client, err := CreateTcpClient(proxy.addr(), func(conn *TcpConn, dp *DataProtocol) {
		got <- *dp
	})
	if err != nil {
		t.Fatalf("CreateTcpClient() error = %v", err)
	}
	defer client.Close()

	const requests = 20
	for i := 0; i < requests; i++ {
		err := client.Send(&DataProtocol{
			Head: Header{PackId: 10, RouteId: uint64(i + 1), HeadUuid: uint64(100 + i)},
			Data: []byte("tcp-jitter"),
		})
		if err != nil {
			t.Fatalf("client Send(%d) error = %v", i, err)
		}
	}

	received := make(map[uint64]bool, requests)
	deadline := time.After(5 * time.Second)
	for len(received) < requests {
		select {
		case dp := <-got:
			if dp.Head.PackId != 11 {
				t.Fatalf("PackId = %d, want 11", dp.Head.PackId)
			}
			if string(dp.Data) != "tcp-jitter" {
				t.Fatalf("Data = %q, want tcp-jitter", dp.Data)
			}
			received[dp.Head.RouteId] = true
		case <-deadline:
			t.Fatalf("timeout waiting for tcp jitter responses: got %d want %d", len(received), requests)
		}
	}
}

func TestWSNetThroughJitterProxy(t *testing.T) {
	got := make(chan DataProtocol, 32)
	server, err := CreateWSServer("127.0.0.1:0", func(conn *WSConn, dp *DataProtocol) {
		if err := conn.Send(&DataProtocol{
			Head: Header{PackId: dp.Head.PackId + 1, RouteId: dp.Head.RouteId, HeadUuid: dp.Head.HeadUuid},
			Data: append([]byte(nil), dp.Data...),
		}); err != nil {
			t.Errorf("server Send() error = %v", err)
		}
	})
	if err != nil {
		t.Fatalf("CreateWSServer() error = %v", err)
	}
	defer server.Close()

	proxy := newJitterTCPProxy(t, server.Addr, 5, time.Millisecond)
	defer proxy.close()

	client, err := CreateWSClient(proxy.addr(), func(conn *WSConn, dp *DataProtocol) {
		got <- *dp
	})
	if err != nil {
		t.Fatalf("CreateWSClient() error = %v", err)
	}
	defer client.Close()

	const requests = 20
	for i := 0; i < requests; i++ {
		err := client.Send(&DataProtocol{
			Head: Header{PackId: 20, RouteId: uint64(i + 1), HeadUuid: uint64(200 + i)},
			Data: []byte("ws-jitter"),
		})
		if err != nil {
			t.Fatalf("client Send(%d) error = %v", i, err)
		}
	}

	received := make(map[uint64]bool, requests)
	deadline := time.After(5 * time.Second)
	for len(received) < requests {
		select {
		case dp := <-got:
			if dp.Head.PackId != 21 {
				t.Fatalf("PackId = %d, want 21", dp.Head.PackId)
			}
			if string(dp.Data) != "ws-jitter" {
				t.Fatalf("Data = %q, want ws-jitter", dp.Data)
			}
			received[dp.Head.RouteId] = true
		case <-deadline:
			t.Fatalf("timeout waiting for websocket jitter responses: got %d want %d", len(received), requests)
		}
	}
}
