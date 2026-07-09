/*
 * @Copyright:
 * @file name: File name
 * @Data: Do not edit
 * @LastEditor:
 * @LastData:
 * @Describe:
 */
package su_mysql

import (
	"context"
	"database/sql"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"go.local/su_errors"
	slog "go.local/su_log"
	"go.uber.org/zap"
	//"time"
)

type MysqlConfig struct {
	Uname           string
	Passwd          string
	Addr            string
	DbName          string
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

type MysqlClient struct {
	Db         *sqlx.DB
	Uname      string
	Passwd     string
	Addr       string
	DbName     string
	MaxOpenCns int
	MaxIdleCns int
	cfg        MysqlConfig
	mu         sync.RWMutex
	closeOnce  sync.Once
	closeErr   error
}

func NewMysqlClient(a_uname, a_passwd, a_addr, a_dbname string, a_max_open_conns, a_max_idle_conns int) *MysqlClient {
	cfg := defaultMysqlConfig(MysqlConfig{
		Uname:        a_uname,
		Passwd:       a_passwd,
		Addr:         a_addr,
		DbName:       a_dbname,
		MaxOpenConns: a_max_open_conns,
		MaxIdleConns: a_max_idle_conns,
	})
	return &MysqlClient{
		Uname:      cfg.Uname,
		Passwd:     cfg.Passwd,
		Addr:       cfg.Addr,
		DbName:     cfg.DbName,
		MaxOpenCns: cfg.MaxOpenConns,
		MaxIdleCns: cfg.MaxIdleConns,
		cfg:        cfg,
	}
}

func NewMysqlClientWithConfig(cfg MysqlConfig) (*MysqlClient, error) {
	cfg = defaultMysqlConfig(cfg)
	if cfg.DSN == "" && (cfg.Uname == "" || cfg.Addr == "" || cfg.DbName == "") {
		return nil, su_errors.New(su_errors.CodeInvalidArgument, "mysql config is incomplete")
	}
	return &MysqlClient{
		Uname:      cfg.Uname,
		Passwd:     cfg.Passwd,
		Addr:       cfg.Addr,
		DbName:     cfg.DbName,
		MaxOpenCns: cfg.MaxOpenConns,
		MaxIdleCns: cfg.MaxIdleConns,
		cfg:        cfg,
	}, nil
}

func (mc *MysqlClient) Connect() error {
	if mc == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "mysql client is nil")
	}
	cfg := defaultMysqlConfig(mc.cfg)
	if cfg.DSN == "" {
		cfg.Uname = firstNonEmpty(cfg.Uname, mc.Uname)
		cfg.Passwd = firstNonEmpty(cfg.Passwd, mc.Passwd)
		cfg.Addr = firstNonEmpty(cfg.Addr, mc.Addr)
		cfg.DbName = firstNonEmpty(cfg.DbName, mc.DbName)
	}
	db, err := sqlx.Open("mysql", mysqlDSN(cfg))
	if err != nil {
		slog.Error("mysql 连接failed", zap.Error(err))
		return su_errors.WrapRetryable(su_errors.CodeUnavailable, "mysql open failed", err)
	}
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		slog.Error("mysql Ping failed", zap.Error(err))
		return su_errors.WrapRetryable(su_errors.CodeUnavailable, "mysql ping failed", err)
	}
	mc.mu.Lock()
	oldDB := mc.Db
	mc.Db = db
	mc.cfg = cfg
	mc.Uname = cfg.Uname
	mc.Passwd = cfg.Passwd
	mc.Addr = cfg.Addr
	mc.DbName = cfg.DbName
	mc.MaxOpenCns = cfg.MaxOpenConns
	mc.MaxIdleCns = cfg.MaxIdleConns
	mc.closeOnce = sync.Once{}
	mc.closeErr = nil
	mc.mu.Unlock()
	if oldDB != nil {
		_ = oldDB.Close()
	}
	return nil
}

func (mc *MysqlClient) dbLocked() (*sqlx.DB, error) {
	if mc == nil || mc.Db == nil {
		return nil, su_errors.New(su_errors.CodeUnavailable, "mysql client is not connected")
	}
	return mc.Db, nil
}

func (mc *MysqlClient) Close() error {
	if mc == nil {
		return nil
	}
	mc.closeOnce.Do(func() {
		mc.mu.Lock()
		if mc.Db == nil {
			mc.mu.Unlock()
			return
		}
		db := mc.Db
		mc.Db = nil
		mc.mu.Unlock()
		mc.closeErr = db.Close()
		slog.Info("mysql Close ", zap.Error(mc.closeErr))
	})
	return mc.closeErr
}

func (mc *MysqlClient) Insert(a_cmd string, a_parm ...interface{}) error {
	return mc.InsertContext(context.Background(), a_cmd, a_parm...)
}

func (mc *MysqlClient) InsertContext(ctx context.Context, a_cmd string, a_parm ...interface{}) error {
	r, err := mc.ExecContext(ctx, a_cmd, a_parm...)
	if err != nil {
		return err
	}
	id, err := r.LastInsertId()
	if err != nil {
		slog.Error("mysql insert result failed", zap.Error(err))
		return err
	}
	slog.Info("success ", zap.Any("id", id))
	return nil
}

func (mc *MysqlClient) Update(a_cmd string, a_parm ...interface{}) error {
	return mc.UpdateContext(context.Background(), a_cmd, a_parm...)
}

func (mc *MysqlClient) UpdateContext(ctx context.Context, a_cmd string, a_parm ...interface{}) error {
	r, err := mc.ExecContext(ctx, a_cmd, a_parm...)
	if err != nil {
		return err
	}
	row, err := r.RowsAffected()
	if err != nil {
		slog.Error("mysql update result failed", zap.Error(err))
		return err
	}
	slog.Info("success ", zap.Any("row", row))
	return nil
}

func (mc *MysqlClient) Delete(a_cmd string, a_parm ...interface{}) error {
	return mc.DeleteContext(context.Background(), a_cmd, a_parm...)
}

func (mc *MysqlClient) DeleteContext(ctx context.Context, a_cmd string, a_parm ...interface{}) error {
	r, err := mc.ExecContext(ctx, a_cmd, a_parm...)
	if err != nil {
		return err
	}
	row, err := r.RowsAffected()
	if err != nil {
		slog.Error("mysql delete result failed", zap.Error(err))
		return err
	}
	slog.Info("success ", zap.Any("row", row))
	return nil
}

func (mc *MysqlClient) Select(a_dest interface{}, a_cmd string, a_parm ...interface{}) error {
	return mc.SelectContext(context.Background(), a_dest, a_cmd, a_parm...)
}

func (mc *MysqlClient) ExecContext(ctx context.Context, a_cmd string, a_parm ...interface{}) (result sql.Result, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if mc == nil {
		return nil, su_errors.New(su_errors.CodeUnavailable, "mysql client is not connected")
	}
	db, err := mc.db()
	if err != nil {
		return nil, err
	}
	result, err = db.ExecContext(ctx, a_cmd, a_parm...)
	if err != nil {
		slog.Error("mysql exec failed", zap.Error(err))
		return nil, su_errors.WrapRetryable(su_errors.CodeUnavailable, "mysql exec failed", err)
	}
	return result, nil
}

func (mc *MysqlClient) SelectContext(ctx context.Context, a_dest interface{}, a_cmd string, a_parm ...interface{}) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if mc == nil {
		return su_errors.New(su_errors.CodeUnavailable, "mysql client is not connected")
	}
	db, err := mc.db()
	if err != nil {
		return err
	}
	err = db.SelectContext(ctx, a_dest, a_cmd, a_parm...)
	if err != nil {
		slog.Error("mysql query failed", zap.Error(err))
		return su_errors.WrapRetryable(su_errors.CodeUnavailable, "mysql query failed", err)
	}
	return nil
}

func (mc *MysqlClient) db() (*sqlx.DB, error) {
	if mc == nil {
		return nil, su_errors.New(su_errors.CodeUnavailable, "mysql client is not connected")
	}
	mc.mu.RLock()
	db, err := mc.dbLocked()
	mc.mu.RUnlock()
	return db, err
}

func defaultMysqlConfig(cfg MysqlConfig) MysqlConfig {
	if cfg.MaxOpenConns <= 0 {
		cfg.MaxOpenConns = 10
	}
	if cfg.MaxIdleConns <= 0 || cfg.MaxIdleConns > cfg.MaxOpenConns {
		cfg.MaxIdleConns = cfg.MaxOpenConns
		if cfg.MaxIdleConns > 5 {
			cfg.MaxIdleConns = 5
		}
	}
	if cfg.ConnMaxLifetime <= 0 {
		cfg.ConnMaxLifetime = time.Hour
	}
	if cfg.ConnMaxIdleTime <= 0 {
		cfg.ConnMaxIdleTime = 10 * time.Minute
	}
	return cfg
}

func mysqlDSN(cfg MysqlConfig) string {
	if cfg.DSN != "" {
		return cfg.DSN
	}
	return cfg.Uname + ":" + cfg.Passwd + "@tcp(" + cfg.Addr + ")/" + cfg.DbName
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
