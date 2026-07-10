package su_net

import (
	"errors"
	"go.local/su_errors"
	slog "go.local/su_log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type WSHandler func(*WSConn, *DataProtocol)

type WSConn struct {
	conn         *websocket.Conn
	closed       int32
	recvData     []byte
	writeMu      sync.Mutex
	closeOnce    sync.Once
	checkTimes   int32
	pendingPings int32
	PingPongMap  sync.Map
	writeTimeout int64
}

func newWSConn(conn *websocket.Conn) *WSConn {
	return newWSConnWithWriteTimeout(conn, DEFAULT_WRITE_TIMEOUT)
}

func newWSConnWithWriteTimeout(conn *websocket.Conn, writeTimeout time.Duration) *WSConn {
	return &WSConn{
		conn:         conn,
		recvData:     make([]byte, 0, 4096),
		writeTimeout: int64(writeTimeout),
	}
}

func (wc *WSConn) SetWriteTimeout(timeout time.Duration) {
	if wc == nil {
		return
	}
	atomic.StoreInt64(&wc.writeTimeout, int64(timeout))
}

func (wc *WSConn) WriteTimeout() time.Duration {
	if wc == nil {
		return 0
	}
	return time.Duration(atomic.LoadInt64(&wc.writeTimeout))
}

func (wc *WSConn) Send(dp *DataProtocol) error {
	bs, err := Encode(dp)
	if err != nil {
		return err
	}
	return wc.SendBytes(bs)
}

func (wc *WSConn) SendBytes(bs []byte) error {
	if wc == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "websocket conn is nil")
	}
	wc.writeMu.Lock()
	defer wc.writeMu.Unlock()
	if atomic.LoadInt32(&wc.closed) == 1 || wc.conn == nil {
		return su_errors.New(su_errors.CodeUnavailable, "websocket conn is closed")
	}
	if writeTimeout := wc.WriteTimeout(); writeTimeout > 0 {
		if err := wc.conn.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
			return su_errors.WrapRetryable(su_errors.CodeUnavailable, "websocket set write deadline failed", err)
		}
		defer wc.conn.SetWriteDeadline(time.Time{})
	}
	if err := wc.conn.WriteMessage(websocket.BinaryMessage, bs); err != nil {
		return su_errors.WrapRetryable(su_errors.CodeUnavailable, "websocket write failed", err)
	}
	return nil
}

func (wc *WSConn) Close() error {
	if wc == nil {
		return nil
	}
	var err error
	wc.closeOnce.Do(func() {
		atomic.StoreInt32(&wc.closed, 1)
		wc.ClearHeartbeat()
		if wc.conn == nil {
			return
		}
		err = wc.conn.Close()
	})
	return err
}

func (wc *WSConn) ClearHeartbeat() {
	if wc == nil {
		return
	}
	deleteAllSyncMap(&wc.PingPongMap)
	atomic.StoreInt32(&wc.pendingPings, 0)
	atomic.StoreInt32(&wc.checkTimes, 0)
}

func (wc *WSConn) Ping() error {
	routeID := nextRouteID()
	microTime := uint64(time.Now().UnixNano() / 1000)
	data, err := PingEncode(Ping{SendTime: microTime})
	if err != nil {
		return err
	}
	wc.PingPongMap.Store(microTime, 1)
	atomic.AddInt32(&wc.pendingPings, 1)
	err = wc.Send(&DataProtocol{
		Head: Header{
			PackId:   PING,
			RouteId:  routeID,
			HeadUuid: microTime,
		},
		Data: data,
	})
	if err != nil {
		if _, ok := wc.PingPongMap.LoadAndDelete(microTime); ok {
			atomic.AddInt32(&wc.pendingPings, -1)
		}
		return err
	}
	return nil
}

func (wc *WSConn) CheckPong() {
	if wc == nil || atomic.LoadInt32(&wc.closed) == 1 {
		return
	}
	count := atomic.LoadInt32(&wc.pendingPings)
	var checkTimes int32
	if count == 0 {
		atomic.StoreInt32(&wc.checkTimes, 0)
	} else {
		checkTimes = atomic.AddInt32(&wc.checkTimes, 1)
	}
	if err := wc.Ping(); err != nil {
		slog.Error("websocket ping failed", zap.Error(err))
		wc.Close()
		return
	}
	if count > 0 && checkTimes >= 2 {
		wc.Close()
	}
}

func (wc *WSConn) readLoop(handler WSHandler) {
	defer wc.Close()
	for {
		messageType, data, err := wc.conn.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				slog.Info("websocket read end", zap.Error(err))
			}
			return
		}
		if messageType != websocket.BinaryMessage {
			continue
		}
		if err := wc.recv(data, handler); err != nil {
			slog.Error("websocket recv failed", zap.Error(err))
			return
		}
	}
}

func (wc *WSConn) recv(frame []byte, handler WSHandler) error {
	wc.recvData = append(wc.recvData, frame...)
	for {
		if len(wc.recvData) < int(HeadLength) {
			return nil
		}
		remain, dp, err := DecodeNoCopy(wc.recvData)
		if err != nil {
			if errors.Is(err, ErrIncompletePacket) {
				return nil
			}
			wc.recvData = wc.recvData[:0]
			return err
		}
		if len(remain) == 0 {
			// DecodeNoCopy returns Data slices backed by recvData. Drop the
			// backing array after a full drain so the next frame cannot
			// overwrite data still being handled asynchronously.
			wc.recvData = nil
		} else {
			wc.recvData = remain
		}
		handled, err := wc.handleControlPacket(&dp)
		if err != nil {
			if atomic.LoadInt32(&wc.closed) == 1 {
				return nil
			}
			return err
		}
		if handled {
			if len(wc.recvData) == 0 {
				return nil
			}
			continue
		}
		if handler != nil {
			handler(wc, &dp)
		}
		if len(wc.recvData) == 0 {
			return nil
		}
	}
}

func (wc *WSConn) handleControlPacket(dp *DataProtocol) (bool, error) {
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
		return true, wc.Send(&DataProtocol{
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
		if _, ok := wc.PingPongMap.LoadAndDelete(pong.PingTime); ok {
			atomic.AddInt32(&wc.pendingPings, -1)
		}
		return true, nil
	default:
		return false, nil
	}
}
