package transport

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type Message struct {
	Header map[string]string
	Body   []byte
}

type Encoder interface {
	Encode(msg *Message) ([]byte, error)
}

type Decoder interface {
	Decode(data []byte) (*Message, error)
}

type LengthPrefixEncoder struct{}

func (e *LengthPrefixEncoder) Encode(msg *Message) ([]byte, error) {
	body := msg.Body
	headerBytes, err := encodeHeader(msg.Header)
	if err != nil {
		return nil, err
	}

	payloadLen := 4 + len(headerBytes) + len(body)
	result := make([]byte, 4+payloadLen)

	binary.BigEndian.PutUint32(result[0:4], uint32(payloadLen))
	binary.BigEndian.PutUint32(result[4:8], uint32(len(headerBytes)))
	copy(result[8:8+len(headerBytes)], headerBytes)
	copy(result[8+len(headerBytes):], body)

	return result, nil
}

func (e *LengthPrefixEncoder) Decode(data []byte) (*Message, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("data too short")
	}

	headerLen := int(binary.BigEndian.Uint32(data[0:4]))
	if len(data) < 4+headerLen {
		return nil, fmt.Errorf("data too short for header")
	}

	header, err := decodeHeader(data[4 : 4+headerLen])
	if err != nil {
		return nil, err
	}

	body := data[4+headerLen:]
	return &Message{Header: header, Body: body}, nil
}

func encodeHeader(header map[string]string) ([]byte, error) {
	var result []byte
	for k, v := range header {
		keyLen := uint16(len(k))
		valLen := uint16(len(v))
		buf := make([]byte, 2+len(k)+2+len(v))
		binary.BigEndian.PutUint16(buf[0:2], keyLen)
		copy(buf[2:2+len(k)], k)
		binary.BigEndian.PutUint16(buf[2+len(k):4+len(k)], valLen)
		copy(buf[4+len(k):], v)
		result = append(result, buf...)
	}
	return result, nil
}

func decodeHeader(data []byte) (map[string]string, error) {
	header := make(map[string]string)
	pos := 0
	for pos < len(data) {
		if pos+2 > len(data) {
			break
		}
		keyLen := int(binary.BigEndian.Uint16(data[pos : pos+2]))
		pos += 2
		if pos+keyLen > len(data) {
			break
		}
		key := string(data[pos : pos+keyLen])
		pos += keyLen
		if pos+2 > len(data) {
			break
		}
		valLen := int(binary.BigEndian.Uint16(data[pos : pos+2]))
		pos += 2
		if pos+valLen > len(data) {
			break
		}
		val := string(data[pos : pos+valLen])
		pos += valLen
		header[key] = val
	}
	return header, nil
}

type Transport struct {
	conn    net.Conn
	encoder Encoder
	decoder Decoder
	sendCh  chan *Message
	recvCh  chan *Message
	closeCh chan struct{}
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

func NewTransport(conn net.Conn, encoder Encoder, decoder Decoder) *Transport {
	ctx, cancel := context.WithCancel(context.Background())
	return &Transport{
		conn:    conn,
		encoder: encoder,
		decoder: decoder,
		sendCh:  make(chan *Message, 100),
		recvCh:  make(chan *Message, 100),
		closeCh: make(chan struct{}),
		ctx:     ctx,
		cancel:  cancel,
	}
}

func (t *Transport) Start() {
	t.wg.Add(2)
	go t.readLoop()
	go t.writeLoop()
}

func (t *Transport) readLoop() {
	defer t.wg.Done()

	headerBuf := make([]byte, 4)
	for {
		select {
		case <-t.ctx.Done():
			return
		default:
		}

		t.conn.SetReadDeadline(time.Now().Add(30 * time.Second))

		if _, err := io.ReadFull(t.conn, headerBuf); err != nil {
			if err != io.EOF {
				// log error
			}
			t.cancel()
			return
		}

		frameLen := binary.BigEndian.Uint32(headerBuf)
		data := make([]byte, frameLen)
		if _, err := io.ReadFull(t.conn, data); err != nil {
			t.cancel()
			return
		}

		msg, err := t.decoder.Decode(data)
		if err != nil {
			continue
		}

		select {
		case t.recvCh <- msg:
		case <-t.ctx.Done():
			return
		case <-t.closeCh:
			return
		}
	}
}

func (t *Transport) writeLoop() {
	defer t.wg.Done()

	for {
		select {
		case msg := <-t.sendCh:
			data, err := t.encoder.Encode(msg)
			if err != nil {
				t.cancel()
				return
			}

			_, err = t.conn.Write(data)
			if err != nil {
				t.cancel()
				return
			}
		case <-t.ctx.Done():
			return
		case <-t.closeCh:
			return
		}
	}
}

func (t *Transport) Send(ctx context.Context, msg *Message) error {
	select {
	case t.sendCh <- msg:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-t.closeCh:
		return fmt.Errorf("transport closed")
	}
}

func (t *Transport) Recv(ctx context.Context) (*Message, error) {
	select {
	case msg := <-t.recvCh:
		return msg, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-t.closeCh:
		return nil, fmt.Errorf("transport closed")
	}
}

func (t *Transport) Close() error {
	close(t.closeCh)
	t.cancel()
	t.wg.Wait()
	return t.conn.Close()
}

func (t *Transport) LocalAddr() net.Addr  { return t.conn.LocalAddr() }
func (t *Transport) RemoteAddr() net.Addr { return t.conn.RemoteAddr() }

type Server struct {
	addr      string
	listener  net.Listener
	transport func(net.Conn) *Transport
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	conns     map[*Transport]struct{}
}

func NewServer(addr string, transport func(net.Conn) *Transport) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		addr:      addr,
		transport: transport,
		ctx:       ctx,
		cancel:    cancel,
		conns:     make(map[*Transport]struct{}),
	}
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	s.listener = ln

	go s.acceptLoop()
	return nil
}

func (s *Server) acceptLoop() {
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		s.listener.(*net.TCPListener).SetDeadline(time.Now().Add(1 * time.Second))
		conn, err := s.listener.Accept()
		if err != nil {
			if s.ctx.Err() != nil {
				return
			}
			continue
		}

		tp := s.transport(conn)
		tp.Start()

		s.mu.Lock()
		s.conns[tp] = struct{}{}
		s.mu.Unlock()
	}
}

func (s *Server) Stop() error {
	s.cancel()

	s.mu.Lock()
	for conn := range s.conns {
		conn.Close()
	}
	s.conns = make(map[*Transport]struct{})
	s.mu.Unlock()

	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

type Client struct {
	addr   string
	conn   net.Conn
	tp     *Transport
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.Mutex
}

func NewClient(addr string) (*Client, error) {
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	tp := NewTransport(conn, &LengthPrefixEncoder{}, &LengthPrefixEncoder{})
	tp.Start()

	return &Client{
		addr:   addr,
		conn:   conn,
		tp:     tp,
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

func (c *Client) Send(ctx context.Context, msg *Message) error {
	return c.tp.Send(ctx, msg)
}

func (c *Client) Recv(ctx context.Context) (*Message, error) {
	return c.tp.Recv(ctx)
}

func (c *Client) Close() error {
	c.cancel()
	return c.tp.Close()
}

type Pool struct {
	clients  []*Client
	mu       sync.RWMutex
	current  uint32
	maxConns int
	addr     string
}

func NewPool(addr string, maxConns int) *Pool {
	return &Pool{
		addr:     addr,
		maxConns: maxConns,
		clients:  make([]*Client, 0, maxConns),
	}
}

func (p *Pool) Get() (*Client, error) {
	p.mu.RLock()
	if len(p.clients) > 0 {
		idx := atomic.AddUint32(&p.current, 1) % uint32(len(p.clients))
		client := p.clients[idx]
		p.mu.RUnlock()
		return client, nil
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.clients) >= p.maxConns {
		idx := atomic.AddUint32(&p.current, 1) % uint32(len(p.clients))
		return p.clients[idx], nil
	}

	client, err := NewClient(p.addr)
	if err != nil {
		return nil, err
	}
	p.clients = append(p.clients, client)
	return client, nil
}

func (p *Pool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, c := range p.clients {
		c.Close()
	}
}
