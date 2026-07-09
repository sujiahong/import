package su_context

import (
	"context"
	"sync"
)

type contextKey string

const (
	requestIDKey contextKey = "request_id"
	traceIDKey   contextKey = "trace_id"
	userIDKey    contextKey = "user_id"
	metadataKey  contextKey = "metadata"
)

func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

func RequestID(ctx context.Context) string {
	return stringValue(ctx, requestIDKey)
}

func WithTraceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, traceIDKey, id)
}

func TraceID(ctx context.Context) string {
	return stringValue(ctx, traceIDKey)
}

func WithUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

func UserID(ctx context.Context) string {
	return stringValue(ctx, userIDKey)
}

func WithMeta(ctx context.Context, key, value string) context.Context {
	meta := MetadataFrom(ctx)
	meta[key] = value
	return context.WithValue(ctx, metadataKey, meta)
}

func Meta(ctx context.Context, key string) string {
	return MetadataFrom(ctx)[key]
}

func MetadataFrom(ctx context.Context) map[string]string {
	if ctx == nil {
		return map[string]string{}
	}
	meta, _ := ctx.Value(metadataKey).(map[string]string)
	copied := make(map[string]string, len(meta)+1)
	for k, v := range meta {
		copied[k] = v
	}
	return copied
}

func stringValue(ctx context.Context, key contextKey) string {
	if ctx == nil {
		return ""
	}
	value, _ := ctx.Value(key).(string)
	return value
}

type SafeMetadata struct {
	mu   sync.RWMutex
	data map[string]string
}

func NewSafeMetadata() *SafeMetadata {
	return &SafeMetadata{data: make(map[string]string)}
}

func (m *SafeMetadata) Set(key, value string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value
}

func (m *SafeMetadata) Get(key string) string {
	if m == nil {
		return ""
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.data[key]
}

func (m *SafeMetadata) Snapshot() map[string]string {
	if m == nil {
		return map[string]string{}
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	copied := make(map[string]string, len(m.data))
	for k, v := range m.data {
		copied[k] = v
	}
	return copied
}
