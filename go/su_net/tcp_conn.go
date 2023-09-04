package su_net

import (
	slog "go/su_log"
	"sync"
	// "sync/atomic"
	//"go/my_util"
	"time"
	"github.com/panjf2000/gnet"
	"go.uber.org/zap"
)

////gnet网络连接结构
type GNetConn struct {
	Gconn gnet.Conn
	state     int32       /////是否使用 1 使用  0 未使用
	recvData []byte     ////网络数据缓存
	checkTimes uint8    /// 检测心跳次数
	PingPongMap sync.Map      /////注册处理映射 
}

func NewGnetConn(c gnet.Conn) *GNetConn {
	gnc := &GNetConn{Gconn: c, recvData: make([]byte, 0, 0), state: 0, checkTimes: 0}
	return gnc
}

func (gnc *GNetConn)Send(a_data []byte){
	gnc.Gconn.AsyncWrite(a_data)
}

func (gnc *GNetConn)Recv(frame []byte, a_handle_func func(a_dp *DataProtocol)){
	gnc.recvData = append(gnc.recvData, frame...)
	var err error
	for {
		var dp DataProtocol
		gnc.recvData, dp, err = Decode(gnc.recvData)
		if err != nil {
			slog.Error("decode data failed", zap.Error(err))
			return
		}
		a_handle_func(&dp)
		slog.Info("剩余数据长度", zap.Int("recv data len:", len(gnc.recvData)), zap.Any("dp: ", dp))
		if len(gnc.recvData) <= 0 {
			return
		}
	}
}

func (gnc *GNetConn)Ping(){
	micro_time := uint64(time.Now().UnixNano()/1000)
	var rq_dp DataProtocol
	var err error
	rq_dp.Head.PackId = 1000
	rq_dp.Head.RouteId = micro_time
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
	gnc.Send(rq_bytes)
	gnc.PingPongMap.Store(ping.SendTime, 1)
}

func (gnc *GNetConn)CheckPong(){
	gnc.checkTimes++
	var count uint8 = 0
	gnc.PingPongMap.Range(func(k, v interface{})bool {
		count++
		return true
	})
	slog.Info("检测", zap.Any("gnc.checkTimes: ", gnc.checkTimes), zap.Any("count: ", count))
	gnc.Ping()
	if count == 0 {
		gnc.checkTimes = 0
	}else{
		if gnc.checkTimes >= 2 {
			/////断开连接,重连
			gnc.Close()
		}
	}
}

func (gnc *GNetConn)Close(){
	gnc.Gconn.Close()
}