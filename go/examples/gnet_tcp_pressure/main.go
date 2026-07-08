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
	mode := flag.String("mode", "proto", "protocol mode: proto or raw")
	pending := flag.Bool("pending", true, "track proto requests in client pending map")
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
	var server *su_net.GTcpServer
	if *mode == "raw" {
		server = su_net.CreateGNetRawServer(serverListenPort(*addr), func(gnc *su_net.GNetConn, dp *su_net.DataProtocol) {
			_ = gnc.SendPacket(&su_net.DataProtocol{
				Head: su_net.Header{PackId: 10001, RouteId: dp.Head.RouteId, HeadUuid: dp.Head.HeadUuid},
				Data: dp.Data,
			})
		})
		server.SetDispatchMode(su_net.GNetDispatchInline)
	} else {
		server = su_net.CreateServer(serverListenPort(*addr))
		server.RegisterHandler(10000, &testpb.TestRQ{}, 10001, &testpb.TestRS{}, func(gnc *su_net.GNetConn, shardingID uint64, rqMsg proto.Message, rsMsg proto.Message) {
			rq := rqMsg.(*testpb.TestRQ)
			rs := rsMsg.(*testpb.TestRS)
			rs.Test1 = proto.Uint32(rq.GetTest1())
			rs.Test2 = proto.String(rq.GetTest2())
		})
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
		var client *su_net.GTcpClient
		if *mode == "raw" {
			client = su_net.CreateGNetRawClient(*addr, uint8(connNum), func(gnc *su_net.GNetConn, dp *su_net.DataProtocol) {
				<-sem
				if atomic.AddUint64(&received, 1) == uint64(*requests) {
					doneOnce.Do(func() { close(done) })
				}
			})
		} else {
			client = su_net.CreateClient(*addr, uint8(connNum))
		}
		if client == nil {
			panic("create client failed")
		}
		if *mode != "raw" {
			client.SetPendingRequestsEnabled(*pending)
			client.RegisterHandler(10000, &testpb.TestRQ{}, 10001, &testpb.TestRS{}, func(gnc *su_net.GNetConn, shardingID uint64, rqMsg proto.Message, rsMsg proto.Message) {
				<-sem
				if atomic.AddUint64(&received, 1) == uint64(*requests) {
					doneOnce.Do(func() { close(done) })
				}
			})
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
		if *mode == "raw" {
			if err := clients[i%len(clients)].SendPacket(&su_net.DataProtocol{
				Head: su_net.Header{PackId: 10000, RouteId: uint64(i + 1), HeadUuid: uint64(i + 1)},
				Data: []byte(payload),
			}); err != nil {
				panic(err)
			}
		} else if !*pending {
			if err := clients[i%len(clients)].SendNoPending(10000, 10001, &testpb.TestRQ{
				Test1: proto.Uint32(uint32(i)),
				Test2: proto.String(payload),
			}); err != nil {
				panic(err)
			}
		} else {
			clients[i%len(clients)].Send(10000, 10001, &testpb.TestRQ{
				Test1: proto.Uint32(uint32(i)),
				Test2: proto.String(payload),
			})
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
	fmt.Printf("mode=%s pending=%t clients=%d engines=%d requests=%d inflight=%d payload=%d gomaxprocs=%d elapsed=%s throughput=%.0f req/s avg_roundtrip=%s received=%d\n",
		*mode, *pending, *clientCount, *engineCount, *requests, *inflight, *payloadBytes, runtime.GOMAXPROCS(0), elapsed, rps, avgLatency, atomic.LoadUint64(&received))
}
