package su_mysql

import (
	"testing"
	"time"
)

func TestMysqlOperationsBeforeConnectReturnError(t *testing.T) {
	mc := NewMysqlClient("root", "root", "127.0.0.1:3306", "test", 1, 1)

	if err := mc.Insert("insert into t(a) values(?)", 1); err == nil {
		t.Fatal("expected insert error before mysql connect")
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
}
