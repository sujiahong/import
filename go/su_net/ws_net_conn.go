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

// wsHandler 处理 WebSocket 连接上解析出的业务数据包。
type wsHandler func(*WSConn, *DataProtocol)

// WSConn 封装 gorilla/websocket.Conn，并维护收包缓存、写锁和心跳状态。
type WSConn struct {
	conn         *websocket.Conn // 底层 WebSocket 连接。
	closed       int32           // 连接是否已关闭，按 atomic 访问。
	recvData     []byte          // WebSocket 帧内半包/粘包缓存。
	writeMu      sync.Mutex      // 串行化写操作和写 deadline 设置。
	closeOnce    sync.Once       // 保证 Close 只执行一次。
	checkTimes   int32           // 连续检测到未完成心跳的次数。
	pendingPings int32           // 尚未收到 PONG 的心跳数量。
	PingPongMap  sync.Map        // Ping 发送时间到占位值的映射。
	writeTimeout int64           // 写超时，存储为 time.Duration 的 int64。
}

// newWSConn 使用默认写超时创建 WebSocket 连接包装。
func newWSConn(conn *websocket.Conn) *WSConn {
	return newWSConnWithWriteTimeout(conn, DEFAULT_WRITE_TIMEOUT)
}

// newWSConnWithWriteTimeout 使用指定写超时创建 WebSocket 连接包装。
func newWSConnWithWriteTimeout(conn *websocket.Conn, writeTimeout time.Duration) *WSConn {
	return &WSConn{
		conn:         conn,
		recvData:     make([]byte, 0, 4096),
		writeTimeout: int64(writeTimeout),
	}
}

// SetWriteTimeout 更新当前连接的单次写超时。
func (wc *WSConn) SetWriteTimeout(timeout time.Duration) {
	if wc == nil {
		return
	}
	atomic.StoreInt64(&wc.writeTimeout, int64(timeout))
}

// WriteTimeout 返回当前连接的单次写超时。
func (wc *WSConn) WriteTimeout() time.Duration {
	if wc == nil {
		return 0
	}
	return time.Duration(atomic.LoadInt64(&wc.writeTimeout))
}

// Send 编码 DataProtocol 并作为二进制 WebSocket 消息发送。
func (wc *WSConn) Send(dp *DataProtocol) error {
	bs, err := Encode(dp)
	if err != nil {
		return err
	}
	return wc.SendBytes(bs)
}

// SendBytes 发送已编码的二进制 WebSocket 消息。
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

// Close 关闭 WebSocket 连接并清理心跳状态。
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

// ClearHeartbeat 清空未收到响应的心跳记录。
func (wc *WSConn) ClearHeartbeat() {
	if wc == nil {
		return
	}
	deleteAllSyncMap(&wc.PingPongMap)
	atomic.StoreInt32(&wc.pendingPings, 0)
	atomic.StoreInt32(&wc.checkTimes, 0)
}

// Ping 发送一次应用层心跳请求。
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

// CheckPong 检查未完成心跳并在连续未响应时关闭连接。
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

// readLoop 持续读取 WebSocket 二进制消息并交给 recv 解析。
func (wc *WSConn) readLoop(handler wsHandler) {
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

// recv 处理 WebSocket 消息中的粘包/半包，并将完整业务包交给 handler。
func (wc *WSConn) recv(frame []byte, handler wsHandler) error {
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

// handleControlPacket 处理 PING/PONG 控制包，返回 true 表示该包已被内部消费。
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
