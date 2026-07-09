package su_errors

import (
	"errors"
	"testing"
)

func TestCodeOfAndRetryable(t *testing.T) {
	base := errors.New("dial failed")
	err := WrapRetryable(CodeUnavailable, "rpc unavailable", base)

	if got := CodeOf(err); got != CodeUnavailable {
		t.Fatalf("CodeOf() = %d, want %d", got, CodeUnavailable)
	}
	if !Retryable(err) {
		t.Fatal("Retryable() = false, want true")
	}
	if !errors.Is(err, base) {
		t.Fatal("wrapped error does not match base")
	}
}

func TestCodeOfUnknown(t *testing.T) {
	if got := CodeOf(errors.New("plain")); got != CodeUnknown {
		t.Fatalf("CodeOf() = %d, want %d", got, CodeUnknown)
	}
	if got := CodeOf(nil); got != CodeOK {
		t.Fatalf("CodeOf(nil) = %d, want %d", got, CodeOK)
	}
}
