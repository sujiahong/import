package my_util

import (
	"runtime"
	"fmt"
    "time"
	"path/filepath"
    slog "go/su_log"
    "go.uber.org/zap"
)

func GetLogFileLine() string {
	pc, name, line, ok := runtime.Caller(1)
	if !ok {
		return ""
	}
	fun := runtime.FuncForPC(pc)
	return fmt.Sprintf("%s %d %s", filepath.Base(name), line, filepath.Base(fun.Name()))
}

func Classifier(items ...interface{}) {/////类型分类函数
    for i, x := range items {
        switch x.(type) {
        case bool:
            slog.Info("Param is a bool", zap.Int("i=", i))
        case float64:
            slog.Info("Param is a float64", zap.Int("i=", i))
        case int, int64:
            slog.Info("Param is a int", zap.Int("i=", i))
        case nil:
            slog.Info("Param is a nil", zap.Int("i=", i))
        case string:
            slog.Info("Param is a string", zap.Int("i=", i))
        default:
            slog.Error("Param is unknown", zap.Int("i=", i))
        }
    }
}

func GetTodayZeroTime() int64 {
	now := time.Now()
	zero_time := time.Date(now.Year(), now.Month(), now.Day(),0,0,0,0,now.Location())
	return zero_time.Unix()
}