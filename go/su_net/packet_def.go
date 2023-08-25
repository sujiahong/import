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
	Send_time      uint64 //////发送时间
}

type Pong struct {
	Send_time      uint64 //////发送时间
}

func Encode(dpt DataProtocol) (byte_arr []byte, err error){
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
	slog.Info("打印", zap.Any("dpt: ", dpt))
	dpt.Data = a_data[HeadLength:dpt.Head.PackLen]
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
	binary.Read(byteBuffer, binary.BigEndian, &ping.Send_time)
	return
}
func PingEncode(a_ping Ping)(byte_arr []byte, err error) {
	byte_arr = make([]byte, 0)
	buffer := bytes.NewBuffer(byte_arr)
	if err = binary.Write(buffer, binary.BigEndian, a_ping.Send_time); err != nil {
		slog.Error("write a_ping.Send_time err", zap.Error(err))
		err = errors.New("write a_ping.Send_time err")
		return
	}
	byte_arr = buffer.Bytes()
	return
}
func PongEncode(a_pong Pong)(byte_arr []byte, err error) {
	byte_arr = make([]byte, 0)
	buffer := bytes.NewBuffer(byte_arr)
	if err = binary.Write(buffer, binary.BigEndian, a_pong.Send_time); err != nil {
		slog.Error("write a_pong.Send_time err", zap.Error(err))
		err = errors.New("write a_pong.Send_time err")
		return
	}
	byte_arr = buffer.Bytes()
	return
}
////轮询路由包
func Route(){

}

type HandlerFuncST struct {

}