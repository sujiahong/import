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
	if err := server.RegisterRequestResponseHandler(10000, 10001, func(ctx *su_net.HandlerContext, req []byte) error {
		rq := &testpb.TestRQ{}
		if err := proto.Unmarshal(req, rq); err != nil {
			return err
		}
		rsBytes, err := proto.Marshal(&testpb.TestRS{
			Test1:  proto.Uint32(rq.GetTest1() + 1),
			Test2:  proto.String("echo:" + rq.GetTest2()),
			Test3S: []uint64{ctx.Packet.Head.RouteId},
		})
		if err != nil {
			return err
		}
		ctx.SetResponse(rsBytes)
		return nil
	}); err != nil {
		panic(err)
	}
	server.Run()
}

func runClient(addr string) {
	client := su_net.CreateClient(addr, 1)
	if client == nil {
		panic("create gnet client failed")
	}
	if err := client.RegisterOneWayHandler(10001, func(ctx *su_net.HandlerContext, req []byte) error {
		rs := &testpb.TestRS{}
		if err := proto.Unmarshal(req, rs); err != nil {
			return err
		}
		fmt.Printf("response route=%d test1=%d test2=%q test3s=%v\n", ctx.Packet.Head.RouteId, rs.GetTest1(), rs.GetTest2(), rs.GetTest3S())
		return nil
	}); err != nil {
		panic(err)
	}
	time.Sleep(500 * time.Millisecond)
	rqBytes, err := proto.Marshal(&testpb.TestRQ{
		Test1: proto.Uint32(41),
		Test2: proto.String("hello"),
	})
	if err != nil {
		panic(err)
	}
	if err := client.Send(&su_net.DataProtocol{
		Head: su_net.Header{PackId: 10000, RouteId: uint64(time.Now().UnixNano()), HeadUuid: uint64(time.Now().UnixNano() / 1000)},
		Data: rqBytes,
	}); err != nil {
		panic(err)
	}
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
