package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"go.uber.org/zap"
	"go/su_da/redis"
	slog "go/su_log"
)

// MCP 服务器结构
type MCPServer struct {
	addr         string
	listener     net.Listener
	redisClient  *redis.RedisClient
	handlers     map[string]HandlerFunc
	modelContext map[string]ModelContext // 模型上下文管理
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

// MCP 请求结构
type MCPRequest struct {
	ID      string          `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	Timeout time.Duration   `json:"timeout"`
}

// MCP 响应结构
type MCPResponse struct {
	ID     string          `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  string          `json:"error,omitempty"`
}

// 处理器函数类型
type HandlerFunc func(ctx context.Context, args json.RawMessage) (interface{}, error)

// Redis 命令参数结构
type RedisCommand struct {
	Command string        `json:"command"`
	Args    []interface{} `json:"args"`
}

// 模型上下文结构
type ModelContext struct {
	ID           string                 `json:"id"`
	ModelName    string                 `json:"model_name"`
	Parameters   map[string]interface{} `json:"parameters"`
	ContextData  map[string]interface{} `json:"context_data"`
	CreatedAt    time.Time              `json:"created_at"`
	LastAccessed time.Time              `json:"last_accessed"`
}

// 初始化 MCP 服务器
func NewMCPServer(addr string, redisAddr string, redisConnNum int) *MCPServer {
	ctx, cancel := context.WithCancel(context.Background())

	// 初始化 Redis 客户端
	redisClient := redis.NewRedisClient(redisAddr, redisConnNum)
	redisClient.Connect()

	return &MCPServer{
		addr:         addr,
		redisClient:  redisClient,
		handlers:     make(map[string]HandlerFunc),
		modelContext: make(map[string]ModelContext),
		ctx:          ctx,
		cancel:       cancel,
	}
}

// 注册处理器
func (s *MCPServer) Register(name string, handler HandlerFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[name] = handler
}

// 处理连接
func (s *MCPServer) handleConnection(conn net.Conn) {
	defer conn.Close()
	defer s.wg.Done()

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		var req MCPRequest
		if err := decoder.Decode(&req); err != nil {
			if err == io.EOF {
				return
			}
			slog.Error("Failed to decode request", zap.Error(err))
			continue
		}

		// 处理请求
		go func() {
			resp := s.handleRequest(req)
			if err := encoder.Encode(resp); err != nil {
				slog.Error("Failed to encode response", zap.Error(err))
			}
		}()
	}
}

// 处理请求
func (s *MCPServer) handleRequest(req MCPRequest) MCPResponse {
	// 查找处理器
	s.mu.RLock()
	handler, ok := s.handlers[req.Method]
	s.mu.RUnlock()

	if !ok {
		return MCPResponse{
			ID:    req.ID,
			Error: fmt.Sprintf("Method not found: %s", req.Method),
		}
	}

	// 执行处理器
	result, err := handler(s.ctx, req.Params)
	if err != nil {
		return MCPResponse{
			ID:    req.ID,
			Error: err.Error(),
		}
	}

	// 序列化结果
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return MCPResponse{
			ID:    req.ID,
			Error: fmt.Sprintf("Failed to marshal result: %v", err),
		}
	}

	return MCPResponse{
		ID:     req.ID,
		Result: resultJSON,
	}
}

// 启动服务器
func (s *MCPServer) Start() error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	s.listener = ln

	slog.Info("MCP Redis Server started", zap.String("addr", s.addr))

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				select {
				case <-s.ctx.Done():
					return
				default:
				}
				slog.Error("Failed to accept connection", zap.Error(err))
				continue
			}

			s.wg.Add(1)
			go s.handleConnection(conn)
		}
	}()

	return nil
}

// 停止服务器
func (s *MCPServer) Stop() error {
	s.cancel()
	if s.listener != nil {
		s.listener.Close()
	}
	s.wg.Wait()
	slog.Info("MCP Redis Server stopped")
	return nil
}

// 主函数
func main() {
	// 初始化服务器
	server := NewMCPServer(":8080", "localhost:6379", 10)

	// 注册模型创建处理器
	server.Register("model.create", func(ctx context.Context, args json.RawMessage) (interface{}, error) {
		var params struct {
			ModelName  string                 `json:"model_name"`
			Parameters map[string]interface{} `json:"parameters,omitempty"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return nil, fmt.Errorf("invalid params format: %v", err)
		}

		// 生成上下文 ID
		ctxID := fmt.Sprintf("model_%d", time.Now().UnixNano())

		// 创建模型上下文
		modelCtx := ModelContext{
			ID:           ctxID,
			ModelName:    params.ModelName,
			Parameters:   params.Parameters,
			ContextData:  make(map[string]interface{}),
			CreatedAt:    time.Now(),
			LastAccessed: time.Now(),
		}

		// 存储到内存
		server.mu.Lock()
		server.modelContext[ctxID] = modelCtx
		server.mu.Unlock()

		// 存储到 Redis
		ctxData, err := json.Marshal(modelCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal model context: %v", err)
		}
		_, err = server.redisClient.Do("SET", fmt.Sprintf("model_ctx:%s", ctxID), ctxData)
		if err != nil {
			return nil, fmt.Errorf("failed to store model context: %v", err)
		}

		return map[string]string{"context_id": ctxID}, nil
	})

	// 注册模型推理处理器
	server.Register("model.infer", func(ctx context.Context, args json.RawMessage) (interface{}, error) {
		var params struct {
			ContextID string      `json:"context_id"`
			Input     interface{} `json:"input"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return nil, fmt.Errorf("invalid params format: %v", err)
		}

		// 获取模型上下文
		server.mu.Lock()
		modelCtx, ok := server.modelContext[params.ContextID]
		if !ok {
			server.mu.Unlock()
			// 尝试从 Redis 加载
			ctxData, err := server.redisClient.Do("GET", fmt.Sprintf("model_ctx:%s", params.ContextID))
			if err != nil || ctxData == nil {
				return nil, fmt.Errorf("model context not found: %s", params.ContextID)
			}

			ctxStr, ok := ctxData.([]byte)
			if !ok {
				return nil, fmt.Errorf("invalid model context data")
			}

			err = json.Unmarshal(ctxStr, &modelCtx)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal model context: %v", err)
			}

			server.modelContext[params.ContextID] = modelCtx
		}

		// 更新最后访问时间
		modelCtx.LastAccessed = time.Now()
		server.modelContext[params.ContextID] = modelCtx
		server.mu.Unlock()

		// 模拟模型推理
		// 实际应用中，这里应该调用真实的模型进行推理
		result := map[string]interface{}{
			"output": fmt.Sprintf("Inference result for model %s with input %v", modelCtx.ModelName, params.Input),
			"model":  modelCtx.ModelName,
			"params": modelCtx.Parameters,
		}

		// 更新上下文数据
		server.mu.Lock()
		modelCtx.ContextData["last_input"] = params.Input
		modelCtx.ContextData["last_output"] = result
		server.modelContext[params.ContextID] = modelCtx
		server.mu.Unlock()

		// 保存到 Redis
		ctxData, err := json.Marshal(modelCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal model context: %v", err)
		}
		_, err = server.redisClient.Do("SET", fmt.Sprintf("model_ctx:%s", params.ContextID), ctxData)
		if err != nil {
			return nil, fmt.Errorf("failed to store model context: %v", err)
		}

		return result, nil
	})

	// 注册模型更新处理器
	server.Register("model.update", func(ctx context.Context, args json.RawMessage) (interface{}, error) {
		var params struct {
			ContextID  string                 `json:"context_id"`
			Parameters map[string]interface{} `json:"parameters"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return nil, fmt.Errorf("invalid params format: %v", err)
		}

		// 获取模型上下文
		server.mu.Lock()
		modelCtx, ok := server.modelContext[params.ContextID]
		if !ok {
			server.mu.Unlock()
			return nil, fmt.Errorf("model context not found: %s", params.ContextID)
		}

		// 更新参数
		modelCtx.Parameters = params.Parameters
		modelCtx.LastAccessed = time.Now()
		server.modelContext[params.ContextID] = modelCtx
		server.mu.Unlock()

		// 保存到 Redis
		ctxData, err := json.Marshal(modelCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal model context: %v", err)
		}
		_, err = server.redisClient.Do("SET", fmt.Sprintf("model_ctx:%s", params.ContextID), ctxData)
		if err != nil {
			return nil, fmt.Errorf("failed to store model context: %v", err)
		}

		return map[string]string{"status": "success"}, nil
	})

	// 注册模型删除处理器
	server.Register("model.delete", func(ctx context.Context, args json.RawMessage) (interface{}, error) {
		var params struct {
			ContextID string `json:"context_id"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return nil, fmt.Errorf("invalid params format: %v", err)
		}

		// 删除内存中的上下文
		server.mu.Lock()
		delete(server.modelContext, params.ContextID)
		server.mu.Unlock()

		// 删除 Redis 中的上下文
		_, err := server.redisClient.Do("DEL", fmt.Sprintf("model_ctx:%s", params.ContextID))
		if err != nil {
			return nil, fmt.Errorf("failed to delete model context: %v", err)
		}

		return map[string]string{"status": "success"}, nil
	})

	// 注册上下文获取处理器
	server.Register("context.get", func(ctx context.Context, args json.RawMessage) (interface{}, error) {
		var params struct {
			ContextID string `json:"context_id"`
			Key       string `json:"key"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return nil, fmt.Errorf("invalid params format: %v", err)
		}

		// 获取模型上下文
		server.mu.Lock()
		modelCtx, ok := server.modelContext[params.ContextID]
		if !ok {
			server.mu.Unlock()
			return nil, fmt.Errorf("model context not found: %s", params.ContextID)
		}
		server.mu.Unlock()

		// 获取上下文数据
		value, ok := modelCtx.ContextData[params.Key]
		if !ok {
			return nil, fmt.Errorf("key not found in context: %s", params.Key)
		}

		return map[string]interface{}{"value": value}, nil
	})

	// 注册上下文设置处理器
	server.Register("context.set", func(ctx context.Context, args json.RawMessage) (interface{}, error) {
		var params struct {
			ContextID string      `json:"context_id"`
			Key       string      `json:"key"`
			Value     interface{} `json:"value"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return nil, fmt.Errorf("invalid params format: %v", err)
		}

		// 获取模型上下文
		server.mu.Lock()
		modelCtx, ok := server.modelContext[params.ContextID]
		if !ok {
			server.mu.Unlock()
			return nil, fmt.Errorf("model context not found: %s", params.ContextID)
		}

		// 设置上下文数据
		modelCtx.ContextData[params.Key] = params.Value
		modelCtx.LastAccessed = time.Now()
		server.modelContext[params.ContextID] = modelCtx
		server.mu.Unlock()

		// 保存到 Redis
		ctxData, err := json.Marshal(modelCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal model context: %v", err)
		}
		_, err = server.redisClient.Do("SET", fmt.Sprintf("model_ctx:%s", params.ContextID), ctxData)
		if err != nil {
			return nil, fmt.Errorf("failed to store model context: %v", err)
		}

		return map[string]string{"status": "success"}, nil
	})

	// 注册 Redis 命令处理器（保持向后兼容）
	server.Register("redis.command", func(ctx context.Context, args json.RawMessage) (interface{}, error) {
		var cmd RedisCommand
		if err := json.Unmarshal(args, &cmd); err != nil {
			return nil, fmt.Errorf("invalid command format: %v", err)
		}

		// 执行 Redis 命令
		result, err := server.redisClient.Do(cmd.Command, cmd.Args...)
		if err != nil {
			return nil, fmt.Errorf("redis command failed: %v", err)
		}

		return result, nil
	})

	// 注册 Redis GET 命令处理器（保持向后兼容）
	server.Register("redis.get", func(ctx context.Context, args json.RawMessage) (interface{}, error) {
		var params struct {
			Key string `json:"key"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return nil, fmt.Errorf("invalid params format: %v", err)
		}

		result, err := server.redisClient.Do("GET", params.Key)
		if err != nil {
			return nil, fmt.Errorf("redis GET failed: %v", err)
		}

		return result, nil
	})

	// 注册 Redis SET 命令处理器（保持向后兼容）
	server.Register("redis.set", func(ctx context.Context, args json.RawMessage) (interface{}, error) {
		var params struct {
			Key    string      `json:"key"`
			Value  interface{} `json:"value"`
			Expire *int        `json:"expire,omitempty"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return nil, fmt.Errorf("invalid params format: %v", err)
		}

		if params.Expire != nil {
			result, err := server.redisClient.Do("SET", params.Key, params.Value, "EX", *params.Expire)
			if err != nil {
				return nil, fmt.Errorf("redis SET with expire failed: %v", err)
			}
			return result, nil
		} else {
			result, err := server.redisClient.Do("SET", params.Key, params.Value)
			if err != nil {
				return nil, fmt.Errorf("redis SET failed: %v", err)
			}
			return result, nil
		}
	})

	// 启动服务器
	if err := server.Start(); err != nil {
		slog.Fatal("Failed to start server", zap.Error(err))
	}

	// 等待中断信号
	select {
	case <-server.ctx.Done():
	}

	// 停止服务器
	if err := server.Stop(); err != nil {
		slog.Error("Failed to stop server", zap.Error(err))
	}
}
