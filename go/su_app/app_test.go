package su_app

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"go.local/su_errors"
)

type testModule struct {
	name     string
	events   *[]string
	startErr error
	stopErr  error
}

func (m testModule) Name() string { return m.name }
func (m testModule) Start(ctx context.Context) error {
	*m.events = append(*m.events, "start:"+m.name)
	return m.startErr
}
func (m testModule) Stop(ctx context.Context) error {
	*m.events = append(*m.events, "stop:"+m.name)
	return m.stopErr
}

func TestAppStartStopOrder(t *testing.T) {
	var events []string
	app := New("svc")
	app.Register(testModule{name: "a", events: &events})
	app.Register(testModule{name: "b", events: &events})

	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := app.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	want := []string{"start:a", "start:b", "stop:b", "stop:a"}
	if !reflect.DeepEqual(events, want) {
		t.Fatalf("events = %v, want %v", events, want)
	}
}

func TestAppStartWrapsModuleErrorCode(t *testing.T) {
	var events []string
	startErr := errors.New("boom")
	app := New("svc")
	app.Register(testModule{name: "a", events: &events, startErr: startErr})

	err := app.Start(context.Background())
	if su_errors.CodeOf(err) != su_errors.CodeInternal {
		t.Fatalf("Start() code = %d, want internal", su_errors.CodeOf(err))
	}
	if !errors.Is(err, startErr) {
		t.Fatal("Start() error should wrap module error")
	}
}
