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

// WSServer 提供 WebSocket HTTP upgrade、连接管理和业务包分发。
type WSServer struct {
	Addr         string             // 实际监听地址。
	Path         string             // WebSocket HTTP path。
	server       *http.Server       // HTTP server。
	listener     net.Listener       // TCP listener。
	conns        map[uint64]*WSConn // 连接 ID 到连接的映射。
	connsMu      sync.Mutex         // 保护 conns。
	nextConnID   uint64             // 递增连接 ID。
	pool         *su_util.GoPool    // 业务包处理 worker 池。
	closeOnce    sync.Once          // 保证 Close 只执行一次。
	upgrader     websocket.Upgrader // HTTP 到 WebSocket 的升级器。
	writeTimeout int64              // 写超时，存储为 time.Duration 的 int64。
	dataHandler  *TcpNetHandler     // 业务数据包处理函数。
}

// CreateWSServer 使用默认配置创建并启动 WebSocket server。
func CreateWSServer(addr string) (*WSServer, error) {
	return CreateWSServerWithConfig(addr, DefaultWSNetConfig())
}

// CreateWSServerWithConfig 使用指定配置创建并启动 WebSocket server。
func CreateWSServerWithConfig(addr string, cfg WSNetConfig) (*WSServer, error) {
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
		dataHandler:  newTcpNetHandler(),
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

// handleHTTP 处理 WebSocket upgrade，并为连接启动读循环。
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
			taskDP := *dp
			taskDP.Data = append([]byte(nil), dp.Data...)
			if !ws.pool.SendTask(taskDP.Head.RouteId, func() {
				ws.HandleMessage(conn, &taskDP)
			}) {
				slog.Warn("websocket server task dropped", zap.Uint64("route_id", taskDP.Head.RouteId))
			}
		})
	}()
}

// Close 关闭 HTTP server、所有 WebSocket 连接和 worker 池。
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

// ConnCount 返回当前存活的 WebSocket 连接数。
func (ws *WSServer) ConnCount() int {
	if ws == nil {
		return 0
	}
	ws.connsMu.Lock()
	defer ws.connsMu.Unlock()
	return len(ws.conns)
}

// SetWriteTimeout 更新服务端默认写超时，并同步到当前所有连接。
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

// WriteTimeout 返回服务端当前默认写超时。
func (ws *WSServer) WriteTimeout() time.Duration {
	if ws == nil {
		return 0
	}
	return time.Duration(atomic.LoadInt64(&ws.writeTimeout))
}

func (ws *WSServer) RegisterManualResponseHandler(rqPackId uint32, rsPackId uint32, handler MessageHandler) error {
	if ws == nil || ws.dataHandler == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "ws or ws.dataHandler is nil")
	}
	return ws.dataHandler.RegisterManualResponseHandler(rqPackId, rsPackId, handler)
}

func (ws *WSServer) RegisterRequestResponseHandler(rqPackId uint32, rsPackId uint32, handler MessageHandler) error {
	if ws == nil || ws.dataHandler == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "ws or ws.dataHandler is nil")
	}
	return ws.dataHandler.RegisterRequestResponseHandler(rqPackId, rsPackId, handler)
}

func (ws *WSServer) RegisterOneWayHandler(packId uint32, handler MessageHandler) error {
	if ws == nil || ws.dataHandler == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "ws or ws.dataHandler is nil")
	}
	return ws.dataHandler.RegisterOneWayHandler(packId, handler)
}

func (ws *WSServer) HandleMessage(conn *WSConn, dp *DataProtocol) {
	if ws == nil || ws.dataHandler == nil || dp == nil {
		slog.Error("websocket server handler unavailable")
		return
	}
	dispatchTcpNetHandler(ws.dataHandler, &HandlerContext{Conn: conn, Packet: dp})
}
