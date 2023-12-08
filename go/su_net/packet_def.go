package su_net

import (
	"bytes"
	"encoding/binary"
	"errors"
	slog "go/su_log"
	//"sync"

	// "github.com/envoyproxy/protoc-gen-validate/templates/shared"
	// "gitlab.ifreetalk.com/servers/paipai_world/common_function/uuid"
	"go.uber.org/zap"
	//"golang.org/x/crypto/openpgp/packet"
)

type Header struct {// 24字节
	PackLen      uint32  ///整个包长度，包头加包
	PackId       uint32
	RouteId      uint64
	HeadUuid     uint64
}

type DataProtocol struct {
	Head Header
	Data []byte
}

const (
	HeadLength uint32 = 24 ///包头长度
	PING       uint32 = 1000
	PONG       uint32 = 1001
)

type Ping struct {
	SendTime      uint64 //////发送时间
}

type Pong struct {
	SendTime      uint64 //////发送时间
	PingTime      uint64 //////ping的发送时间
}

func Encode(dpt *DataProtocol) (byte_arr []byte, err error){
	byte_arr = make([]byte, 0)
	buffer := bytes.NewBuffer(byte_arr)
	if err = binary.Write(buffer, binary.BigEndian, dpt.Head.PackLen); err != nil {
		err = errors.New("write pack len err")
		return
	}
	if err = binary.Write(buffer, binary.BigEndian, dpt.Head.PackId); err != nil {
		return
	}
	if err = binary.Write(buffer, binary.BigEndian, dpt.Head.RouteId); err != nil {
		return
	}
	if err = binary.Write(buffer, binary.BigEndian, dpt.Head.HeadUuid); err != nil {
		return
	}	
	if err = binary.Write(buffer, binary.BigEndian, dpt.Data); err != nil {
		return
	}	
	byte_arr = buffer.Bytes()
	return
}
func Decode(a_data []byte) (remain_bytes []byte, dpt DataProtocol, err error){
	tmp_len := uint32(len(a_data))
	if tmp_len < HeadLength{
		err = errors.New("data length < head lenght")
		return
	}
	byteBuffer := bytes.NewBuffer(a_data)
	binary.Read(byteBuffer, binary.BigEndian, &dpt.Head.PackLen)
	binary.Read(byteBuffer, binary.BigEndian, &dpt.Head.PackId)
	binary.Read(byteBuffer, binary.BigEndian, &dpt.Head.RouteId)
	binary.Read(byteBuffer, binary.BigEndian, &dpt.Head.HeadUuid)
	if dpt.Head.PackLen > tmp_len{
		slog.Error("a_data length < decode length", zap.Uint32("tmp_len: ", tmp_len), zap.Uint32("decode len:", dpt.Head.PackLen))
		err = errors.New("数据长度过短")
		return
	}
	//slog.Info("打印", zap.Any("dpt: ", dpt))
	dpt.Data = a_data[HeadLength:dpt.Head.PackLen]
	remain_bytes = a_data[dpt.Head.PackLen:]
	return
}

func Encode1(dpt *DataProtocol) (byte_arr []byte, err error){
	byte_arr = make([]byte, HeadLength+len(dpt.Data))
	binary.BigEndian.PutUint32(byte_arr[0:4], dpt.Head.PackLen)
	binary.BigEndian.PutUint32(byte_arr[4:8], dpt.Head.PackId)
	binary.BigEndian.PutUint32(byte_arr[8:16], dpt.Head.RouteId)
	binary.BigEndian.PutUint32(byte_arr[16:24], dpt.Head.HeadUuid)
	//copy(byte_arr[24:], dpt.Data)
	byte_arr = append(byte_arr, dpt.Data)
	return
}
func Decode1(a_data []byte) (remain_bytes []byte, dpt DataProtocol, err error){
	packLen := binary.BigEndian.Uint32(a_data[0:4])
	tmp_len := uint32(len(a_data))
	if tmp_len < packLen {
		slog.Error("a_data length < decode length", zap.Uint32("tmp_len: ", tmp_len), zap.Uint32("decode len:", dpt.Head.PackLen))
		err = errors.New("数据长度过短")
		return
	}
	dpt.Head.PackLen = packLen
	dpt.Head.PackId = binary.BigEndian.Uint32(a_data[4:8])
	dpt.Head.RouteId = binary.BigEndian.Uint64(a_data[8:16])
	dpt.Head.HeadUuid = binary.BigEndian.Uint64(a_data[16:24])
 
	//slog.Info("打印", zap.Any("dpt: ", dpt))
	dpt.Data = a_data[HeadLength:packLen]
	remain_bytes = a_data[dpt.Head.PackLen:]
	return
}

func PingDecode(a_data []byte, a_pack_len uint32) (ping Ping, err error) {
	data_len := uint32(len(a_data))
	if data_len != a_pack_len - HeadLength {
		slog.Error("byte length < data length", zap.Uint32("data_len: ", data_len), zap.Uint32("a_pack_len - HeadLength", a_pack_len - HeadLength))
		err = errors.New("byte length < data length")
		return
	}
	byteBuffer := bytes.NewBuffer(a_data)
	binary.Read(byteBuffer, binary.BigEndian, &ping.SendTime)
	return
}
func PingEncode(a_ping Ping)(byte_arr []byte, err error) {
	byte_arr = make([]byte, 0)
	buffer := bytes.NewBuffer(byte_arr)
	if err = binary.Write(buffer, binary.BigEndian, a_ping.SendTime); err != nil {
		slog.Error("write a_ping.SendTime err", zap.Error(err))
		err = errors.New("write a_ping.SendTime err")
		return
	}
	byte_arr = buffer.Bytes()
	return
}
func PongEncode(a_pong Pong)(byte_arr []byte, err error) {
	byte_arr = make([]byte, 0)
	buffer := bytes.NewBuffer(byte_arr)
	if err = binary.Write(buffer, binary.BigEndian, a_pong.SendTime); err != nil {
		slog.Error("write a_pong.SendTime err", zap.Error(err))
		err = errors.New("write a_pong.SendTime err")
		return
	}
	if err = binary.Write(buffer, binary.BigEndian, a_pong.PingTime); err != nil {
		slog.Error("write a_pong.PingTime err", zap.Error(err))
		err = errors.New("write a_pong.PingTime err")
		return
	}
	byte_arr = buffer.Bytes()
	return
}
func PongDecode(a_data []byte, a_pack_len uint32) (pong Pong, err error) {
	data_len := uint32(len(a_data))
	if data_len != a_pack_len - HeadLength {
		slog.Error("byte length < data length", zap.Uint32("data_len: ", data_len), zap.Uint32("a_pack_len - HeadLength", a_pack_len - HeadLength))
		err = errors.New("byte length < data length")
		return
	}
	byteBuffer := bytes.NewBuffer(a_data)
	binary.Read(byteBuffer, binary.BigEndian, &pong.SendTime)
	binary.Read(byteBuffer, binary.BigEndian, &pong.PingTime)
	return
}
////轮询路由包
func Route(){

}