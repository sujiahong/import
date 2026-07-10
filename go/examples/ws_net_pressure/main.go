package main

import (
	"flag"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang/protobuf/proto"
	testpb "go.local/proto/Test"
	"go.local/su_net"
)

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
	addr := flag.String("addr", "127.0.0.1:10040", "listen/connect address")
	clientCount := flag.Int("clients", 32, "number of websocket clients")
	senderCount := flag.Int("senders", 1, "number of request sender goroutines")
	requests := flag.Int("requests", 1000000, "total requests")
	inflight := flag.Int("inflight", 4096, "maximum in-flight requests")
	payloadBytes := flag.Int("payload", 32, "request payload bytes")
	timeout := flag.Duration("timeout", 120*time.Second, "wait timeout")
	writeTimeout := flag.Duration("write-timeout", 0, "per-message write timeout; 0 disables SetWriteDeadline")
	flag.Parse()

	if *clientCount <= 0 {
		panic("clients must be > 0")
	}
	if *senderCount <= 0 {
		panic("senders must be > 0")
	}
	if *requests <= 0 {
		panic("requests must be > 0")
	}
	if *inflight <= 0 {
		panic("inflight must be > 0")
	}

	payload := strings.Repeat("x", *payloadBytes)
	cfg := su_net.WSNetConfig{WriteTimeout: *writeTimeout}
	server, err := su_net.CreateWSServerWithConfig(*addr, cfg, func(conn *su_net.WSConn, dp *su_net.DataProtocol) {
		rq := &testpb.TestRQ{}
		if err := proto.Unmarshal(dp.Data, rq); err != nil {
			panic(err)
		}
		rsBytes, err := proto.Marshal(&testpb.TestRS{
			Test1: proto.Uint32(rq.GetTest1()),
			Test2: proto.String(rq.GetTest2()),
		})
		if err != nil {
			panic(err)
		}
		err = conn.Send(&su_net.DataProtocol{
			Head: su_net.Header{
				PackId:   dp.Head.PackId + 1,
				RouteId:  dp.Head.RouteId,
				HeadUuid: dp.Head.HeadUuid,
			},
			Data: rsBytes,
		})
		if err != nil {
			panic(err)
		}
	})
	if err != nil {
		panic(err)
	}
	defer server.Close()

	var received uint64
	done := make(chan struct{})
	sem := make(chan struct{}, *inflight)
	var doneOnce sync.Once
	clients := make([]*su_net.WSClient, 0, *clientCount)
	for i := 0; i < *clientCount; i++ {
		client, err := su_net.CreateWSClientWithConfig(server.Addr, cfg, func(conn *su_net.WSConn, dp *su_net.DataProtocol) {
			rs := &testpb.TestRS{}
			if err := proto.Unmarshal(dp.Data, rs); err != nil {
				panic(err)
			}
			<-sem
			if atomic.AddUint64(&received, 1) == uint64(*requests) {
				doneOnce.Do(func() { close(done) })
			}
		})
		if err != nil {
			panic(err)
		}
		clients = append(clients, client)
	}
	defer func() {
		for _, client := range clients {
			client.Close()
		}
	}()

	if err := waitUntil("clients connect", 3*time.Second, func() bool {
		return server.ConnCount() >= *clientCount
	}); err != nil {
		panic(err)
	}

	start := time.Now()
	var sendWG sync.WaitGroup
	for i := 0; i < *senderCount; i++ {
		senderID := i
		sendWG.Add(1)
		go func() {
			defer sendWG.Done()
			for idx := senderID; idx < *requests; idx += *senderCount {
				sem <- struct{}{}
				rqBytes, err := proto.Marshal(&testpb.TestRQ{
					Test1: proto.Uint32(uint32(idx)),
					Test2: proto.String(payload),
				})
				if err != nil {
					<-sem
					panic(err)
				}
				err = clients[idx%len(clients)].Send(&su_net.DataProtocol{
					Head: su_net.Header{
						PackId:   10000,
						RouteId:  uint64(idx + 1),
						HeadUuid: uint64(idx + 1),
					},
					Data: rqBytes,
				})
				if err != nil {
					<-sem
					panic(err)
				}
			}
		}()
	}
	sendWG.Wait()

	select {
	case <-done:
	case <-time.After(*timeout):
		panic(fmt.Sprintf("timeout waiting for responses: received=%d want=%d", atomic.LoadUint64(&received), *requests))
	}
	elapsed := time.Since(start)
	rps := float64(*requests) / elapsed.Seconds()
	avgLatency := elapsed / time.Duration(*requests)
	fmt.Printf("clients=%d senders=%d requests=%d inflight=%d payload=%d write_timeout=%s gomaxprocs=%d elapsed=%s throughput=%.0f req/s avg_roundtrip=%s received=%d\n",
		*clientCount, *senderCount, *requests, *inflight, *payloadBytes, *writeTimeout, runtime.GOMAXPROCS(0), elapsed, rps, avgLatency, atomic.LoadUint64(&received))
}
