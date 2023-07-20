package su_net

import (
	"bytes"
	"encoding/binary"
	"errors"
	slog "go/su_log"
	"sync"

	"github.com/envoyproxy/protoc-gen-validate/templates/shared"
	"gitlab.ifreetalk.com/servers/paipai_world/common_function/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/openpgp/packet"
)

type Header struct {
	Pack_len      uint32  ///整个包长度，包头加包
	Pack_id       uint32
	Route_id      uint64
	Head_uuid     uint64
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

func Encode(dpt DataProtocol) (byte_arr []byte, err error){
	byte_arr = make([]byte, 0)
	buffer := bytes.NewBuffer(byte_arr)
	if err = binary.Write(buffer, binary.BigEndian, dpt.Head.Pack_len); err != nil {
		errors.New("write pack len err")
		return
	}
	if err = binary.Write(buffer, binary.BigEndian, dpt.Head.Pack_id); err != nil {
		return
	}
	if err = binary.Write(buffer, binary.BigEndian, dpt.Head.Route_id); err != nil {
		return
	}
	if err = binary.Write(buffer, binary.BigEndian, dpt.Head.Head_uuid); err != nil {
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
	binary.Read(byteBuffer, binary.BigEndian, &dpt.Head.Pack_len)
	binary.Read(byteBuffer, binary.BigEndian, &dpt.Head.Pack_id)
	binary.Read(byteBuffer, binary.BigEndian, &dpt.Head.Route_id)
	binary.Read(byteBuffer, binary.BigEndian, &dpt.Head.Head_uuid)
	if dpt.Head.Pack_len > tmp_len{
		slog.Error("a_data length < decode length", zap.Uint32("tmp_len: ", tmp_len), zap.Uint32("decode len:", dpt.Head.Pack_len))
		err = errors.New("数据长度过短")
		return
	}
	dpt.Data = a_data[HeadLength:dpt.Head.Pack_len]
	remain_bytes = a_data[dpt.Head.Pack_len:]
	return
}
////轮询路由包
func Route(){

}
////分片ID取模