package su_net

import (
	"fmt"
	"go.local/su_errors"
	"reflect"
	"time"

	"github.com/golang/protobuf/proto"
)

type HandleFuncType func(*GNetConn, uint64, proto.Message, proto.Message)
type GNetDispatchMode uint8

const (
	GNetDispatchPool GNetDispatchMode = iota
	GNetDispatchInline
)

// / 业务处理函数结构
type HandlerFuncST struct {
	RQ         proto.Message
	RQPackId   uint32
	RS         proto.Message
	RSPackId   uint32
	HandleFunc HandleFuncType
	RQType     reflect.Type
	RSType     reflect.Type
}

type pendingGNetRequest struct {
	rq        proto.Message
	createdAt time.Time
}

func newProtoType(template proto.Message) (reflect.Type, error) {
	if template == nil {
		return nil, su_errors.New(su_errors.CodeInvalidArgument, "nil proto template")
	}
	t := reflect.TypeOf(template)
	if t.Kind() != reflect.Ptr {
		return nil, su_errors.New(su_errors.CodeInvalidArgument, fmt.Sprintf("proto template must be pointer, got %s", t.Kind()))
	}
	elem := t.Elem()
	if _, ok := reflect.New(elem).Interface().(proto.Message); !ok {
		return nil, su_errors.New(su_errors.CodeInvalidArgument, fmt.Sprintf("%s does not implement proto.Message", t.String()))
	}
	return elem, nil
}

func newProtoFromType(t reflect.Type) proto.Message {
	if t == nil {
		return nil
	}
	msg, _ := reflect.New(t).Interface().(proto.Message)
	return msg
}
