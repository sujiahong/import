package su_mysql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"go.local/su_errors"
)

func TestMysqlOperationsBeforeConnectReturnError(t *testing.T) {
	mc := NewMysqlClient("root", "root", "127.0.0.1:3306", "test", 1, 1)

	if err := mc.Insert("insert into t(a) values(?)", 1); err == nil {
		t.Fatal("expected insert error before mysql connect")
	} else if su_errors.CodeOf(err) != su_errors.CodeUnavailable {
		t.Fatalf("error code = %d, want unavailable", su_errors.CodeOf(err))
	}
	if err := mc.Update("update t set a=?", 1); err == nil {
		t.Fatal("expected update error before mysql connect")
	}
	if err := mc.Delete("delete from t where a=?", 1); err == nil {
		t.Fatal("expected delete error before mysql connect")
	}
	var rows []struct{ A int }
	if err := mc.Select(&rows, "select a from t"); err == nil {
		t.Fatal("expected select error before mysql connect")
	}
}

func TestMysqlCloseIsNilSafe(t *testing.T) {
	var mc *MysqlClient
	if err := mc.Close(); err != nil {
		t.Fatalf("nil close failed: %v", err)
	}

	mc = NewMysqlClient("root", "root", "127.0.0.1:3306", "test", 1, 1)
	if err := mc.Close(); err != nil {
		t.Fatalf("unconnected close failed: %v", err)
	}
}

func TestNewMysqlClientIncompleteConfigDoesNotReturnNil(t *testing.T) {
	mc := NewMysqlClient("", "", "", "", 1, 1)
	if mc == nil {
		t.Fatal("NewMysqlClient returned nil")
	}
	if err := mc.Connect(); err == nil {
		t.Fatal("expected connect error for incomplete mysql config")
	} else if su_errors.CodeOf(err) != su_errors.CodeInvalidArgument {
		t.Fatalf("error code = %d, want invalid argument", su_errors.CodeOf(err))
	}
}

func TestMysqlCloseOnceAndReconnectReset(t *testing.T) {
	var closeCount atomic.Int32
	mc := &MysqlClient{Db: newFakeMysqlDB(&closeCount)}
	if err := mc.Close(); err != nil {
		t.Fatalf("first close failed: %v", err)
	}
	if err := mc.Close(); err != nil {
		t.Fatalf("second close failed: %v", err)
	}
	if got := closeCount.Load(); got != 1 {
		t.Fatalf("closeCount = %d, want 1", got)
	}

	mc.mu.Lock()
	mc.Db = newFakeMysqlDB(&closeCount)
	mc.closeOnce = sync.Once{}
	mc.closeErr = nil
	mc.mu.Unlock()
	if err := mc.Close(); err != nil {
		t.Fatalf("close after reconnect failed: %v", err)
	}
	if got := closeCount.Load(); got != 2 {
		t.Fatalf("closeCount after reconnect = %d, want 2", got)
	}
}

func TestMysqlCloseWaitsForReconnectLock(t *testing.T) {
	var closeCount atomic.Int32
	mc := &MysqlClient{Db: newFakeMysqlDB(&closeCount)}
	mc.reconnectMu.Lock()
	closeDone := make(chan error, 1)
	go func() {
		closeDone <- mc.Close()
	}()
	select {
	case err := <-closeDone:
		t.Fatalf("Close completed while reconnectMu was held: %v", err)
	case <-time.After(10 * time.Millisecond):
	}
	mc.reconnectMu.Unlock()
	select {
	case err := <-closeDone:
		if err != nil {
			t.Fatalf("Close error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Close did not finish after reconnectMu was released")
	}
}

func TestMysqlConfigConstructorDefaults(t *testing.T) {
	mc, err := NewMysqlClientWithConfig(MysqlConfig{
		Uname:  "root",
		Addr:   "127.0.0.1:3306",
		DbName: "test",
	})
	if err != nil {
		t.Fatalf("config constructor failed: %v", err)
	}
	if mc.MaxOpenCns != 10 {
		t.Fatalf("MaxOpenCns = %d, want 10", mc.MaxOpenCns)
	}
	if mc.MaxIdleCns != 5 {
		t.Fatalf("MaxIdleCns = %d, want 5", mc.MaxIdleCns)
	}
	if mc.cfg.ConnMaxLifetime != time.Hour {
		t.Fatalf("ConnMaxLifetime = %v, want %v", mc.cfg.ConnMaxLifetime, time.Hour)
	}
	if mc.cfg.ConnectTimeout != 5*time.Second {
		t.Fatalf("ConnectTimeout = %v, want 5s", mc.cfg.ConnectTimeout)
	}
	if mc.cfg.ReadTimeout != 5*time.Second {
		t.Fatalf("ReadTimeout = %v, want 5s", mc.cfg.ReadTimeout)
	}
	if mc.cfg.WriteTimeout != 5*time.Second {
		t.Fatalf("WriteTimeout = %v, want 5s", mc.cfg.WriteTimeout)
	}
	if mc.cfg.PingTimeout != 5*time.Second {
		t.Fatalf("PingTimeout = %v, want 5s", mc.cfg.PingTimeout)
	}
}

func TestMysqlDSNIncludesTimeoutsForSplitConfig(t *testing.T) {
	dsn, err := mysqlDSN(defaultMysqlConfig(MysqlConfig{
		Uname:  "root",
		Passwd: "secret",
		Addr:   "127.0.0.1:3306",
		DbName: "test",
	}))
	if err != nil {
		t.Fatalf("mysqlDSN() error = %v", err)
	}
	cfg, err := mysqlDriver.ParseDSN(dsn)
	if err != nil {
		t.Fatalf("ParseDSN() error = %v", err)
	}
	if cfg.Timeout != 5*time.Second {
		t.Fatalf("Timeout = %v, want 5s", cfg.Timeout)
	}
	if cfg.ReadTimeout != 5*time.Second {
		t.Fatalf("ReadTimeout = %v, want 5s", cfg.ReadTimeout)
	}
	if cfg.WriteTimeout != 5*time.Second {
		t.Fatalf("WriteTimeout = %v, want 5s", cfg.WriteTimeout)
	}
}

func TestMysqlDSNAppliesExplicitTimeoutsToProvidedDSN(t *testing.T) {
	dsn, err := mysqlDSN(MysqlConfig{
		DSN:            "u:p@tcp(127.0.0.1:3306)/db",
		ConnectTimeout: 2 * time.Second,
		ReadTimeout:    3 * time.Second,
		WriteTimeout:   4 * time.Second,
	})
	if err != nil {
		t.Fatalf("mysqlDSN() error = %v", err)
	}
	cfg, err := mysqlDriver.ParseDSN(dsn)
	if err != nil {
		t.Fatalf("ParseDSN() error = %v", err)
	}
	if cfg.Timeout != 2*time.Second {
		t.Fatalf("Timeout = %v, want 2s", cfg.Timeout)
	}
	if cfg.ReadTimeout != 3*time.Second {
		t.Fatalf("ReadTimeout = %v, want 3s", cfg.ReadTimeout)
	}
	if cfg.WriteTimeout != 4*time.Second {
		t.Fatalf("WriteTimeout = %v, want 4s", cfg.WriteTimeout)
	}
}

func newFakeMysqlDB(closeCount *atomic.Int32) *sqlx.DB {
	db := sql.OpenDB(fakeMysqlConnector{closeCount: closeCount})
	if err := db.Ping(); err != nil {
		panic(err)
	}
	return sqlx.NewDb(db, "fake-mysql")
}

type fakeMysqlConnector struct {
	closeCount *atomic.Int32
}

func (c fakeMysqlConnector) Connect(ctx context.Context) (driver.Conn, error) {
	return fakeMysqlConn{closeCount: c.closeCount}, nil
}

func (c fakeMysqlConnector) Driver() driver.Driver {
	return fakeMysqlDriver{}
}

type fakeMysqlDriver struct{}

func (fakeMysqlDriver) Open(name string) (driver.Conn, error) {
	return fakeMysqlConn{}, nil
}

type fakeMysqlConn struct {
	closeCount *atomic.Int32
}

func (c fakeMysqlConn) Prepare(query string) (driver.Stmt, error) {
	return nil, driver.ErrSkip
}

func (c fakeMysqlConn) Close() error {
	if c.closeCount != nil {
		c.closeCount.Add(1)
	}
	return nil
}

func (c fakeMysqlConn) Begin() (driver.Tx, error) {
	return nil, driver.ErrSkip
}

func (c fakeMysqlConn) Ping(ctx context.Context) error {
	return nil
}

func (c fakeMysqlConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	return fakeMysqlResult{}, nil
}

func (c fakeMysqlConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	return fakeMysqlRows{}, nil
}

type fakeMysqlResult struct{}

func (fakeMysqlResult) LastInsertId() (int64, error) { return 1, nil }

func (fakeMysqlResult) RowsAffected() (int64, error) { return 1, nil }

type fakeMysqlRows struct{}

func (fakeMysqlRows) Columns() []string { return []string{"a"} }

func (fakeMysqlRows) Close() error { return nil }

func (fakeMysqlRows) Next(dest []driver.Value) error { return io.EOF }
