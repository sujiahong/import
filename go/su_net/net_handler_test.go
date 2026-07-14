package su_net

import (
	"errors"
	"testing"
)

type fakeDataProtocolSender struct {
	sent []*DataProtocol
}

func (s *fakeDataProtocolSender) Send(dp *DataProtocol) error {
	s.sent = append(s.sent, dp)
	return nil
}

func TestTcpNetHandlerRegisterOneWayHandler(t *testing.T) {
	handler := newTcpNetHandler()
	called := false
	if err := handler.RegisterOneWayHandler(42, func(ctx *HandlerContext, req []byte) error {
		called = true
		if ctx.Packet.Head.PackId != 42 {
			t.Fatalf("PackId = %d, want 42", ctx.Packet.Head.PackId)
		}
		if string(req) != "request" {
			t.Fatalf("req = %q, want request", req)
		}
		return nil
	}); err != nil {
		t.Fatalf("RegisterOneWayHandler() error = %v", err)
	}

	sender := &fakeDataProtocolSender{}
	dispatchTcpNetHandler(handler, &HandlerContext{
		Conn:   sender,
		Packet: &DataProtocol{Head: Header{PackId: 42}, Data: []byte("request")},
	})
	if !called {
		t.Fatal("registered handler was not called")
	}
	if len(sender.sent) != 0 {
		t.Fatalf("one-way handler sent %d packets, want 0", len(sender.sent))
	}
}

func TestTcpNetHandlerAutoResponse(t *testing.T) {
	handler := newTcpNetHandler()
	if err := handler.RegisterRequestResponseHandler(42, 43, func(ctx *HandlerContext, req []byte) error {
		ctx.SetResponse([]byte("response"))
		return nil
	}); err != nil {
		t.Fatalf("RegisterRequestResponseHandler() error = %v", err)
	}

	sender := &fakeDataProtocolSender{}
	dispatchTcpNetHandler(handler, &HandlerContext{
		Conn: sender,
		Packet: &DataProtocol{
			Head: Header{PackId: 42, RouteId: 7, HeadUuid: 8},
			Data: []byte("request"),
		},
	})
	if len(sender.sent) != 1 {
		t.Fatalf("sent packets = %d, want 1", len(sender.sent))
	}
	got := sender.sent[0]
	if got.Head.PackId != 43 || got.Head.RouteId != 7 || got.Head.HeadUuid != 8 {
		t.Fatalf("response head = %+v", got.Head)
	}
	if string(got.Data) != "response" {
		t.Fatalf("response data = %q, want response", got.Data)
	}
}

func TestTcpNetHandlerManualResponse(t *testing.T) {
	handler := newTcpNetHandler()
	if err := handler.RegisterManualResponseHandler(42, 43, func(ctx *HandlerContext, req []byte) error {
		return ctx.SendResponse([]byte("manual"))
	}); err != nil {
		t.Fatalf("RegisterManualResponseHandler() error = %v", err)
	}

	sender := &fakeDataProtocolSender{}
	dispatchTcpNetHandler(handler, &HandlerContext{
		Conn: sender,
		Packet: &DataProtocol{
			Head: Header{PackId: 42, RouteId: 7, HeadUuid: 8},
			Data: []byte("request"),
		},
	})
	if len(sender.sent) != 1 {
		t.Fatalf("sent packets = %d, want 1", len(sender.sent))
	}
	got := sender.sent[0]
	if got.Head.PackId != 43 || got.Head.RouteId != 7 || got.Head.HeadUuid != 8 {
		t.Fatalf("response head = %+v", got.Head)
	}
	if string(got.Data) != "manual" {
		t.Fatalf("response data = %q, want manual", got.Data)
	}
}

func TestTcpNetHandlerErrorDoesNotAutoResponse(t *testing.T) {
	handler := newTcpNetHandler()
	if err := handler.RegisterRequestResponseHandler(42, 43, func(ctx *HandlerContext, req []byte) error {
		ctx.SetResponse([]byte("response"))
		return errors.New("handler failed")
	}); err != nil {
		t.Fatalf("RegisterRequestResponseHandler() error = %v", err)
	}

	sender := &fakeDataProtocolSender{}
	dispatchTcpNetHandler(handler, &HandlerContext{
		Conn:   sender,
		Packet: &DataProtocol{Head: Header{PackId: 42, RouteId: 7, HeadUuid: 8}},
	})
	if len(sender.sent) != 0 {
		t.Fatalf("sent packets = %d, want 0", len(sender.sent))
	}
}

func TestTcpNetHandlerSkipAutoResponse(t *testing.T) {
	handler := newTcpNetHandler()
	if err := handler.RegisterRequestResponseHandler(42, 43, func(ctx *HandlerContext, req []byte) error {
		ctx.SetResponse([]byte("response"))
		ctx.SkipAutoResponse()
		return nil
	}); err != nil {
		t.Fatalf("RegisterRequestResponseHandler() error = %v", err)
	}

	sender := &fakeDataProtocolSender{}
	dispatchTcpNetHandler(handler, &HandlerContext{
		Conn:   sender,
		Packet: &DataProtocol{Head: Header{PackId: 42, RouteId: 7, HeadUuid: 8}},
	})
	if len(sender.sent) != 0 {
		t.Fatalf("sent packets = %d, want 0", len(sender.sent))
	}
}

func TestTcpNetHandlerRejectsInvalidRegistration(t *testing.T) {
	handler := newTcpNetHandler()
	noop := func(ctx *HandlerContext, req []byte) error { return nil }
	if err := handler.RegisterOneWayHandler(42, noop); err != nil {
		t.Fatalf("RegisterOneWayHandler() error = %v", err)
	}
	if err := handler.RegisterOneWayHandler(42, noop); err == nil {
		t.Fatal("duplicate RegisterOneWayHandler() error = nil")
	}
	if err := handler.RegisterOneWayHandler(43, nil); err == nil {
		t.Fatal("nil RegisterOneWayHandler() error = nil")
	}
	if err := handler.RegisterOneWayHandler(PING, noop); err == nil {
		t.Fatal("PING RegisterOneWayHandler() error = nil")
	}
	if err := handler.RegisterOneWayHandler(PONG, noop); err == nil {
		t.Fatal("PONG RegisterOneWayHandler() error = nil")
	}
	if err := handler.RegisterRequestResponseHandler(44, 0, noop); err == nil {
		t.Fatal("empty response RegisterRequestResponseHandler() error = nil")
	}
	if err := handler.RegisterManualResponseHandler(45, 0, noop); err == nil {
		t.Fatal("empty response RegisterManualResponseHandler() error = nil")
	}
}

func TestHandlerContextSendPacket(t *testing.T) {
	sender := &fakeDataProtocolSender{}
	ctx := &HandlerContext{Conn: sender}
	dp := &DataProtocol{Head: Header{PackId: 42}, Data: []byte("payload")}
	if err := ctx.SendPacket(dp); err != nil {
		t.Fatalf("SendPacket() error = %v", err)
	}
	if len(sender.sent) != 1 || sender.sent[0] != dp {
		t.Fatal("SendPacket() did not send original packet")
	}
}
