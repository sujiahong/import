package su_net

import (
	"sync"

	"go.local/su_errors"
	slog "go.local/su_log"

	"go.uber.org/zap"
)

type HandlerContext struct {
	Conn   any
	Packet *DataProtocol

	responsePackID uint32
	responseData   []byte
	skipAutoResp   bool
}

type dataProtocolSender interface {
	Send(*DataProtocol) error
}

type packetSender interface {
	SendPacket(*DataProtocol) error
}

func (ctx *HandlerContext) SendPacket(dp *DataProtocol) error {
	if ctx == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "handler context is nil")
	}
	if dp == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "data protocol is nil")
	}
	switch conn := ctx.Conn.(type) {
	case packetSender:
		return conn.SendPacket(dp)
	case dataProtocolSender:
		return conn.Send(dp)
	default:
		return su_errors.New(su_errors.CodeInvalidArgument, "handler context conn does not support packet send")
	}
}

func (ctx *HandlerContext) SetResponse(data []byte) {
	if ctx == nil {
		return
	}
	ctx.responseData = data
}

func (ctx *HandlerContext) SkipAutoResponse() {
	if ctx == nil {
		return
	}
	ctx.skipAutoResp = true
}

func (ctx *HandlerContext) SendResponse(data []byte) error {
	if ctx == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "handler context is nil")
	}
	if ctx.Packet == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "handler context packet is nil")
	}
	if ctx.responsePackID == 0 {
		return su_errors.New(su_errors.CodeInvalidArgument, "response pack id is empty")
	}
	return ctx.SendPacket(&DataProtocol{
		Head: Header{
			PackId:   ctx.responsePackID,
			RouteId:  ctx.Packet.Head.RouteId,
			HeadUuid: ctx.Packet.Head.HeadUuid,
		},
		Data: data,
	})
}

// MessageHandler 是函数式网络消息处理器。
type MessageHandler func(ctx *HandlerContext, msg []byte) error

// 注册处理器。
type RegisterHandler interface {
	RegisterManualResponseHandler(uint32, uint32, MessageHandler) error
	RegisterRequestResponseHandler(uint32, uint32, MessageHandler) error
	RegisterOneWayHandler(uint32, MessageHandler) error
}

const (
	tcpNetRegisterManual uint32 = 1
	tcpNetRegisterAuto   uint32 = 2
	tcpNetRegisterOneWay uint32 = 3
)

type TcpNetDataHandler struct {
	registerType uint32 /////注册类型   0 不处理  1 手动回包， 2 自动回包，3 不回包
	rqId         uint32
	rsId         uint32
	handler      MessageHandler
}

func (tndh *TcpNetDataHandler) HandleMessage(ctx *HandlerContext, msg []byte) error {
	if tndh == nil || tndh.handler == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "tcp net data handler is nil")
	}
	return tndh.handler(ctx, msg)
}

type TcpNetHandler struct {
	packetHandlerMap    map[uint32]*TcpNetDataHandler
	packetHandlerMapMux sync.RWMutex
}

func newTcpNetHandler() *TcpNetHandler {
	return &TcpNetHandler{
		packetHandlerMap: make(map[uint32]*TcpNetDataHandler, 5),
	}
}

func (rh *TcpNetHandler) registerHandler(registerType uint32, dispatchPackId uint32, rqPackId uint32, rsPackId uint32, handler MessageHandler) error {
	if rh == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "tcp net handler is nil")
	}
	if handler == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "message handler is nil")
	}
	if dispatchPackId == PING || dispatchPackId == PONG {
		return su_errors.New(su_errors.CodeInvalidArgument, "cannot register control packet handler")
	}
	if (registerType == tcpNetRegisterManual || registerType == tcpNetRegisterAuto) && rsPackId == 0 {
		return su_errors.New(su_errors.CodeInvalidArgument, "response pack id is empty")
	}
	tndh := &TcpNetDataHandler{
		registerType: registerType,
		rqId:         rqPackId,
		rsId:         rsPackId,
		handler:      handler,
	}
	rh.packetHandlerMapMux.Lock()
	defer rh.packetHandlerMapMux.Unlock()
	if rh.packetHandlerMap == nil {
		rh.packetHandlerMap = make(map[uint32]*TcpNetDataHandler, 5)
	}
	if _, ok := rh.packetHandlerMap[dispatchPackId]; ok {
		return su_errors.New(su_errors.CodeInvalidArgument, "message handler already registered")
	}
	rh.packetHandlerMap[dispatchPackId] = tndh
	return nil
}

func (rh *TcpNetHandler) RegisterManualResponseHandler(rqPackId uint32, rsPackId uint32, handler MessageHandler) error {
	return rh.registerHandler(tcpNetRegisterManual, rqPackId, rqPackId, rsPackId, handler)
}

func (rh *TcpNetHandler) RegisterRequestResponseHandler(rqPackId uint32, rsPackId uint32, handler MessageHandler) error {
	return rh.registerHandler(tcpNetRegisterAuto, rqPackId, rqPackId, rsPackId, handler)
}

func (rh *TcpNetHandler) RegisterOneWayHandler(rsPackId uint32, handler MessageHandler) error {
	return rh.registerHandler(tcpNetRegisterOneWay, rsPackId, rsPackId, 0, handler)
}

func (rh *TcpNetHandler) GetTcpNetDataHandler(rsPackId uint32) (*TcpNetDataHandler, bool) {
	if rh == nil {
		return nil, false
	}
	rh.packetHandlerMapMux.RLock()
	defer rh.packetHandlerMapMux.RUnlock()
	handler, exists := rh.packetHandlerMap[rsPackId]
	return handler, exists
}

func dispatchTcpNetHandler(router *TcpNetHandler, ctx *HandlerContext) {
	if router == nil || ctx == nil || ctx.Packet == nil {
		slog.Error("tcp net handler unavailable")
		return
	}
	dp := ctx.Packet
	netHandler, ok := router.GetTcpNetDataHandler(dp.Head.PackId)
	if !ok {
		slog.Warn("tcp net handler not registered", zap.Uint32("pack_id", dp.Head.PackId), zap.Uint64("route_id", dp.Head.RouteId))
		return
	}
	ctx.responsePackID = netHandler.rsId
	if err := netHandler.HandleMessage(ctx, dp.Data); err != nil {
		slog.Error("tcp net handle message failed", zap.Uint32("pack_id", dp.Head.PackId), zap.Uint64("route_id", dp.Head.RouteId), zap.Error(err))
		return
	}
	if netHandler.registerType != tcpNetRegisterAuto || ctx.skipAutoResp {
		return
	}
	if err := ctx.SendResponse(ctx.responseData); err != nil {
		slog.Error("tcp net send auto response failed", zap.Uint32("pack_id", dp.Head.PackId), zap.Uint32("response_pack_id", netHandler.rsId), zap.Uint64("route_id", dp.Head.RouteId), zap.Error(err))
	}
}

// GNetDispatchMode 定义 gnet 包处理在协程池或事件循环内执行。
type GNetDispatchMode uint8

const (
	GNetDispatchPool GNetDispatchMode = iota
	GNetDispatchInline
)
