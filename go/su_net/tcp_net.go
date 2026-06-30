package su_net

import (
	"errors"
	"fmt"
	"go.local/my_util"
	slog "go.local/su_log"
	"io"
	"net"
	"sync"
	"time"

	"go.uber.org/zap"
)

type TcpHandler func(*TcpConn, *DataProtocol)

type TcpConn struct {
	conn        *net.TCPConn
	recvData    []byte
	writeMu     sync.Mutex
	closeOnce   sync.Once
	checkTimes  uint8
	PingPongMap sync.Map
}

func newTcpConn(conn *net.TCPConn) *TcpConn {
	return &TcpConn{
		conn:     conn,
		recvData: make([]byte, 0, 4096),
	}
}

func (tc *TcpConn) RemoteAddr() net.Addr {
	if tc == nil || tc.conn == nil {
		return nil
	}
	return tc.conn.RemoteAddr()
}

func (tc *TcpConn) LocalAddr() net.Addr {
	if tc == nil || tc.conn == nil {
		return nil
	}
	return tc.conn.LocalAddr()
}

func (tc *TcpConn) Send(dp *DataProtocol) error {
	bs, err := Encode(dp)
	if err != nil {
		return err
	}
	return tc.SendBytes(bs)
}

func (tc *TcpConn) SendBytes(bs []byte) error {
	if tc == nil || tc.conn == nil {
		return errors.New("tcp conn is nil")
	}
	tc.writeMu.Lock()
	defer tc.writeMu.Unlock()
	for len(bs) > 0 {
		n, err := tc.conn.Write(bs)
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
		bs = bs[n:]
	}
	return nil
}

func (tc *TcpConn) Close() error {
	if tc == nil || tc.conn == nil {
		return nil
	}
	var err error
	tc.closeOnce.Do(func() {
		err = tc.conn.Close()
	})
	return err
}

func (tc *TcpConn) Ping() error {
	routeID := nextRouteID()
	microTime := uint64(time.Now().UnixNano() / 1000)
	data, err := PingEncode(Ping{SendTime: microTime})
	if err != nil {
		return err
	}
	tc.PingPongMap.Store(microTime, 1)
	err = tc.Send(&DataProtocol{
		Head: Header{
			PackId:   PING,
			RouteId:  routeID,
			HeadUuid: microTime,
		},
		Data: data,
	})
	if err != nil {
		tc.PingPongMap.Delete(microTime)
		return err
	}
	return nil
}

func (tc *TcpConn) CheckPong() {
	if tc == nil {
		return
	}
	count := 0
	tc.PingPongMap.Range(func(k, v interface{}) bool {
		count++
		return true
	})
	if count == 0 {
		tc.checkTimes = 0
	} else {
		tc.checkTimes++
	}
	if err := tc.Ping(); err != nil {
		slog.Error("tcp ping failed", zap.Error(err))
		tc.Close()
		return
	}
	if count > 0 && tc.checkTimes >= 2 {
		tc.Close()
	}
}

func (tc *TcpConn) readLoop(handler TcpHandler) {
	buf := make([]byte, 4096)
	defer tc.Close()
	for {
		n, err := tc.conn.Read(buf)
		if err != nil {
			if !errors.Is(err, net.ErrClosed) {
				slog.Info("tcp read end", zap.Error(err))
			}
			return
		}
		if n == 0 {
			continue
		}
		if err := tc.recv(buf[:n], handler); err != nil {
			slog.Error("tcp recv failed", zap.Error(err))
			return
		}
	}
}

func (tc *TcpConn) recv(frame []byte, handler TcpHandler) error {
	tc.recvData = append(tc.recvData, frame...)
	for {
		if len(tc.recvData) < int(HeadLength) {
			return nil
		}
		remain, dp, err := Decode(tc.recvData)
		if err != nil {
			if errors.Is(err, ErrIncompletePacket) {
				return nil
			}
			tc.recvData = tc.recvData[:0]
			return err
		}
		tc.recvData = remain
		handled, err := tc.handleControlPacket(&dp)
		if err != nil {
			return err
		}
		if handled {
			if len(tc.recvData) == 0 {
				return nil
			}
			continue
		}
		if handler != nil {
			handler(tc, &dp)
		}
		if len(tc.recvData) == 0 {
			return nil
		}
	}
}

func (tc *TcpConn) handleControlPacket(dp *DataProtocol) (bool, error) {
	switch dp.Head.PackId {
	case PING:
		ping, err := PingDecode(dp.Data, dp.Head.PackLen)
		if err != nil {
			return true, err
		}
		microTime := uint64(time.Now().UnixNano() / 1000)
		data, err := PongEncode(Pong{SendTime: microTime, PingTime: ping.SendTime})
		if err != nil {
			return true, err
		}
		return true, tc.Send(&DataProtocol{
			Head: Header{
				PackId:   PONG,
				RouteId:  dp.Head.RouteId,
				HeadUuid: microTime,
			},
			Data: data,
		})
	case PONG:
		pong, err := PongDecode(dp.Data, dp.Head.PackLen)
		if err != nil {
			return true, err
		}
		tc.PingPongMap.Delete(pong.PingTime)
		return true, nil
	default:
		return false, nil
	}
}

type TcpServer struct {
	Addr      string
	listener  *net.TCPListener
	handler   TcpHandler
	conns     sync.Map
	closeOnce sync.Once
	pool      *my_util.GoPool
}

func CreateTcpServer(addr string, handlers ...TcpHandler) (*TcpServer, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, err
	}
	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return nil, err
	}
	server := &TcpServer{
		Addr:     listener.Addr().String(),
		listener: listener,
		pool:     my_util.NewGoPool(16, 1024),
	}
	if len(handlers) > 0 {
		server.handler = handlers[0]
	}
	go server.acceptLoop()
	return server, nil
}

func (ts *TcpServer) acceptLoop() {
	for {
		conn, err := ts.listener.AcceptTCP()
		if err != nil {
			if !errors.Is(err, net.ErrClosed) {
				slog.Error("tcp accept failed", zap.Error(err))
			}
			return
		}
		tcpConn := newTcpConn(conn)
		ts.conns.Store(conn.RemoteAddr().String(), tcpConn)
		go func() {
			defer ts.conns.Delete(conn.RemoteAddr().String())
			tcpConn.readLoop(func(conn *TcpConn, dp *DataProtocol) {
				if ts.handler == nil {
					return
				}
				taskDP := *dp
				ts.pool.SendTask(taskDP.Head.RouteId, func() {
					ts.handler(conn, &taskDP)
				})
			})
		}()
	}
}

func (ts *TcpServer) Close() error {
	if ts == nil || ts.listener == nil {
		return nil
	}
	var err error
	ts.closeOnce.Do(func() {
		err = ts.listener.Close()
		ts.conns.Range(func(k, v interface{}) bool {
			if conn, ok := v.(*TcpConn); ok {
				conn.Close()
			}
			return true
		})
		if ts.pool != nil {
			ts.pool.Stop()
		}
	})
	return err
}

func (ts *TcpServer) ConnCount() int {
	if ts == nil {
		return 0
	}
	count := 0
	ts.conns.Range(func(k, v interface{}) bool {
		count++
		return true
	})
	return count
}

type TcpClient struct {
	Addr     string
	Conn     *TcpConn
	handler  TcpHandler
	done     chan struct{}
	stopOnce sync.Once
}

func CreateTcpClient(addr string, handlers ...TcpHandler) (*TcpClient, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return nil, err
	}
	client := &TcpClient{
		Addr: addr,
		Conn: newTcpConn(conn),
		done: make(chan struct{}),
	}
	if len(handlers) > 0 {
		client.handler = handlers[0]
	}
	go func() {
		defer client.stopHeartbeat()
		client.Conn.readLoop(client.handler)
	}()
	go client.heartbeatLoop()
	return client, nil
}

func (tc *TcpClient) stopHeartbeat() {
	tc.stopOnce.Do(func() {
		close(tc.done)
	})
}

func (tc *TcpClient) heartbeatLoop() {
	ticker := time.NewTicker(time.Duration(PING_PONG_INTERVAL) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			tc.Conn.CheckPong()
		case <-tc.done:
			return
		}
	}
}

func (tc *TcpClient) Send(dp *DataProtocol) error {
	if tc == nil || tc.Conn == nil {
		return fmt.Errorf("tcp client is nil")
	}
	return tc.Conn.Send(dp)
}

func (tc *TcpClient) Close() error {
	if tc == nil || tc.Conn == nil {
		return nil
	}
	tc.stopHeartbeat()
	return tc.Conn.Close()
}
