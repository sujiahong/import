package su_net

import (
	"encoding/binary"
	"errors"
	"testing"
)

func TestEncodeDecodeRoundTrip(t *testing.T) {
	dp := &DataProtocol{
		Head: Header{
			PackId:   10000,
			RouteId:  123,
			HeadUuid: 456,
		},
		Data: []byte("hello"),
	}

	encoded, err := Encode(dp)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	if got, want := uint32(len(encoded)), HeadLength+uint32(len(dp.Data)); got != want {
		t.Fatalf("encoded length = %d, want %d", got, want)
	}
	if dp.Head.PackLen != uint32(len(encoded)) {
		t.Fatalf("PackLen = %d, want %d", dp.Head.PackLen, len(encoded))
	}

	remain, got, err := Decode(encoded)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if len(remain) != 0 {
		t.Fatalf("remain length = %d, want 0", len(remain))
	}
	if got.Head != dp.Head {
		t.Fatalf("head = %+v, want %+v", got.Head, dp.Head)
	}
	if string(got.Data) != string(dp.Data) {
		t.Fatalf("data = %q, want %q", got.Data, dp.Data)
	}
}

func TestDecodeShortHeaderIsIncomplete(t *testing.T) {
	remain, _, err := Decode([]byte{1, 2, 3})
	if !errors.Is(err, ErrIncompletePacket) {
		t.Fatalf("Decode() error = %v, want ErrIncompletePacket", err)
	}
	if len(remain) != 3 {
		t.Fatalf("remain length = %d, want 3", len(remain))
	}
}

func TestDecodeIncompleteBody(t *testing.T) {
	buf := make([]byte, HeadLength)
	binary.BigEndian.PutUint32(buf[0:4], HeadLength+10)

	remain, _, err := Decode(buf)
	if !errors.Is(err, ErrIncompletePacket) {
		t.Fatalf("Decode() error = %v, want ErrIncompletePacket", err)
	}
	if len(remain) != int(HeadLength) {
		t.Fatalf("remain length = %d, want %d", len(remain), HeadLength)
	}
}

func TestDecodeInvalidSmallPacketLength(t *testing.T) {
	buf := make([]byte, HeadLength)
	binary.BigEndian.PutUint32(buf[0:4], HeadLength-1)

	_, _, err := Decode(buf)
	if !errors.Is(err, ErrInvalidPacket) {
		t.Fatalf("Decode() error = %v, want ErrInvalidPacket", err)
	}
}

func TestDecodeInvalidLargePacketLength(t *testing.T) {
	buf := make([]byte, HeadLength)
	binary.BigEndian.PutUint32(buf[0:4], MaxPacketSize+1)

	_, _, err := Decode(buf)
	if !errors.Is(err, ErrInvalidPacket) {
		t.Fatalf("Decode() error = %v, want ErrInvalidPacket", err)
	}
}

func TestDecodeMultiplePackets(t *testing.T) {
	first, err := Encode(&DataProtocol{Head: Header{PackId: 1}, Data: []byte("one")})
	if err != nil {
		t.Fatalf("Encode(first) error = %v", err)
	}
	second, err := Encode(&DataProtocol{Head: Header{PackId: 2}, Data: []byte("two")})
	if err != nil {
		t.Fatalf("Encode(second) error = %v", err)
	}

	remain, dp, err := Decode(append(first, second...))
	if err != nil {
		t.Fatalf("Decode(first) error = %v", err)
	}
	if dp.Head.PackId != 1 {
		t.Fatalf("first PackId = %d, want 1", dp.Head.PackId)
	}

	remain, dp, err = Decode(remain)
	if err != nil {
		t.Fatalf("Decode(second) error = %v", err)
	}
	if dp.Head.PackId != 2 {
		t.Fatalf("second PackId = %d, want 2", dp.Head.PackId)
	}
	if len(remain) != 0 {
		t.Fatalf("remain length = %d, want 0", len(remain))
	}
}

func TestPingPongEncodeDecode(t *testing.T) {
	pingBytes, err := PingEncode(Ping{SendTime: 11})
	if err != nil {
		t.Fatalf("PingEncode() error = %v", err)
	}
	ping, err := PingDecode(pingBytes, HeadLength+uint32(len(pingBytes)))
	if err != nil {
		t.Fatalf("PingDecode() error = %v", err)
	}
	if ping.SendTime != 11 {
		t.Fatalf("ping.SendTime = %d, want 11", ping.SendTime)
	}

	pongBytes, err := PongEncode(Pong{SendTime: 22, PingTime: 11})
	if err != nil {
		t.Fatalf("PongEncode() error = %v", err)
	}
	pong, err := PongDecode(pongBytes, HeadLength+uint32(len(pongBytes)))
	if err != nil {
		t.Fatalf("PongDecode() error = %v", err)
	}
	if pong.SendTime != 22 || pong.PingTime != 11 {
		t.Fatalf("pong = %+v, want SendTime=22 PingTime=11", pong)
	}
}
