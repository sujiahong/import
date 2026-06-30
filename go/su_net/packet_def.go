package su_net

import (
	"encoding/binary"
	"errors"
	"fmt"
	slog "go.local/su_log"

	"go.uber.org/zap"
)

type Header struct { // 24字节
	PackLen  uint32 ///整个包长度，包头加包
	PackId   uint32
	RouteId  uint64
	HeadUuid uint64
}

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
	ErrIncompletePacket = errors.New("incomplete packet")
	ErrInvalidPacket    = errors.New("invalid packet")
)

type Ping struct {
	SendTime uint64 //////发送时间
}

type Pong struct {
	SendTime uint64 //////发送时间
	PingTime uint64 //////ping的发送时间
}

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

func Decode(a_data []byte) (remain_bytes []byte, dpt DataProtocol, err error) {
	remain_bytes, dpt, err = decodePacket(a_data, true)
	return
}

func DecodeNoCopy(a_data []byte) (remain_bytes []byte, dpt DataProtocol, err error) {
	remain_bytes, dpt, err = decodePacket(a_data, false)
	return
}

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

func PingDecode(a_data []byte, a_pack_len uint32) (ping Ping, err error) {
	if a_pack_len < HeadLength {
		err = fmt.Errorf("%w: ping packet length %d < head length %d", ErrInvalidPacket, a_pack_len, HeadLength)
		return
	}
	data_len := uint32(len(a_data))
	if data_len != a_pack_len-HeadLength {
		slog.Error("byte length < data length", zap.Uint32("data_len: ", data_len), zap.Uint32("a_pack_len - HeadLength", a_pack_len-HeadLength))
		err = errors.New("byte length < data length")
		return
	}
	if data_len != 8 {
		err = fmt.Errorf("%w: ping payload length %d", ErrInvalidPacket, data_len)
		return
	}
	ping.SendTime = binary.BigEndian.Uint64(a_data[0:8])
	return
}
func PingEncode(a_ping Ping) (byte_arr []byte, err error) {
	byte_arr = make([]byte, 8)
	binary.BigEndian.PutUint64(byte_arr[0:8], a_ping.SendTime)
	return
}
func PongEncode(a_pong Pong) (byte_arr []byte, err error) {
	byte_arr = make([]byte, 16)
	binary.BigEndian.PutUint64(byte_arr[0:8], a_pong.SendTime)
	binary.BigEndian.PutUint64(byte_arr[8:16], a_pong.PingTime)
	return
}
func PongDecode(a_data []byte, a_pack_len uint32) (pong Pong, err error) {
	if a_pack_len < HeadLength {
		err = fmt.Errorf("%w: pong packet length %d < head length %d", ErrInvalidPacket, a_pack_len, HeadLength)
		return
	}
	data_len := uint32(len(a_data))
	if data_len != a_pack_len-HeadLength {
		slog.Error("byte length < data length", zap.Uint32("data_len: ", data_len), zap.Uint32("a_pack_len - HeadLength", a_pack_len-HeadLength))
		err = errors.New("byte length < data length")
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

// //轮询路由包
func Route() {

}
