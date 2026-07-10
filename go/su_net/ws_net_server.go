package su_net

import (
	"context"
	"errors"
	"go.local/su_errors"
	slog "go.local/su_log"
	"go.local/su_util"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type WSServer struct {
	Addr         string
	Path         string
	server       *http.Server
	listener     net.Listener
	handler      WSHandler
	conns        map[uint64]*WSConn
	connsMu      sync.Mutex
	nextConnID   uint64
	pool         *su_util.GoPool
	closeOnce    sync.Once
	upgrader     websocket.Upgrader
	writeTimeout int64
}

func CreateWSServer(addr string, handlers ...WSHandler) (*WSServer, error) {
	return CreateWSServerWithConfig(addr, DefaultWSNetConfig(), handlers...)
}

func CreateWSServerWithConfig(addr string, cfg WSNetConfig, handlers ...WSHandler) (*WSServer, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, su_errors.WrapRetryable(su_errors.CodeUnavailable, "listen websocket failed", err)
	}
	ws := &WSServer{
		Addr:     listener.Addr().String(),
		Path:     defaultWSPath,
		listener: listener,
		conns:    make(map[uint64]*WSConn),
		pool:     su_util.NewGoPool(DEFAULT_POOL_WORKERS, DEFAULT_POOL_QUEUE_SIZE),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		writeTimeout: int64(cfg.WriteTimeout),
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
	wsConn := newWSConnWithWriteTimeout(conn, ws.WriteTimeout())
	key := atomic.AddUint64(&ws.nextConnID, 1)
	ws.connsMu.Lock()
	wsConn.SetWriteTimeout(ws.WriteTimeout())
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
			if errors.Is(err, net.ErrClosed) {
				err = nil
			}
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

func (ws *WSServer) SetWriteTimeout(timeout time.Duration) {
	if ws == nil {
		return
	}
	atomic.StoreInt64(&ws.writeTimeout, int64(timeout))
	conns := make([]*WSConn, 0)
	ws.connsMu.Lock()
	for _, conn := range ws.conns {
		conns = append(conns, conn)
	}
	ws.connsMu.Unlock()
	for _, conn := range conns {
		conn.SetWriteTimeout(timeout)
	}
}

func (ws *WSServer) WriteTimeout() time.Duration {
	if ws == nil {
		return 0
	}
	return time.Duration(atomic.LoadInt64(&ws.writeTimeout))
}
