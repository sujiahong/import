package su_app

import (
	"context"

	"go.local/su_config"
	"go.local/su_errors"
)

type ConfigModule struct {
	Path      string
	EnvPrefix string
	Target    any
}

func NewConfigModule(path, envPrefix string, target any) *ConfigModule {
	return &ConfigModule{Path: path, EnvPrefix: envPrefix, Target: target}
}

func (m *ConfigModule) Name() string {
	return "config"
}

func (m *ConfigModule) Start(ctx context.Context) error {
	if m == nil || m.Target == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "config module target is nil")
	}
	switch {
	case m.Path != "" && m.EnvPrefix != "":
		if err := su_config.LoadWithEnv(m.Path, m.EnvPrefix, m.Target); err != nil {
			return su_errors.Wrap(su_errors.CodeInvalidArgument, "load config with env failed", err)
		}
		return nil
	case m.Path != "":
		if err := su_config.Load(m.Path, m.Target); err != nil {
			return su_errors.Wrap(su_errors.CodeInvalidArgument, "load config failed", err)
		}
		return nil
	case m.EnvPrefix != "":
		if err := su_config.LoadEnv(m.EnvPrefix, m.Target); err != nil {
			return su_errors.Wrap(su_errors.CodeInvalidArgument, "load config env failed", err)
		}
		return nil
	default:
		return su_errors.New(su_errors.CodeInvalidArgument, "config module path and env prefix are empty")
	}
}

func (m *ConfigModule) Stop(ctx context.Context) error {
	return nil
}
