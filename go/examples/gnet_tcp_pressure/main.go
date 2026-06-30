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
	clientCount := flag.Int("clients", 2, "number of gnet clients")
	requests := flag.Int("requests", 100000, "total requests")
	inflight := flag.Int("inflight", 4096, "maximum in-flight requests")
	payloadBytes := flag.Int("payload", 32, "request string payload bytes")
	timeout := flag.Duration("timeout", 30*time.Second, "wait timeout")
	flag.Parse()

	if *clientCount <= 0 {
		panic("clients must be > 0")
	}
	if *requests <= 0 {
		panic("requests must be > 0")
	}
	if *inflight <= 0 {
		panic("inflight must be > 0")
	}

	payload := strings.Repeat("x", *payloadBytes)
	server := su_net.CreateServer(serverListenPort(*addr))
	server.RegisterHandler(10000, &testpb.TestRQ{}, 10001, &testpb.TestRS{}, func(gnc *su_net.GNetConn, shardingID uint64, rqMsg proto.Message, rsMsg proto.Message) {
		rq := rqMsg.(*testpb.TestRQ)
		rs := rsMsg.(*testpb.TestRS)
		rs.Test1 = proto.Uint32(rq.GetTest1())
		rs.Test2 = proto.String(rq.GetTest2())
	})
	go server.Run()
	defer server.Close()
	if err := waitUntil("server start", 3*time.Second, func() bool { return server.State() == 2 }); err != nil {
		panic(err)
	}

	var received uint64
	done := make(chan struct{})
	sem := make(chan struct{}, *inflight)
	var doneOnce sync.Once
	clients := make([]*su_net.GTcpClient, 0, *clientCount)
	for i := 0; i < *clientCount; i++ {
		client := su_net.CreateClient(*addr, 1)
		if client == nil {
			panic("create client failed")
		}
		client.RegisterHandler(10000, &testpb.TestRQ{}, 10001, &testpb.TestRS{}, func(gnc *su_net.GNetConn, shardingID uint64, rqMsg proto.Message, rsMsg proto.Message) {
			<-sem
			if atomic.AddUint64(&received, 1) == uint64(*requests) {
				doneOnce.Do(func() { close(done) })
			}
		})
		clients = append(clients, client)
	}
	defer func() {
		for _, client := range clients {
			client.Stop()
		}
	}()

	for i, client := range clients {
		if err := waitUntil(fmt.Sprintf("client %d connect", i), 3*time.Second, func() bool {
			return client.State() == 2 && client.ConnCount() > 0
		}); err != nil {
			panic(err)
		}
	}

	start := time.Now()
	for i := 0; i < *requests; i++ {
		sem <- struct{}{}
		clients[i%len(clients)].Send(10000, 10001, &testpb.TestRQ{
			Test1: proto.Uint32(uint32(i)),
			Test2: proto.String(payload),
		})
	}

	select {
	case <-done:
	case <-time.After(*timeout):
		panic(fmt.Sprintf("timeout waiting for responses: received=%d want=%d", atomic.LoadUint64(&received), *requests))
	}
	elapsed := time.Since(start)
	rps := float64(*requests) / elapsed.Seconds()
	avgLatency := elapsed / time.Duration(*requests)
	fmt.Printf("clients=%d requests=%d inflight=%d payload=%d gomaxprocs=%d elapsed=%s throughput=%.0f req/s avg_roundtrip=%s received=%d\n",
		*clientCount, *requests, *inflight, *payloadBytes, runtime.GOMAXPROCS(0), elapsed, rps, avgLatency, atomic.LoadUint64(&received))
}
