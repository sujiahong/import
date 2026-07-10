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

	mysql "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"go.local/su_errors"
	slog "go.local/su_log"
	"go.uber.org/zap"
	//"time"
)

// MysqlConfig 定义 MySQL DSN、连接池容量和连接/读写/Ping 超时配置。
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
	ConnectTimeout  time.Duration
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	PingTimeout     time.Duration
}

// MysqlClient 封装 sqlx.DB，并管理连接池重建、关闭和基础 CRUD 操作。
type MysqlClient struct {
	Db          *sqlx.DB
	Uname       string
	Passwd      string
	Addr        string
	DbName      string
	MaxOpenCns  int
	MaxIdleCns  int
	cfg         MysqlConfig
	mu          sync.RWMutex
	reconnectMu sync.Mutex
	closeOnce   sync.Once
	closeErr    error
}

// NewMysqlClient 使用传统账号参数创建 MySQL client。
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

// NewMysqlClientWithConfig 使用完整配置创建 MySQL client。
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

// Connect 创建并 Ping 新的 sqlx.DB，成功后替换当前连接池。
func (mc *MysqlClient) Connect() error {
	if mc == nil {
		return su_errors.New(su_errors.CodeInvalidArgument, "mysql client is nil")
	}
	mc.reconnectMu.Lock()
	defer mc.reconnectMu.Unlock()
	cfg := defaultMysqlConfig(mc.cfg)
	if cfg.DSN == "" {
		cfg.Uname = firstNonEmpty(cfg.Uname, mc.Uname)
		cfg.Passwd = firstNonEmpty(cfg.Passwd, mc.Passwd)
		cfg.Addr = firstNonEmpty(cfg.Addr, mc.Addr)
		cfg.DbName = firstNonEmpty(cfg.DbName, mc.DbName)
	}
	if cfg.DSN == "" && (cfg.Uname == "" || cfg.Addr == "" || cfg.DbName == "") {
		return su_errors.New(su_errors.CodeInvalidArgument, "mysql config is incomplete")
	}
	dsn, err := mysqlDSN(cfg)
	if err != nil {
		slog.Error("mysql DSN invalid", zap.Error(err))
		return su_errors.Wrap(su_errors.CodeInvalidArgument, "mysql dsn invalid", err)
	}
	db, err := sqlx.Open("mysql", dsn)
	if err != nil {
		slog.Error("mysql 连接failed", zap.Error(err))
		return su_errors.WrapRetryable(su_errors.CodeUnavailable, "mysql open failed", err)
	}
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	pingCtx := context.Background()
	var cancel context.CancelFunc
	if cfg.PingTimeout > 0 {
		pingCtx, cancel = context.WithTimeout(context.Background(), cfg.PingTimeout)
		defer cancel()
	}
	if err := db.PingContext(pingCtx); err != nil {
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

// Reconnect 显式重建 MySQL 连接池。
func (mc *MysqlClient) Reconnect() error {
	return mc.Connect()
}

// dbLocked 在调用方已持有锁时返回当前数据库连接池。
func (mc *MysqlClient) dbLocked() (*sqlx.DB, error) {
	if mc == nil || mc.Db == nil {
		return nil, su_errors.New(su_errors.CodeUnavailable, "mysql client is not connected")
	}
	return mc.Db, nil
}

// Close 关闭当前 MySQL 连接池，并与 Connect/Reconnect 互斥。
func (mc *MysqlClient) Close() error {
	if mc == nil {
		return nil
	}
	mc.reconnectMu.Lock()
	defer mc.reconnectMu.Unlock()
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

// Insert 执行插入 SQL，并记录 LastInsertId。
func (mc *MysqlClient) Insert(a_cmd string, a_parm ...interface{}) error {
	return mc.InsertContext(context.Background(), a_cmd, a_parm...)
}

// InsertContext 使用指定 context 执行插入 SQL。
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

// Update 执行更新 SQL，并记录受影响行数。
func (mc *MysqlClient) Update(a_cmd string, a_parm ...interface{}) error {
	return mc.UpdateContext(context.Background(), a_cmd, a_parm...)
}

// UpdateContext 使用指定 context 执行更新 SQL。
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

// Delete 执行删除 SQL，并记录受影响行数。
func (mc *MysqlClient) Delete(a_cmd string, a_parm ...interface{}) error {
	return mc.DeleteContext(context.Background(), a_cmd, a_parm...)
}

// DeleteContext 使用指定 context 执行删除 SQL。
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

// Select 执行查询 SQL 并将结果写入目标对象。
func (mc *MysqlClient) Select(a_dest interface{}, a_cmd string, a_parm ...interface{}) error {
	return mc.SelectContext(context.Background(), a_dest, a_cmd, a_parm...)
}

// ExecContext 执行写类 SQL，运行时错误会包装为 retryable。
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

// SelectContext 使用指定 context 执行查询 SQL，运行时错误会包装为 retryable。
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

// db 并发安全地返回当前 sqlx.DB。
func (mc *MysqlClient) db() (*sqlx.DB, error) {
	if mc == nil {
		return nil, su_errors.New(su_errors.CodeUnavailable, "mysql client is not connected")
	}
	mc.mu.RLock()
	db, err := mc.dbLocked()
	mc.mu.RUnlock()
	return db, err
}

// defaultMysqlConfig 填充 MySQL 连接池和超时默认值。
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
	if cfg.DSN == "" {
		if cfg.ConnectTimeout <= 0 {
			cfg.ConnectTimeout = 5 * time.Second
		}
		if cfg.ReadTimeout <= 0 {
			cfg.ReadTimeout = 5 * time.Second
		}
		if cfg.WriteTimeout <= 0 {
			cfg.WriteTimeout = 5 * time.Second
		}
	}
	if cfg.PingTimeout <= 0 {
		cfg.PingTimeout = 5 * time.Second
	}
	return cfg
}

// mysqlDSN 根据显式 DSN 或拆分字段生成 go-sql-driver/mysql 可用的 DSN。
func mysqlDSN(cfg MysqlConfig) (string, error) {
	if cfg.DSN != "" {
		if cfg.ConnectTimeout <= 0 && cfg.ReadTimeout <= 0 && cfg.WriteTimeout <= 0 {
			return cfg.DSN, nil
		}
		dsnCfg, err := mysql.ParseDSN(cfg.DSN)
		if err != nil {
			return "", err
		}
		applyMysqlTimeouts(dsnCfg, cfg)
		return dsnCfg.FormatDSN(), nil
	}
	dsnCfg := mysql.NewConfig()
	dsnCfg.User = cfg.Uname
	dsnCfg.Passwd = cfg.Passwd
	dsnCfg.Net = "tcp"
	dsnCfg.Addr = cfg.Addr
	dsnCfg.DBName = cfg.DbName
	applyMysqlTimeouts(dsnCfg, cfg)
	return dsnCfg.FormatDSN(), nil
}

// applyMysqlTimeouts 将连接、读、写超时写入 mysql.Config。
func applyMysqlTimeouts(dsnCfg *mysql.Config, cfg MysqlConfig) {
	if cfg.ConnectTimeout > 0 {
		dsnCfg.Timeout = cfg.ConnectTimeout
	}
	if cfg.ReadTimeout > 0 {
		dsnCfg.ReadTimeout = cfg.ReadTimeout
	}
	if cfg.WriteTimeout > 0 {
		dsnCfg.WriteTimeout = cfg.WriteTimeout
	}
}

// firstNonEmpty 返回参数列表中的第一个非空字符串。
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
