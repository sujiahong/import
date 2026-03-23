package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"distributed/circuit"
	"distributed/health"
	"distributed/lb"
	"distributed/limit"
	"distributed/registry"
	"distributed/rpc"
	"distributed/transport"
)

type Args struct {
	A int `json:"a"`
	B int `json:"b"`
}

type Reply struct {
	Result int `json:"result"`
}

type MathService struct{}

func (m *MathService) Add(ctx context.Context, args *Args) (*Reply, error) {
	return &Reply{Result: args.A + args.B}, nil
}

func (m *MathService) Multiply(ctx context.Context, args *Args) (*Reply, error) {
	return &Reply{Result: args.A * args.B}, nil
}

func startServer(addr string, serviceName string, id string) error {
	rpcServer := rpc.NewRPCServer(addr)

	rpcServer.Register("Add", func(ctx context.Context, args interface{}) (interface{}, error) {
		a := args.(*Args)
		return &Reply{Result: a.A + a.B}, nil
	})

	rpcServer.Register("Multiply", func(ctx context.Context, args interface{}) (interface{}, error) {
		a := args.(*Args)
		return &Reply{Result: a.A * a.B}, nil
	})

	if err := rpcServer.Start(""); err != nil {
		return err
	}

	registryInstance := registry.NewInMemoryRegistry()
	instance := &registry.ServiceInstance{
		ID:       id,
		Name:     serviceName,
		Address:  "127.0.0.1",
		Port:     8080,
		Metadata: map[string]string{"version": "1.0.0"},
	}

	client := registry.NewRegistryClient(registryInstance, instance)
	go func() {
		if err := client.RegisterAndKeepalive(context.Background()); err != nil {
			log.Printf("Registry error: %v", err)
		}
	}()

	log.Printf("Server started at %s", rpcServer.Addr())
	log.Printf("Service %s instance %s registered", serviceName, id)

	return nil
}

func clientExample() {
	registryInstance := registry.NewInMemoryRegistry()

	instance1 := &registry.ServiceInstance{
		ID:      "instance-1",
		Name:    "math-service",
		Address: "127.0.0.1",
		Port:    8080,
	}
	instance2 := &registry.ServiceInstance{
		ID:      "instance-2",
		Name:    "math-service",
		Address: "127.0.0.1",
		Port:    8081,
	}

	registryInstance.Register(context.Background(), instance1)
	registryInstance.Register(context.Background(), instance2)

	balancer := lb.NewRoundRobinBalancer()

	breaker := circuit.NewCircuit(3, 10*time.Second)
	breaker.OnOpen(func() {
		log.Println("Circuit breaker opened!")
	})
	breaker.OnClose(func() {
		log.Println("Circuit breaker closed!")
	})

	limiter := limit.NewTokenBucketLimiter(100, 100)

	checker := health.NewTCPHealthChecker(2 * time.Second)
	healthManager := health.NewHealthCheckManager(checker, 5*time.Second)
	healthManager.AddInstance("instance-1", "127.0.0.1", 8080)
	healthManager.AddInstance("instance-2", "127.0.0.1", 8081)
	healthManager.Start(context.Background())

	for i := 0; i < 10; i++ {
		go func(i int) {
			ctx := context.Background()

			allowed, err := limiter.Allow(ctx)
			if err != nil || !allowed {
				log.Printf("Request %d: Rate limited", i)
				return
			}

			instances, err := registryInstance.Discover(ctx, "math-service")
			if err != nil {
				log.Printf("Request %d: Discover error - %v", i, err)
				return
			}

			instance, err := balancer.Select(ctx, instances)
			if err != nil {
				log.Printf("Request %d: Load balance error - %v", i, err)
				return
			}

			err = breaker.Execute(ctx, func() error {
				log.Printf("Request %d: Calling %s:%d", i, instance.Address, instance.Port)
				return nil
			})
			if err != nil {
				log.Printf("Request %d: Circuit breaker error - %v", i, err)
			}
		}(i)
	}

	time.Sleep(2 * time.Second)
	healthManager.Stop()
}

func transportExample() {
	ln, err := net.Listen("tcp", "127.0.0.1:9000")
	if err != nil {
		log.Fatal(err)
	}

	// Custom server that echoes messages
	server := transport.NewServer("127.0.0.1:9000", func(conn net.Conn) *transport.Transport {
		tp := transport.NewTransport(conn, &transport.LengthPrefixEncoder{}, &transport.LengthPrefixEncoder{})

		// Start a goroutine to echo messages for this connection
		go func() {
			for {
				msg, err := tp.Recv(context.Background())
				if err != nil {
					return
				}
				// Echo back
				tp.Send(context.Background(), msg)
			}
		}()

		return tp
	})
	// We manually set listener because NewServer tries to Listen again which would fail or we need to pass address.
	// Actually NewServer calls Listen. So we should close our temp listener or just let NewServer do it.
	// But NewServer takes addr string.
	ln.Close() // Close our check listener

	if err := server.Start(); err != nil {
		log.Fatal(err)
	}

	log.Println("Transport server started")

	client, err := transport.NewClient("127.0.0.1:9000")
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Transport client connected")

	// Send multiple messages to test stream handling
	for i := 0; i < 5; i++ {
		msg := &transport.Message{
			Header: map[string]string{"type": "request", "id": fmt.Sprintf("%d", i)},
			Body:   []byte(fmt.Sprintf("Message %d", i)),
		}
		if err := client.Send(context.Background(), msg); err != nil {
			log.Fatal(err)
		}
		log.Printf("Sent message %d", i)
	}

	for i := 0; i < 5; i++ {
		log.Printf("Waiting for response %d", i)
		resp, err := client.Recv(context.Background())
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Received response: %s (ID: %s)", string(resp.Body), resp.Header["id"])
	}

	client.Close()
	server.Stop()
}

func rpcClientServerExample() {
	server := rpc.NewRPCServer(":9001")

	server.Register("Add", func(ctx context.Context, args interface{}) (interface{}, error) {
		a := args.(rpc.Args)
		return &Reply{Result: a.A + a.B}, nil
	})

	server.Register("Multiply", func(ctx context.Context, args interface{}) (interface{}, error) {
		a := args.(rpc.Args)
		return &Reply{Result: a.A * a.B}, nil
	})

	server.Register("Panic", func(ctx context.Context, args interface{}) (interface{}, error) {
		panic("something went wrong")
	})

	go func() {
		if err := server.Start(""); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	client, err := rpc.NewRPCClient("127.0.0.1:9001", 5*time.Second)
	if err != nil {
		log.Fatalf("Client connect error: %v", err)
	}
	defer client.Close()

	args := &Args{A: 10, B: 20}
	reply := &Reply{}

	err = client.Call(context.Background(), "Add", args, reply)
	if err != nil {
		log.Printf("Call error: %v", err)
	} else {
		log.Printf("Add result: %d", reply.Result)
	}

	args = &Args{A: 5, B: 6}
	reply = &Reply{}

	err = client.Call(context.Background(), "Multiply", args, reply)
	if err != nil {
		log.Printf("Call error: %v", err)
	} else {
		log.Printf("Multiply result: %d", reply.Result)
	}

	// Test Panic
	err = client.Call(context.Background(), "Panic", args, reply)
	if err != nil {
		log.Printf("Panic test (expected error): %v", err)
	} else {
		log.Printf("Panic test failed (unexpected success)")
	}

	server.Stop()
}

func main() {
	log.Println("=== RPC Client/Server Example ===")
	rpcClientServerExample()

	time.Sleep(1 * time.Second)

	log.Println("\n=== Client Example with All Components ===")
	clientExample()

	time.Sleep(1 * time.Second)

	log.Println("\n=== Transport Example ===")
	transportExample()
}
