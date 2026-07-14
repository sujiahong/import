package main

import (
	"flag"
	"fmt"
	"net"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
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

func waitUntil(name string, timeout time.Duration, ok func() bool) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if ok() {
			return nil
		}
		time.Sleep(20 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for %s", name)
}

func main() {
	addr := flag.String("addr", "127.0.0.1:9992", "listen/connect address")
	clientCount := flag.Int("clients", 32, "total number of gnet client connections")
	engineCount := flag.Int("engines", 0, "number of gnet client engines; default equals clients")
	requests := flag.Int("requests", 100000, "total requests")
	inflight := flag.Int("inflight", 4096, "maximum in-flight requests")
	payloadBytes := flag.Int("payload", 32, "request string payload bytes")
	timeout := flag.Duration("timeout", 30*time.Second, "wait timeout")
	flag.Parse()

	if *clientCount <= 0 {
		panic("clients must be > 0")
	}
	if *engineCount == 0 {
		*engineCount = *clientCount
	}
	if *engineCount < 0 {
		panic("engines must be >= 0")
	}
	if *engineCount > *clientCount {
		panic("engines must be <= clients")
	}
	if *clientCount > 255*(*engineCount) {
		panic("clients per engine must be <= 255")
	}
	if *requests <= 0 {
		panic("requests must be > 0")
	}
	if *inflight <= 0 {
		panic("inflight must be > 0")
	}

	payload := strings.Repeat("x", *payloadBytes)
	server := su_net.CreateServer(serverListenPort(*addr))
	if err := server.RegisterRequestResponseHandler(10000, 10001, func(ctx *su_net.HandlerContext, req []byte) error {
		rq := &testpb.TestRQ{}
		if err := proto.Unmarshal(req, rq); err != nil {
			return err
		}
		rsBytes, err := proto.Marshal(&testpb.TestRS{
			Test1: proto.Uint32(rq.GetTest1()),
			Test2: proto.String(rq.GetTest2()),
		})
		if err != nil {
			return err
		}
		ctx.SetResponse(rsBytes)
		return nil
	}); err != nil {
		panic(err)
	}
	go server.Run()
	defer server.Close()
	if err := waitUntil("server start", 3*time.Second, func() bool { return server.State() == 2 }); err != nil {
		panic(err)
	}

	var received uint64
	done := make(chan struct{})
	sem := make(chan struct{}, *inflight)
	var doneOnce sync.Once
	clients := make([]*su_net.GTcpClient, 0, *engineCount)
	connTargets := make([]int, 0, *engineCount)
	baseConns := *clientCount / *engineCount
	extraConns := *clientCount % *engineCount
	for i := 0; i < *engineCount; i++ {
		connNum := baseConns
		if i < extraConns {
			connNum++
		}
		client := su_net.CreateClient(*addr, uint8(connNum))
		if client == nil {
			panic("create client failed")
		}
		if err := client.RegisterOneWayHandler(10001, func(ctx *su_net.HandlerContext, req []byte) error {
			rs := &testpb.TestRS{}
			if err := proto.Unmarshal(req, rs); err != nil {
				return err
			}
			<-sem
			if atomic.AddUint64(&received, 1) == uint64(*requests) {
				doneOnce.Do(func() { close(done) })
			}
			return nil
		}); err != nil {
			panic(err)
		}
		clients = append(clients, client)
		connTargets = append(connTargets, connNum)
	}
	defer func() {
		for _, client := range clients {
			client.Stop()
		}
	}()

	for i, client := range clients {
		if err := waitUntil(fmt.Sprintf("client %d connect", i), 3*time.Second, func() bool {
			return client.State() == 2 && client.ConnCount() >= connTargets[i]
		}); err != nil {
			panic(err)
		}
	}

	start := time.Now()
	for i := 0; i < *requests; i++ {
		sem <- struct{}{}
		rqBytes, err := proto.Marshal(&testpb.TestRQ{
			Test1: proto.Uint32(uint32(i)),
			Test2: proto.String(payload),
		})
		if err != nil {
			panic(err)
		}
		if err := clients[i%len(clients)].Send(&su_net.DataProtocol{
			Head: su_net.Header{PackId: 10000, RouteId: uint64(i + 1), HeadUuid: uint64(i + 1)},
			Data: rqBytes,
		}); err != nil {
			panic(err)
		}
	}

	select {
	case <-done:
	case <-time.After(*timeout):
		panic(fmt.Sprintf("timeout waiting for responses: received=%d want=%d", atomic.LoadUint64(&received), *requests))
	}
	elapsed := time.Since(start)
	rps := float64(*requests) / elapsed.Seconds()
	avgLatency := elapsed / time.Duration(*requests)
	fmt.Printf("clients=%d engines=%d requests=%d inflight=%d payload=%d gomaxprocs=%d elapsed=%s throughput=%.0f req/s avg_roundtrip=%s received=%d\n",
		*clientCount, *engineCount, *requests, *inflight, *payloadBytes, runtime.GOMAXPROCS(0), elapsed, rps, avgLatency, atomic.LoadUint64(&received))
}
