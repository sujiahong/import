package su_net

import (
	"errors"
	slog "go.local/su_log"
	"sync"
	"sync/atomic"
	//"go.local/my_util"
	"time"

	"github.com/panjf2000/gnet/v2"
	"go.uber.org/zap"
)

// //gnet网络连接结构
type GNetConn struct {
	Gconn        gnet.Conn
	RemoteAddr   string
	LocalAddr    string
	closed       int32
	recvData     []byte ////网络数据缓存
	closeOnce    sync.Once
	checkTimes   int32 /// 检测心跳次数
	pendingPings int32
	PingPongMap  sync.Map /////注册处理映射
}

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

func (gnc *GNetConn) Send(a_data []byte) error {
	if gnc == nil || gnc.Gconn == nil {
		return errors.New("gnet conn is nil")
	}
	if atomic.LoadInt32(&gnc.closed) == 1 {
		return errors.New("gnet conn is closed")
	}
	if err := gnc.Gconn.AsyncWrite(a_data, nil); err != nil {
		slog.Error("gnet async write failed", zap.Error(err))
		return err
	}
	return nil
}

func (gnc *GNetConn) SendPacket(dp *DataProtocol) error {
	if dp == nil {
		return errors.New("nil data protocol")
	}
	bs, err := Encode(dp)
	if err != nil {
		return err
	}
	return gnc.Send(bs)
}

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

func (gnc *GNetConn) markClosed() {
	if gnc == nil {
		return
	}
	atomic.StoreInt32(&gnc.closed, 1)
	gnc.ClearHeartbeat()
}

func (gnc *GNetConn) ClearHeartbeat() {
	if gnc == nil {
		return
	}
	deleteAllSyncMap(&gnc.PingPongMap)
	atomic.StoreInt32(&gnc.pendingPings, 0)
	atomic.StoreInt32(&gnc.checkTimes, 0)
}
