package su_net

import (
	"encoding/binary"
	"testing"
)

func TestRecvWaitsForSplitPacket(t *testing.T) {
	encoded, err := Encode(&DataProtocol{Head: Header{PackId: 7}, Data: []byte("payload")})
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	gnc := NewGnetConn(nil)
	var handled []DataProtocol
	handler := func(dp *DataProtocol) {
		handled = append(handled, *dp)
	}

	gnc.Recv(encoded[:10], handler)
	if len(handled) != 0 {
		t.Fatalf("handled %d packets before full packet arrived", len(handled))
	}

	gnc.Recv(encoded[10:], handler)
	if len(handled) != 1 {
		t.Fatalf("handled %d packets, want 1", len(handled))
	}
	if handled[0].Head.PackId != 7 || string(handled[0].Data) != "payload" {
		t.Fatalf("handled packet = %+v data=%q", handled[0].Head, handled[0].Data)
	}
}

func TestRecvHandlesStickyPackets(t *testing.T) {
	first, err := Encode(&DataProtocol{Head: Header{PackId: 1}, Data: []byte("one")})
	if err != nil {
		t.Fatalf("Encode(first) error = %v", err)
	}
	second, err := Encode(&DataProtocol{Head: Header{PackId: 2}, Data: []byte("two")})
	if err != nil {
		t.Fatalf("Encode(second) error = %v", err)
	}

	gnc := NewGnetConn(nil)
	var ids []uint32
	gnc.Recv(append(first, second...), func(dp *DataProtocol) {
		ids = append(ids, dp.Head.PackId)
	})

	if len(ids) != 2 || ids[0] != 1 || ids[1] != 2 {
		t.Fatalf("ids = %v, want [1 2]", ids)
	}
}

func TestRecvInvalidPacketClearsBuffer(t *testing.T) {
	buf := make([]byte, HeadLength)
	binary.BigEndian.PutUint32(buf[0:4], HeadLength-1)

	gnc := NewGnetConn(nil)
	gnc.Recv(buf, func(dp *DataProtocol) {
		t.Fatalf("handler should not be called for invalid packet")
	})

	if len(gnc.recvData) != 0 {
		t.Fatalf("recvData length = %d, want 0", len(gnc.recvData))
	}
}

func TestCloseClearsPingPongMapWithoutConn(t *testing.T) {
	gnc := NewGnetConn(nil)
	gnc.PingPongMap.Store(uint64(1), 1)
	gnc.PingPongMap.Store(uint64(2), 1)

	gnc.Close()

	count := 0
	gnc.PingPongMap.Range(func(k, v interface{}) bool {
		count++
		return true
	})
	if count != 0 {
		t.Fatalf("PingPongMap count = %d, want 0", count)
	}
}
