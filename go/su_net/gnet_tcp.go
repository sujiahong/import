package su_net

import (
	"fmt"
	"go.local/su_errors"
	"reflect"
	"time"

	"github.com/golang/protobuf/proto"
)

// HandleFuncType 是 gnet typed 模式下的请求/响应处理函数。
type HandleFuncType func(*GNetConn, uint64, proto.Message, proto.Message)

// GNetDispatchMode 定义 gnet 包处理在协程池或事件循环内执行。
type GNetDispatchMode uint8

const (
	GNetDispatchPool GNetDispatchMode = iota
	GNetDispatchInline
)

// HandlerFuncST 保存一个请求包 ID 到响应包 ID 的 proto 处理注册信息。
type HandlerFuncST struct {
	RQ         proto.Message  // 请求 proto 模板。
	RQPackId   uint32         // 请求包 ID。
	RS         proto.Message  // 响应 proto 模板。
	RSPackId   uint32         // 响应包 ID。
	HandleFunc HandleFuncType // 业务处理函数。
	RQType     reflect.Type   // 请求 proto 元素类型。
	RSType     reflect.Type   // 响应 proto 元素类型。
}

// pendingGNetRequest 保存等待响应的原始请求消息及创建时间。
type pendingGNetRequest struct {
	rq        proto.Message // 等待响应的请求消息副本。
	createdAt time.Time     // pending 创建时间。
}

// newProtoType 校验 proto 模板并返回其元素类型，用于后续反射创建消息。
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

// newProtoFromType 根据注册的 proto 元素类型创建新消息实例。
func newProtoFromType(t reflect.Type) proto.Message {
	if t == nil {
		return nil
	}
	msg, _ := reflect.New(t).Interface().(proto.Message)
	return msg
}
