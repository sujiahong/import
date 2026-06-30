package main

import (
	"flag"
	"fmt"
	"net"
	"time"

	"github.com/golang/protobuf/proto"
	testpb "go.local/proto/Test"
	"go.local/su_net"
)

func serverListenPort(addr string) string {
	_, port, err := net.SplitHostPort(addr)
	if err == nil {
		return port
	}
	return addr
}

func runServer(addr string) {
	server := su_net.CreateServer(serverListenPort(addr))
	server.RegisterHandler(10000, &testpb.TestRQ{}, 10001, &testpb.TestRS{}, func(gnc *su_net.GNetConn, shardingID uint64, rqMsg proto.Message, rsMsg proto.Message) {
		rq := rqMsg.(*testpb.TestRQ)
		rs := rsMsg.(*testpb.TestRS)
		rs.Test1 = proto.Uint32(rq.GetTest1() + 1)
		rs.Test2 = proto.String("echo:" + rq.GetTest2())
		rs.Test3S = []uint64{shardingID}
	})
	server.Run()
}

func runClient(addr string) {
	client := su_net.CreateClient(addr, 1)
	if client == nil {
		panic("create gnet client failed")
	}
	client.RegisterHandler(10000, &testpb.TestRQ{}, 10001, &testpb.TestRS{}, func(gnc *su_net.GNetConn, shardingID uint64, rqMsg proto.Message, rsMsg proto.Message) {
		rs := rsMsg.(*testpb.TestRS)
		fmt.Printf("response route=%d test1=%d test2=%q test3s=%v\n", shardingID, rs.GetTest1(), rs.GetTest2(), rs.GetTest3S())
	})
	time.Sleep(500 * time.Millisecond)
	client.Send(10000, 10001, &testpb.TestRQ{
		Test1: proto.Uint32(41),
		Test2: proto.String("hello"),
	})
	time.Sleep(2 * time.Second)
	client.Stop()
}

func main() {
	mode := flag.String("mode", "server", "server or client")
	addr := flag.String("addr", "127.0.0.1:9990", "server address for client, listen port/address for server")
	flag.Parse()

	switch *mode {
	case "server":
		runServer(*addr)
	case "client":
		runClient(*addr)
	default:
		panic("unknown mode: " + *mode)
	}
}
