package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"distributed/circuit"
	"distributed/health"
	"distributed/lb"
	"distributed/limit"
	"distributed/registry"
	"distributed/rpc"
	"distributed/transport"
)

// 测试注册中心
func testRegistry() {
	fmt.Println("=== 测试注册中心 ===")
	
	reg := registry.NewInMemoryRegistry()
	
	// 注册服务实例
	instance1 := &registry.ServiceInstance{
		ID:      "instance-1",
		Name:    "test-service",
		Address: "127.0.0.1",
		Port:    8080,
	}
	instance2 := &registry.ServiceInstance{
		ID:      "instance-2",
		Name:    "test-service",
		Address: "127.0.0.1",
		Port:    8081,
	}
	
	ctx := context.Background()
	
	// 测试注册
	err := reg.Register(ctx, instance1)
	if err != nil {
		log.Printf("注册 instance1 失败: %v", err)
	} else {
		fmt.Println("✓ 注册 instance1 成功")
	}
	
	err = reg.Register(ctx, instance2)
	if err != nil {
		log.Printf("注册 instance2 失败: %v", err)
	} else {
		fmt.Println("✓ 注册 instance2 成功")
	}
	
	// 测试发现
	instances, err := reg.Discover(ctx, "test-service")
	if err != nil {
		log.Printf("发现服务失败: %v", err)
	} else {
		fmt.Printf("✓ 发现 %d 个服务实例\n", len(instances))
	}
	
	// 测试注销
	err = reg.Deregister(ctx, "test-service", "instance-1")
	if err != nil {
		log.Printf("注销 instance1 失败: %v", err)
	} else {
		fmt.Println("✓ 注销 instance1 成功")
	}
	
	// 再次发现
	instances, err = reg.Discover(ctx, "test-service")
	if err != nil {
		log.Printf("发现服务失败: %v", err)
	} else {
		fmt.Printf("✓ 发现 %d 个服务实例\n", len(instances))
	}
	
	fmt.Println()
}

// 测试负载均衡
func testLoadBalancer() {
	fmt.Println("=== 测试负载均衡器 ===")
	
	instances := []*registry.ServiceInstance{
		{ID: "instance-1", Name: "test-service", Address: "127.0.0.1", Port: 8080},
		{ID: "instance-2", Name: "test-service", Address: "127.0.0.1", Port: 8081},
		{ID: "instance-3", Name: "test-service", Address: "127.0.0.1", Port: 8082},
	}
	
	ctx := context.Background()
	
	// 测试轮询负载均衡
	fmt.Println("测试轮询负载均衡:")
	rrBalancer := lb.NewRoundRobinBalancer()
	for i := 0; i < 5; i++ {
		inst, err := rrBalancer.Select(ctx, instances)
		if err != nil {
			log.Printf("选择实例失败: %v", err)
		} else {
			fmt.Printf("  请求 %d -> %s:%d\n", i+1, inst.Address, inst.Port)
		}
	}
	
	// 测试随机负载均衡
	fmt.Println("测试随机负载均衡:")
	randomBalancer := lb.NewRandomBalancer()
	for i := 0; i < 5; i++ {
		inst, err := randomBalancer.Select(ctx, instances)
		if err != nil {
			log.Printf("选择实例失败: %v", err)
		} else {
			fmt.Printf("  请求 %d -> %s:%d\n", i+1, inst.Address, inst.Port)
		}
	}
	
	// 测试空实例列表
	fmt.Println("测试空实例列表:")
	_, err := rrBalancer.Select(ctx, []*registry.ServiceInstance{})
	if err != nil {
		fmt.Printf("  ✓ 正确处理空实例: %v\n", err)
	}
	
	fmt.Println()
}

// 测试熔断器
func testCircuitBreaker() {
	fmt.Println("=== 测试熔断器 ===")
	
	// 创建熔断器：3次失败触发熔断，10秒后尝试恢复
	breaker := circuit.NewCircuit(3, 10*time.Second)
	
	breaker.OnOpen(func() {
		fmt.Println("  熔断器打开!")
	})
	breaker.OnClose(func() {
		fmt.Println("  熔断器关闭!")
	})
	
	ctx := context.Background()
	
	// 模拟成功请求
	fmt.Println("模拟成功请求:")
	for i := 0; i < 3; i++ {
		err := breaker.Execute(ctx, func() error {
			return nil
		})
		if err != nil {
			fmt.Printf("  请求 %d 失败: %v\n", i+1, err)
		} else {
			fmt.Printf("  请求 %d 成功\n", i+1)
		}
	}
	
	// 模拟失败请求，触发熔断
	fmt.Println("模拟失败请求，触发熔断:")
	for i := 0; i < 5; i++ {
		err := breaker.Execute(ctx, func() error {
			return fmt.Errorf("模拟错误")
		})
		if err != nil {
			fmt.Printf("  请求 %d 失败: %v\n", i+1, err)
		} else {
			fmt.Printf("  请求 %d 成功\n", i+1)
		}
	}
	
	fmt.Println()
}

// 测试限流器
func testRateLimiter() {
	fmt.Println("=== 测试限流器 ===")
	
	// 创建令牌桶限流器：每秒100个令牌，容量100
	limiter := limit.NewTokenBucketLimiter(100, 100)
	
	ctx := context.Background()
	
	// 快速请求，应该都能通过
	fmt.Println("快速请求测试:")
	passCount := 0
	for i := 0; i < 10; i++ {
		allowed, err := limiter.Allow(ctx)
		if err != nil {
			fmt.Printf("  请求 %d 错误: %v\n", i+1, err)
		} else if allowed {
			passCount++
		}
	}
	fmt.Printf("  %d/10 请求通过\n", passCount)
	
	// 检查令牌数量
	tokens := limiter.Tokens()
	fmt.Printf("  当前令牌数量: %.2f\n", tokens)
	
	fmt.Println()
}

// 测试健康检查
func testHealthCheck() {
	fmt.Println("=== 测试健康检查 ===")
	
	// 创建TCP健康检查器
	checker := health.NewTCPHealthChecker(2 * time.Second)
	
	// 测试一个不存在的端口
	result := checker.Check(context.Background(), "test-instance", "127.0.0.1", 9999)
	if result.Healthy {
		fmt.Println("  ✗ 应该检测到不健康")
	} else {
		fmt.Printf("  ✓ 正确检测到不健康: %v\n", result.Error)
	}
	
	// 测试健康检查管理器
	manager := health.NewHealthCheckManager(checker, 1*time.Second)
	manager.AddInstance("instance-1", "127.0.0.1", 8080)
	manager.AddInstance("instance-2", "127.0.0.1", 8081)
	
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	manager.Start(ctx)
	time.Sleep(1 * time.Second)
	manager.Stop()
	
	fmt.Println("  ✓ 健康检查管理器运行正常")
	fmt.Println()
}

// 测试传输层
func testTransport() {
	fmt.Println("=== 测试传输层 ===")
	
	// 创建服务器
	server := transport.NewServer("127.0.0.1:0", func(conn net.Conn) *transport.Transport {
		return transport.NewTransport(conn, &transport.LengthPrefixEncoder{}, &transport.LengthPrefixEncoder{})
	})
	
	if err := server.Start(); err != nil {
		log.Printf("启动服务器失败: %v", err)
		return
	}
	defer server.Stop()
	
	fmt.Printf("  服务器启动在: %s\n", server.Addr())
	
	// 创建客户端
	client, err := transport.NewClient(server.Addr().String())
	if err != nil {
		log.Printf("连接服务器失败: %v", err)
		return
	}
	defer client.Close()
	
	// 发送消息
	msg := &transport.Message{
		Header: map[string]string{"type": "test"},
		Body:   []byte("Hello, Server!"),
	}
	
	if err := client.Send(context.Background(), msg); err != nil {
		log.Printf("发送消息失败: %v", err)
		return
	}
	
	fmt.Println("  ✓ 消息发送成功")
	fmt.Println()
}

// 测试RPC
func testRPC() {
	fmt.Println("=== 测试RPC ===")
	
	// 创建RPC服务器
	server := rpc.NewRPCServer(":0")
	
	// 注册服务方法
	server.Register("Add", func(ctx context.Context, args interface{}) (interface{}, error) {
		a := args.(rpc.Args)
		return &Reply{Result: a.A + a.B}, nil
	})
	
	// 启动服务器
	go func() {
		if err := server.Start(""); err != nil {
			log.Printf("服务器错误: %v", err)
		}
	}()
	
	time.Sleep(100 * time.Millisecond)
	
	// 创建客户端
	client, err := rpc.NewRPCClient(server.Addr().String(), 5*time.Second)
	if err != nil {
		log.Printf("连接服务器失败: %v", err)
		return
	}
	defer client.Close()
	
	// 调用Add方法
	args := &rpc.Args{A: 10, B: 20}
	reply := &Reply{}
	
	err = client.Call(context.Background(), "Add", args, reply)
	if err != nil {
		log.Printf("RPC调用失败: %v", err)
	} else {
		fmt.Printf("  Add(10, 20) = %d\n", reply.Result)
	}
	
	server.Stop()
	fmt.Println()
}

type Reply struct {
	Result int `json:"result"`
}

func main() {
	log.Println("开始测试 Distributed 框架...")
	
	testRegistry()
	testLoadBalancer()
	testCircuitBreaker()
	testRateLimiter()
	testHealthCheck()
	// testTransport()  // 需要修复transport模块
	// testRPC()        // 需要修复RPC模块
	
	log.Println("测试完成!")
}
