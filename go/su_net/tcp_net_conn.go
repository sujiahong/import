package su_net

import (
	"errors"
	"go.local/su_errors"
	slog "go.local/su_log"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// TcpHandler 处理 TCP 连接上解析出的业务数据包。
type TcpHandler func(*TcpConn, *DataProtocol)

// TcpConn 封装 net.TCPConn，并维护收包缓存、写锁和心跳状态。
type TcpConn struct {
	conn         *net.TCPConn
	closed       int32
	recvData     []byte
	writeMu      sync.Mutex
	closeOnce    sync.Once
	checkTimes   int32
	pendingPings int32
	PingPongMap  sync.Map
	writeTimeout int64
}

// newTcpConn 使用默认写超时创建 TCP 连接包装。
func newTcpConn(conn *net.TCPConn) *TcpConn {
	return newTcpConnWithWriteTimeout(conn, DEFAULT_WRITE_TIMEOUT)
}

// newTcpConnWithWriteTimeout 使用指定写超时创建 TCP 连接包装。
func newTcpConnWithWriteTimeout(conn *net.TCPConn, writeTimeout time.Duration) *TcpConn {
	return &TcpConn{
		conn:         conn,
		recvData:     make([]byte, 0, 4096),
		writeTimeout: int64(writeTimeout),
	}
}

// SetWriteTimeout 更新当前连接的单次写超时。
func (tc *TcpConn) SetWriteTimeout(timeout time.Duration) {
	if tc == nil {
		return
	}
	atomic.StoreInt64(&tc.writeTimeout, int64(timeout))
}

// WriteTimeout 返回当前连接的单次写超时。
func (tc *TcpConn) WriteTimeout() time.Duration {
	if tc == nil {
		return 0
	}
	return time.Duration(atomic.LoadInt64(&tc.writeTimeout))
}

// RemoteAddr 返回底层 TCP 连接的远端地址。
func (tc *TcpConn) RemoteAddr() net.Addr {
	if tc == nil || tc.conn == nil {
		return nil
	}
	return tc.conn.RemoteAddr()
}

// LocalAddr 返回底层 TCP 连接的本地地址。
func (tc *TcpConn) LocalAddr() net.Addr {
	if tc == nil || tc.conn == nil {
		return nil
	}
	return tc.conn.LocalAddr()
}

// Send 编码 DataProtocol 并写入 TCP 连接。
func (tc *TcpConn) Send(dp *DataProtocol) error {
	bs, err := Encode(dp)
	if err != nil {
		return err
	}
	return tc.SendBytes(bs)
}

// SendBytes 将完整二进制包写入 TCP 连接，并处理短写。
func (tc *TcpConn) SendBytes(bs []byte) error {
	if tc == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "tcp conn is nil")
	}
	tc.writeMu.Lock()
	defer tc.writeMu.Unlock()
	if atomic.LoadInt32(&tc.closed) == 1 || tc.conn == nil {
		return su_errors.New(su_errors.CodeUnavailable, "tcp conn is closed")
	}
	if writeTimeout := tc.WriteTimeout(); writeTimeout > 0 {
		if err := tc.conn.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
			return su_errors.WrapRetryable(su_errors.CodeUnavailable, "tcp set write deadline failed", err)
		}
		defer tc.conn.SetWriteDeadline(time.Time{})
	}
	for len(bs) > 0 {
		n, err := tc.conn.Write(bs)
		if err != nil {
			return su_errors.WrapRetryable(su_errors.CodeUnavailable, "tcp write failed", err)
		}
		if n == 0 {
			return su_errors.WrapRetryable(su_errors.CodeUnavailable, "tcp short write", io.ErrShortWrite)
		}
		bs = bs[n:]
	}
	return nil
}

// Close 关闭 TCP 连接并清理心跳状态。
func (tc *TcpConn) Close() error {
	if tc == nil {
		return nil
	}
	var err error
	tc.closeOnce.Do(func() {
		atomic.StoreInt32(&tc.closed, 1)
		tc.ClearHeartbeat()
		if tc.conn == nil {
			return
		}
		err = tc.conn.Close()
	})
	return err
}

// ClearHeartbeat 清空未收到响应的心跳记录。
func (tc *TcpConn) ClearHeartbeat() {
	if tc == nil {
		return
	}
	deleteAllSyncMap(&tc.PingPongMap)
	atomic.StoreInt32(&tc.pendingPings, 0)
	atomic.StoreInt32(&tc.checkTimes, 0)
}

// Ping 发送一次应用层心跳请求。
func (tc *TcpConn) Ping() error {
	routeID := nextRouteID()
	microTime := uint64(time.Now().UnixNano() / 1000)
	data, err := PingEncode(Ping{SendTime: microTime})
	if err != nil {
		return err
	}
	tc.PingPongMap.Store(microTime, 1)
	atomic.AddInt32(&tc.pendingPings, 1)
	err = tc.Send(&DataProtocol{
		Head: Header{
			PackId:   PING,
			RouteId:  routeID,
			HeadUuid: microTime,
		},
		Data: data,
	})
	if err != nil {
		if _, ok := tc.PingPongMap.LoadAndDelete(microTime); ok {
			atomic.AddInt32(&tc.pendingPings, -1)
		}
		return err
	}
	return nil
}

// CheckPong 检查未完成心跳并在连续未响应时关闭连接。
func (tc *TcpConn) CheckPong() {
	if tc == nil || atomic.LoadInt32(&tc.closed) == 1 {
		return
	}
	count := atomic.LoadInt32(&tc.pendingPings)
	var checkTimes int32
	if count == 0 {
		atomic.StoreInt32(&tc.checkTimes, 0)
	} else {
		checkTimes = atomic.AddInt32(&tc.checkTimes, 1)
	}
	if err := tc.Ping(); err != nil {
		slog.Error("tcp ping failed", zap.Error(err))
		tc.Close()
		return
	}
	if count > 0 && checkTimes >= 2 {
		tc.Close()
	}
}

// readLoop 持续读取底层 TCP 流并交给 recv 解析。
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
			slog.Warn("tcp read returned zero bytes without error")
			return
		}
		if err := tc.recv(buf[:n], handler); err != nil {
			slog.Error("tcp recv failed", zap.Error(err))
			return
		}
	}
}

// recv 处理 TCP 粘包/半包，并将完整业务包交给 handler。
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
			if atomic.LoadInt32(&tc.closed) == 1 {
				return nil
			}
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

// handleControlPacket 处理 PING/PONG 控制包，返回 true 表示该包已被内部消费。
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
		if _, ok := tc.PingPongMap.LoadAndDelete(pong.PingTime); ok {
			atomic.AddInt32(&tc.pendingPings, -1)
		}
		return true, nil
	default:
		return false, nil
	}
}
