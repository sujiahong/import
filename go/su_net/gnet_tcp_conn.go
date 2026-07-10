package su_net

import (
	"errors"
	"go.local/su_errors"
	slog "go.local/su_log"
	"sync"
	"sync/atomic"
	//"go.local/su_util"
	"time"

	"github.com/panjf2000/gnet/v2"
	"go.uber.org/zap"
)

// GNetRawHandler 处理 gnet 原始 DataProtocol 数据包。
type GNetRawHandler func(*GNetConn, *DataProtocol)

// GNetConn 封装 gnet.Conn，并维护收包缓存、地址信息和心跳状态。
type GNetConn struct {
	Gconn        gnet.Conn // 底层 gnet 连接。
	RemoteAddr   string    // 远端地址字符串。
	LocalAddr    string    // 本地地址字符串。
	closed       int32     // 连接是否已关闭，按 atomic 访问。
	recvData     []byte    // gnet 收包半包/粘包缓存。
	closeOnce    sync.Once // 保证 Close 只执行一次。
	checkTimes   int32     // 连续检测到未完成心跳的次数。
	pendingPings int32     // 尚未收到 PONG 的心跳数量。
	PingPongMap  sync.Map  // Ping 发送时间到占位值的映射。
}

// NewGnetConn 创建 gnet 连接包装并缓存本地/远端地址。
func NewGnetConn(c gnet.Conn) *GNetConn {
	gnc := &GNetConn{
		Gconn:    c,
		recvData: make([]byte, 0, 4096),
	}
	if c != nil {
		gnc.RemoteAddr = c.RemoteAddr().String()
		gnc.LocalAddr = c.LocalAddr().String()
	}
	return gnc
}

// Send 通过 gnet 异步写接口发送已编码数据。
func (gnc *GNetConn) Send(a_data []byte) error {
	if gnc == nil || gnc.Gconn == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "gnet conn is nil")
	}
	if atomic.LoadInt32(&gnc.closed) == 1 {
		return su_errors.New(su_errors.CodeUnavailable, "gnet conn is closed")
	}
	if err := gnc.Gconn.AsyncWrite(a_data, nil); err != nil {
		slog.Error("gnet async write failed", zap.Error(err))
		return su_errors.WrapRetryable(su_errors.CodeUnavailable, "gnet async write failed", err)
	}
	return nil
}

// SendPacket 编码 DataProtocol 后发送。
func (gnc *GNetConn) SendPacket(dp *DataProtocol) error {
	if dp == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "nil data protocol")
	}
	bs, err := Encode(dp)
	if err != nil {
		return err
	}
	return gnc.Send(bs)
}

// Recv 处理 gnet 读到的帧数据，支持粘包/半包解析。
func (gnc *GNetConn) Recv(frame []byte, a_handle_func func(a_dp *DataProtocol)) {
	gnc.recvData = append(gnc.recvData, frame...)
	var err error
	for {
		if len(gnc.recvData) < int(HeadLength) {
			return
		}
		var dp DataProtocol
		gnc.recvData, dp, err = Decode(gnc.recvData)
		if err != nil {
			if errors.Is(err, ErrIncompletePacket) {
				return
			}
			slog.Error("decode data failed", zap.Error(err))
			gnc.recvData = gnc.recvData[:0]
			gnc.Close()
			return
		}
		a_handle_func(&dp)
		if len(gnc.recvData) <= 0 {
			return
		}
	}
}

// Ping 发送一次应用层心跳请求。
func (gnc *GNetConn) Ping() error {
	routeID := nextRouteID()
	micro_time := uint64(time.Now().UnixNano() / 1000)
	var rq_dp DataProtocol
	var err error
	rq_dp.Head.PackId = 1000
	rq_dp.Head.RouteId = routeID
	rq_dp.Head.HeadUuid = micro_time
	ping := Ping{SendTime: micro_time}
	rq_dp.Data, err = PingEncode(ping)
	if err != nil {
		slog.Error("Ping 封包失败", zap.Error(err))
		return err
	}
	rq_bytes, err := Encode(&rq_dp)
	if err != nil {
		slog.Error("rq_dp 封包失败", zap.Error(err))
		return err
	}
	gnc.PingPongMap.Store(ping.SendTime, 1)
	atomic.AddInt32(&gnc.pendingPings, 1)
	if err := gnc.Send(rq_bytes); err != nil {
		if _, ok := gnc.PingPongMap.LoadAndDelete(ping.SendTime); ok {
			atomic.AddInt32(&gnc.pendingPings, -1)
		}
		return err
	}
	return nil
}

// CheckPong 检查未完成心跳并在连续未响应时关闭连接。
func (gnc *GNetConn) CheckPong() {
	if gnc == nil || atomic.LoadInt32(&gnc.closed) == 1 {
		return
	}
	count := atomic.LoadInt32(&gnc.pendingPings)
	var checkTimes int32
	if count == 0 {
		atomic.StoreInt32(&gnc.checkTimes, 0)
	} else {
		checkTimes = atomic.AddInt32(&gnc.checkTimes, 1)
	}
	slog.Info("检测", zap.Int32("gnc.checkTimes", atomic.LoadInt32(&gnc.checkTimes)), zap.Int32("count", count))
	if err := gnc.Ping(); err != nil {
		slog.Error("gnet ping failed", zap.Error(err))
		gnc.Close()
		return
	}
	if count > 0 && checkTimes >= 2 {
		/////断开连接,重连
		gnc.Close()
	}
}

// Close 关闭 gnet 连接并清理心跳状态。
func (gnc *GNetConn) Close() {
	if gnc == nil {
		return
	}
	gnc.closeOnce.Do(func() {
		atomic.StoreInt32(&gnc.closed, 1)
		gnc.ClearHeartbeat()
		if gnc.Gconn == nil {
			return
		}
		gnc.Gconn.Close()
	})
}

// markClosed 标记连接已关闭，用于 gnet OnClose 回调中避免重复关闭底层连接。
func (gnc *GNetConn) markClosed() {
	if gnc == nil {
		return
	}
	atomic.StoreInt32(&gnc.closed, 1)
	gnc.ClearHeartbeat()
}

// ClearHeartbeat 清空未收到响应的心跳记录。
func (gnc *GNetConn) ClearHeartbeat() {
	if gnc == nil {
		return
	}
	deleteAllSyncMap(&gnc.PingPongMap)
	atomic.StoreInt32(&gnc.pendingPings, 0)
	atomic.StoreInt32(&gnc.checkTimes, 0)
}
