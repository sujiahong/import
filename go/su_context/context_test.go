package su_context

import (
	"context"
	"testing"
)

func TestContextValues(t *testing.T) {
	ctx := context.Background()
	ctx = WithRequestID(ctx, "req-1")
	ctx = WithTraceID(ctx, "trace-1")
	ctx = WithUserID(ctx, "user-1")
	ctx = WithMeta(ctx, "k", "v")

	if RequestID(ctx) != "req-1" || TraceID(ctx) != "trace-1" || UserID(ctx) != "user-1" {
		t.Fatalf("unexpected context values")
	}
	if got := Meta(ctx, "k"); got != "v" {
		t.Fatalf("Meta() = %q, want v", got)
	}
}

func TestMetadataCopy(t *testing.T) {
	ctx := WithMeta(context.Background(), "k", "v")
	meta := MetadataFrom(ctx)
	meta["k"] = "changed"
	if got := Meta(ctx, "k"); got != "v" {
		t.Fatalf("Meta() = %q, want original value", got)
	}
}
