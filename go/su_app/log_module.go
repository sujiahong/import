package su_app

import (
	"context"

	"go.local/su_errors"
	"go.local/su_log"
)

type LogModule struct {
	FileName string
}

func NewLogModule(fileName string) *LogModule {
	return &LogModule{FileName: fileName}
}

func (m *LogModule) Name() string {
	return "log"
}

func (m *LogModule) Start(ctx context.Context) error {
	if m == nil || m.FileName == "" {
		return su_errors.New(su_errors.CodeInvalidArgument, "log module file name is empty")
	}
	su_log.Init(m.FileName)
	return nil
}

func (m *LogModule) Stop(ctx context.Context) error {
	return nil
}
