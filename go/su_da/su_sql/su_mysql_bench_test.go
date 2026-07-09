package su_mysql

import (
	"context"
	"sync/atomic"
	"testing"
)

func BenchmarkMysqlExecContext(b *testing.B) {
	mc := newBenchMysqlClient()
	defer mc.Close()
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := mc.ExecContext(ctx, "update t set a=?", 1); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMysqlSelectContext(b *testing.B) {
	mc := newBenchMysqlClient()
	defer mc.Close()
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var rows []struct{ A int }
		if err := mc.SelectContext(ctx, &rows, "select a from t"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMysqlExecContextParallel(b *testing.B) {
	mc := newBenchMysqlClient()
	defer mc.Close()
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, err := mc.ExecContext(ctx, "update t set a=?", 1); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func newBenchMysqlClient() *MysqlClient {
	var closeCount atomic.Int32
	db := newFakeMysqlDB(&closeCount)
	db.SetMaxOpenConns(64)
	db.SetMaxIdleConns(64)
	return &MysqlClient{Db: db}
}
