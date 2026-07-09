package su_net

import (
	"context"
	"errors"
	"fmt"
	"go.local/my_util"
	slog "go.local/su_log"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

const defaultWSPath = "/ws"

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
}

func newWSConn(conn *websocket.Conn) *WSConn {
	return &WSConn{
		conn:     conn,
		recvData: make([]byte, 0, 4096),
	}
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
		return errors.New("websocket conn is nil")
	}
	wc.writeMu.Lock()
	defer wc.writeMu.Unlock()
	if atomic.LoadInt32(&wc.closed) == 1 || wc.conn == nil {
		return errors.New("websocket conn is closed")
	}
	return wc.conn.WriteMessage(websocket.BinaryMessage, bs)
}

func (wc *WSConn) Close() error {
	if wc == nil {
		return nil
	}
	var err error
	wc.closeOnce.Do(func() {
		atomic.StoreInt32(&wc.closed, 1)
		wc.ClearHeartbeat()
		wc.writeMu.Lock()
		defer wc.writeMu.Unlock()
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

type WSServer struct {
	Addr       string
	Path       string
	server     *http.Server
	listener   net.Listener
	handler    WSHandler
	conns      map[uint64]*WSConn
	connsMu    sync.Mutex
	nextConnID uint64
	pool       *my_util.GoPool
	closeOnce  sync.Once
	upgrader   websocket.Upgrader
}

func CreateWSServer(addr string, handlers ...WSHandler) (*WSServer, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	ws := &WSServer{
		Addr:     listener.Addr().String(),
		Path:     defaultWSPath,
		listener: listener,
		conns:    make(map[uint64]*WSConn),
		pool:     my_util.NewGoPool(16, 1024),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
	if len(handlers) > 0 {
		ws.handler = handlers[0]
	}
	mux := http.NewServeMux()
	mux.HandleFunc(ws.Path, ws.handleHTTP)
	ws.server = &http.Server{Handler: mux}
	go func() {
		if err := ws.server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("websocket server failed", zap.Error(err))
		}
	}()
	return ws, nil
}

func (ws *WSServer) handleHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := ws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("websocket upgrade failed", zap.Error(err))
		return
	}
	wsConn := newWSConn(conn)
	key := atomic.AddUint64(&ws.nextConnID, 1)
	ws.connsMu.Lock()
	ws.conns[key] = wsConn
	ws.connsMu.Unlock()
	go func() {
		defer func() {
			ws.connsMu.Lock()
			if ws.conns[key] == wsConn {
				delete(ws.conns, key)
			}
			ws.connsMu.Unlock()
		}()
		wsConn.readLoop(func(conn *WSConn, dp *DataProtocol) {
			if ws.handler == nil {
				return
			}
			taskDP := *dp
			taskDP.Data = append([]byte(nil), dp.Data...)
			if !ws.pool.SendTask(taskDP.Head.RouteId, func() {
				ws.handler(conn, &taskDP)
			}) {
				slog.Warn("websocket server task dropped", zap.Uint64("route_id", taskDP.Head.RouteId))
			}
		})
	}()
}

func (ws *WSServer) Close() error {
	if ws == nil {
		return nil
	}
	var err error
	ws.closeOnce.Do(func() {
		if ws.listener != nil {
			ws.listener.Close()
		}
		if ws.server != nil {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			err = ws.server.Shutdown(ctx)
			cancel()
		}
		conns := make([]*WSConn, 0)
		ws.connsMu.Lock()
		for _, conn := range ws.conns {
			conns = append(conns, conn)
		}
		ws.connsMu.Unlock()
		for _, conn := range conns {
			conn.Close()
		}
		if ws.pool != nil && !ws.pool.StopAndDrain(DEFAULT_CLOSE_TIMEOUT) {
			slog.Warn("websocket server pool drain timeout")
		}
	})
	return err
}

func (ws *WSServer) ConnCount() int {
	if ws == nil {
		return 0
	}
	ws.connsMu.Lock()
	defer ws.connsMu.Unlock()
	return len(ws.conns)
}

type WSClient struct {
	Addr     string
	Conn     *WSConn
	handler  WSHandler
	done     chan struct{}
	stopOnce sync.Once
}

func CreateWSClient(addr string, handlers ...WSHandler) (*WSClient, error) {
	url := normalizeWSURL(addr)
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, err
	}
	client := &WSClient{
		Addr: url,
		Conn: newWSConn(conn),
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

func normalizeWSURL(addr string) string {
	if strings.HasPrefix(addr, "ws://") || strings.HasPrefix(addr, "wss://") {
		return addr
	}
	return fmt.Sprintf("ws://%s%s", addr, defaultWSPath)
}

func (wc *WSClient) heartbeatLoop() {
	ticker := time.NewTicker(time.Duration(PING_PONG_INTERVAL) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			wc.Conn.CheckPong()
		case <-wc.done:
			return
		}
	}
}

func (wc *WSClient) stopHeartbeat() {
	wc.stopOnce.Do(func() {
		close(wc.done)
	})
}

func (wc *WSClient) Send(dp *DataProtocol) error {
	if wc == nil || wc.Conn == nil {
		return fmt.Errorf("websocket client is nil")
	}
	return wc.Conn.Send(dp)
}

func (wc *WSClient) Close() error {
	if wc == nil || wc.Conn == nil {
		return nil
	}
	wc.stopHeartbeat()
	return wc.Conn.Close()
}
