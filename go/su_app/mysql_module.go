package su_app

import (
	"context"

	su_mysql "go.local/su_da/su_sql"
	"go.local/su_errors"
)

type MysqlConnector interface {
	Connect() error
	Close() error
}

type MysqlFactory func(cfg su_mysql.MysqlConfig) (MysqlConnector, error)

type MysqlModule struct {
	Config  su_mysql.MysqlConfig
	Factory MysqlFactory
	Client  MysqlConnector
}

func NewMysqlModule(cfg su_mysql.MysqlConfig) *MysqlModule {
	return &MysqlModule{Config: cfg}
}

func (m *MysqlModule) Name() string {
	return "mysql"
}

func (m *MysqlModule) Start(ctx context.Context) error {
	if m == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "mysql module is nil")
	}
	if m.Client == nil {
		factory := m.Factory
		if factory == nil {
			factory = defaultMysqlFactory
		}
		client, err := factory(m.Config)
		if err != nil {
			return su_errors.WrapRetryable(su_errors.CodeUnavailable, "create mysql client failed", err)
		}
		m.Client = client
	}
	if err := m.Client.Connect(); err != nil {
		return su_errors.WrapRetryable(su_errors.CodeUnavailable, "connect mysql failed", err)
	}
	return nil
}

func (m *MysqlModule) Stop(ctx context.Context) error {
	if m == nil || m.Client == nil {
		return nil
	}
	if err := m.Client.Close(); err != nil {
		return su_errors.Wrap(su_errors.CodeInternal, "close mysql failed", err)
	}
	return nil
}

func defaultMysqlFactory(cfg su_mysql.MysqlConfig) (MysqlConnector, error) {
	return su_mysql.NewMysqlClientWithConfig(cfg)
}
