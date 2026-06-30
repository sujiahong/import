package su_net

import (
	"fmt"
	"net"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	testpb "go.local/proto/Test"
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
