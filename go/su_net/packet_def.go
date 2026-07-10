package su_net

import (
	"encoding/binary"
	"fmt"
	"go.local/su_errors"
	slog "go.local/su_log"
	"sync"

	"go.uber.org/zap"
)

// Header 是自定义网络协议的固定 24 字节包头。
type Header struct { // 24字节
	PackLen  uint32 ///整个包长度，包头加包
	PackId   uint32
	RouteId  uint64
	HeadUuid uint64
}

// DataProtocol 表示一帧完整业务数据包。
type DataProtocol struct {
	Head Header
	Data []byte
}

const (
	HeadLength    uint32 = 24 ///包头长度
	MaxPacketSize uint32 = 4 * 1024 * 1024
	PING          uint32 = 1000
	PONG          uint32 = 1001
)

var (
	ErrIncompletePacket = su_errors.ErrIncompletePacket
	ErrInvalidPacket    = su_errors.ErrInvalidPacket
)

// deleteAllSyncMap 删除 sync.Map 中的所有键值。
func deleteAllSyncMap(m *sync.Map) {
	if m == nil {
		return
	}
	keys := make([]interface{}, 0)
	m.Range(func(k, v interface{}) bool {
		keys = append(keys, k)
		return true
	})
	for _, key := range keys {
		m.Delete(key)
	}
}

// deleteSyncMapValue 仅当 key 当前值等于 value 时删除该 key。
func deleteSyncMapValue(m *sync.Map, key interface{}, value interface{}) bool {
	if m == nil {
		return false
	}
	current, ok := m.Load(key)
	if !ok || current != value {
		return false
	}
	m.Delete(key)
	return true
}

// Ping 是心跳请求载荷。
type Ping struct {
	SendTime uint64 //////发送时间
}

// Pong 是心跳响应载荷，包含响应时间和对应 Ping 时间。
type Pong struct {
	SendTime uint64 //////发送时间
	PingTime uint64 //////ping的发送时间
}

// Encode 将 DataProtocol 编码为二进制网络包。
func Encode(dpt *DataProtocol) (byte_arr []byte, err error) {
	if dpt == nil {
		err = fmt.Errorf("%w: nil data protocol", ErrInvalidPacket)
		return nil, err
	}
	packLen := uint32(HeadLength + uint32(len(dpt.Data)))
	if packLen > MaxPacketSize {
		err = fmt.Errorf("%w: packet length %d exceeds max %d", ErrInvalidPacket, packLen, MaxPacketSize)
		return nil, err
	}
	dpt.Head.PackLen = packLen
	byte_arr = make([]byte, packLen)
	binary.BigEndian.PutUint32(byte_arr[0:4], dpt.Head.PackLen)
	binary.BigEndian.PutUint32(byte_arr[4:8], dpt.Head.PackId)
	binary.BigEndian.PutUint64(byte_arr[8:16], dpt.Head.RouteId)
	binary.BigEndian.PutUint64(byte_arr[16:24], dpt.Head.HeadUuid)
	copy(byte_arr[HeadLength:], dpt.Data)
	return
}

// Decode 解析一个网络包，并复制解析出的 payload 和剩余字节。
func Decode(a_data []byte) (remain_bytes []byte, dpt DataProtocol, err error) {
	remain_bytes, dpt, err = decodePacket(a_data, true)
	return
}

// DecodeNoCopy 解析一个网络包，payload 和剩余字节复用输入切片底层数组。
func DecodeNoCopy(a_data []byte) (remain_bytes []byte, dpt DataProtocol, err error) {
	remain_bytes, dpt, err = decodePacket(a_data, false)
	return
}

// decodePacket 执行实际包解析，copyData 控制是否复制 payload。
func decodePacket(a_data []byte, copyData bool) (remain_bytes []byte, dpt DataProtocol, err error) {
	tmp_len := uint32(len(a_data))
	if tmp_len < HeadLength {
		err = ErrIncompletePacket
		return a_data, dpt, err
	}
	dpt.Head.PackLen = binary.BigEndian.Uint32(a_data[0:4])
	if dpt.Head.PackLen < HeadLength {
		err = fmt.Errorf("%w: packet length %d < head length %d", ErrInvalidPacket, dpt.Head.PackLen, HeadLength)
		return a_data, dpt, err
	}
	if dpt.Head.PackLen > MaxPacketSize {
		err = fmt.Errorf("%w: packet length %d exceeds max %d", ErrInvalidPacket, dpt.Head.PackLen, MaxPacketSize)
		return a_data, dpt, err
	}
	if dpt.Head.PackLen > tmp_len {
		err = ErrIncompletePacket
		return a_data, dpt, err
	}
	dpt.Head.PackId = binary.BigEndian.Uint32(a_data[4:8])
	dpt.Head.RouteId = binary.BigEndian.Uint64(a_data[8:16])
	dpt.Head.HeadUuid = binary.BigEndian.Uint64(a_data[16:24])
	if copyData {
		dpt.Data = append([]byte(nil), a_data[HeadLength:dpt.Head.PackLen]...)
		remain_bytes = append([]byte(nil), a_data[dpt.Head.PackLen:]...)
	} else {
		dpt.Data = a_data[HeadLength:dpt.Head.PackLen:dpt.Head.PackLen]
		remain_bytes = a_data[dpt.Head.PackLen:]
	}
	return
}

// PingDecode 从心跳请求 payload 中解析 Ping。
func PingDecode(a_data []byte, a_pack_len uint32) (ping Ping, err error) {
	if a_pack_len < HeadLength {
		err = fmt.Errorf("%w: ping packet length %d < head length %d", ErrInvalidPacket, a_pack_len, HeadLength)
		return
	}
	data_len := uint32(len(a_data))
	if data_len != a_pack_len-HeadLength {
		slog.Error("byte length < data length", zap.Uint32("data_len: ", data_len), zap.Uint32("a_pack_len - HeadLength", a_pack_len-HeadLength))
		err = fmt.Errorf("%w: byte length %d < data length %d", ErrInvalidPacket, data_len, a_pack_len-HeadLength)
		return
	}
	if data_len != 8 {
		err = fmt.Errorf("%w: ping payload length %d", ErrInvalidPacket, data_len)
		return
	}
	ping.SendTime = binary.BigEndian.Uint64(a_data[0:8])
	return
}

// PingEncode 将 Ping 编码为心跳请求 payload。
func PingEncode(a_ping Ping) (byte_arr []byte, err error) {
	byte_arr = make([]byte, 8)
	binary.BigEndian.PutUint64(byte_arr[0:8], a_ping.SendTime)
	return
}

// PongEncode 将 Pong 编码为心跳响应 payload。
func PongEncode(a_pong Pong) (byte_arr []byte, err error) {
	byte_arr = make([]byte, 16)
	binary.BigEndian.PutUint64(byte_arr[0:8], a_pong.SendTime)
	binary.BigEndian.PutUint64(byte_arr[8:16], a_pong.PingTime)
	return
}

// PongDecode 从心跳响应 payload 中解析 Pong。
func PongDecode(a_data []byte, a_pack_len uint32) (pong Pong, err error) {
	if a_pack_len < HeadLength {
		err = fmt.Errorf("%w: pong packet length %d < head length %d", ErrInvalidPacket, a_pack_len, HeadLength)
		return
	}
	data_len := uint32(len(a_data))
	if data_len != a_pack_len-HeadLength {
		slog.Error("byte length < data length", zap.Uint32("data_len: ", data_len), zap.Uint32("a_pack_len - HeadLength", a_pack_len-HeadLength))
		err = fmt.Errorf("%w: byte length %d < data length %d", ErrInvalidPacket, data_len, a_pack_len-HeadLength)
		return
	}
	if data_len != 16 {
		err = fmt.Errorf("%w: pong payload length %d", ErrInvalidPacket, data_len)
		return
	}
	pong.SendTime = binary.BigEndian.Uint64(a_data[0:8])
	pong.PingTime = binary.BigEndian.Uint64(a_data[8:16])
	return
}
