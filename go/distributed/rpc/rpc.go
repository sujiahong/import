package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

type Request struct {
	ID      string          `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	Timeout time.Duration   `json:"timeout"`
}

type Response struct {
	ID     string          `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  string          `json:"error,omitempty"`
}

type Codec interface {
	Encode(v interface{}) ([]byte, error)
	Decode(data []byte, v interface{}) error
}

type JSONCodec struct{}

func (c *JSONCodec) Encode(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (c *JSONCodec) Decode(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

type Client interface {
	Call(ctx context.Context, method string, args, reply interface{}) error
	Close() error
}

type Server interface {
	Register(name string, handler interface{})
	Start(addr string) error
	Stop() error
}

type HandlerFunc func(ctx context.Context, args interface{}) (interface{}, error)

type RPCServer struct {
	addr     string
	listener net.Listener
	codec    Codec
	handlers map[string]HandlerFunc
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

type Args struct {
	A int `json:"a"`
	B int `json:"b"`
}

func NewRPCServer(addr string) *RPCServer {
	ctx, cancel := context.WithCancel(context.Background())
	return &RPCServer{
		addr:     addr,
		codec:    &JSONCodec{},
		handlers: make(map[string]HandlerFunc),
		ctx:      ctx,
		cancel:   cancel,
	}
}

func (s *RPCServer) Register(name string, handler interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch h := handler.(type) {
	case HandlerFunc:
		s.handlers[name] = h
	case func(ctx context.Context, args interface{}) (interface{}, error):
		s.handlers[name] = h
	default:
		panic("invalid handler type")
	}
}

func (s *RPCServer) handleConnection(conn net.Conn) {
	defer conn.Close()
	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		var req Request
		if err := decoder.Decode(&req); err != nil {
			if err == io.EOF {
				return
			}
			continue
		}

		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.handleRequest(conn, req, encoder)
		}()
	}
}

func (s *RPCServer) handleRequest(conn net.Conn, req Request, encoder *json.Encoder) {
	s.mu.RLock()
	handler, ok := s.handlers[req.Method]
	s.mu.RUnlock()

	resp := Response{ID: req.ID}

	if !ok {
		resp.Error = "method not found"
		encoder.Encode(resp)
		return
	}

	ctx, cancel := context.WithTimeout(s.ctx, req.Timeout)
	defer cancel()

	var args Args
	if err := json.Unmarshal(req.Params, &args); err != nil {
		resp.Error = err.Error()
		encoder.Encode(resp)
		return
	}

	result, err := handler(ctx, args)
	if err != nil {
		resp.Error = err.Error()
	} else if result != nil {
		resp.Result, _ = json.Marshal(result)
	}

	encoder.Encode(resp)
}

func (s *RPCServer) Start(addr string) error {
	if addr != "" {
		s.addr = addr
	}

	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	s.listener = ln

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				select {
				case <-s.ctx.Done():
					return
				default:
				}
				continue
			}
			go s.handleConnection(conn)
		}
	}()

	return nil
}

func (s *RPCServer) Stop() error {
	s.cancel()
	if s.listener != nil {
		s.listener.Close()
	}
	s.wg.Wait()
	return nil
}

func (s *RPCServer) Addr() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return s.addr
}

type RPCClient struct {
	conn    net.Conn
	codec   Codec
	mu      sync.Mutex
	pending map[string]chan *Response
	timeout time.Duration
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

func NewRPCClient(addr string, timeout time.Duration) (*RPCClient, error) {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	client := &RPCClient{
		conn:    conn,
		codec:   &JSONCodec{},
		pending: make(map[string]chan *Response),
		timeout: timeout,
		ctx:     ctx,
		cancel:  cancel,
	}

	client.wg.Add(1)
	go client.receiver()

	return client, nil
}

func (c *RPCClient) receiver() {
	defer c.wg.Done()
	decoder := json.NewDecoder(c.conn)

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		c.conn.SetReadDeadline(time.Now().Add(c.timeout))
		var resp Response
		if err := decoder.Decode(&resp); err != nil {
			if err == io.EOF || c.ctx.Err() != nil {
				return
			}
			continue
		}

		c.mu.Lock()
		if ch, ok := c.pending[resp.ID]; ok {
			select {
			case ch <- &resp:
			default:
			}
			delete(c.pending, resp.ID)
		}
		c.mu.Unlock()
	}
}

func (c *RPCClient) Call(ctx context.Context, method string, args, reply interface{}) error {
	id := generateID()

	params, err := c.codec.Encode(args)
	if err != nil {
		return err
	}

	req := Request{
		ID:      id,
		Method:  method,
		Params:  params,
		Timeout: c.timeout,
	}

	respCh := make(chan *Response, 1)
	c.mu.Lock()
	c.pending[id] = respCh
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
	}()

	data, err := c.codec.Encode(req)
	if err != nil {
		return err
	}

	_, err = c.conn.Write(data)
	if err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case resp := <-respCh:
		if resp.Error != "" {
			return fmt.Errorf(resp.Error)
		}
		if reply != nil && len(resp.Result) > 0 {
			return c.codec.Decode(resp.Result, reply)
		}
		return nil
	}
}

func (c *RPCClient) Close() error {
	c.cancel()
	c.wg.Wait()
	return c.conn.Close()
}

func generateID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().Nanosecond())
}

type Pool struct {
	clients map[string]*RPCClient
	mu      sync.RWMutex
}

func NewPool() *Pool {
	return &Pool{
		clients: make(map[string]*RPCClient),
	}
}

func (p *Pool) GetClient(addr string) (*RPCClient, error) {
	p.mu.RLock()
	if client, ok := p.clients[addr]; ok {
		p.mu.RUnlock()
		return client, nil
	}
	p.mu.RUnlock()

	client, err := NewRPCClient(addr, 10*time.Second)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	p.clients[addr] = client
	p.mu.Unlock()

	return client, nil
}

func (p *Pool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, client := range p.clients {
		client.Close()
	}
}
