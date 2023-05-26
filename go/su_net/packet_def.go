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
	packet_id     uint32
	data_len      uint32
	route_id      uint64
	head_uuid     uint64
}

type DataProtocol struct {
	head Header
	data []byte
}

const (
	HeadLength uint32 = 24 ///包头长度
	PING       uint32 = 1000
	PONG       uint32 = 1001
)

func Encode() {

}

func Decode(a_data []byte) ([]byte, error){
	tmp_len := uint32(len(a_data))
	if tmp_len < HeadLength{
		return nil, errors.New("data length < head lenght")
	}
	byteBuffer := bytes.NewBuffer(a_data)
	var packet_id, data_len uint32
	var route_id, head_uuid uint64
	binary.Read(byteBuffer, binary.BigEndian, &packet_id)
	binary.Read(byteBuffer, binary.BigEndian, &data_len)
	binary.Read(byteBuffer, binary.BigEndian, &route_id)
	binary.Read(byteBuffer, binary.BigEndian, &head_uuid)
	if data_len + HeadLength > tmp_len{
		slog.Error("a_data length < decode length", zap.Uint32("tmp_len: ", tmp_len), zap.Uint32("decode len:", data_len + HeadLength))
		return nil, errors.New("数据长度过短")
	}
	
}