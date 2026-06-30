package su_net

import (
	"errors"
	slog "go.local/su_log"
	"sync"
	// "sync/atomic"
	//"go.local/my_util"
	"github.com/panjf2000/gnet"
	"go.uber.org/zap"
	"time"
)

// //gnet网络连接结构
type GNetConn struct {
	Gconn       gnet.Conn
	state       int32    /////是否使用 1 使用  0 未使用
	recvData    []byte   ////网络数据缓存
	checkTimes  uint8    /// 检测心跳次数
	PingPongMap sync.Map /////注册处理映射
}

func NewGnetConn(c gnet.Conn) *GNetConn {
	gnc := &GNetConn{Gconn: c, recvData: make([]byte, 0, 4096), state: 0, checkTimes: 0}
	return gnc
}

func (gnc *GNetConn) Send(a_data []byte) {
	if gnc == nil || gnc.Gconn == nil {
		return
	}
	gnc.Gconn.AsyncWrite(a_data)
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

func (gnc *GNetConn) Ping() {
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
		return
	}
	rq_dp.Head.PackLen = uint32(24 + len(rq_dp.Data))
	rq_bytes, err := Encode(&rq_dp)
	if err != nil {
		slog.Error("rq_dp 封包失败", zap.Error(err))
		return
	}
	gnc.PingPongMap.Store(ping.SendTime, 1)
	gnc.Send(rq_bytes)
}

func (gnc *GNetConn) CheckPong() {
	var count uint8 = 0
	gnc.PingPongMap.Range(func(k, v interface{}) bool {
		count++
		return true
	})
	if count == 0 {
		gnc.checkTimes = 0
	} else {
		gnc.checkTimes++
	}
	slog.Info("检测", zap.Any("gnc.checkTimes: ", gnc.checkTimes), zap.Any("count: ", count))
	gnc.Ping()
	if count > 0 && gnc.checkTimes >= 2 {
		/////断开连接,重连
		gnc.Close()
	}
}

func (gnc *GNetConn) Close() {
	if gnc == nil {
		return
	}
	gnc.PingPongMap.Range(func(k, v interface{}) bool {
		gnc.PingPongMap.Delete(k)
		return true
	})
	if gnc.Gconn == nil {
		return
	}
	gnc.Gconn.Close()
}
