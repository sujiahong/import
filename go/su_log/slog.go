package su_log

import (
	"os"
	"path/filepath"
	"runtime"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.Logger
func Init(file_name string) {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoder := zapcore.NewJSONEncoder(encoderConfig)

	file, _ := os.OpenFile(file_name, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	fileWriteSyncer := zapcore.AddSync(file)

	core := zapcore.NewTee(
		zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zapcore.DebugLevel),
		zapcore.NewCore(encoder, fileWriteSyncer, zapcore.DebugLevel),
	)
	logger = zap.New(core)
}

func getCallerInfoForLog() (callerFields []zap.Field){
	pc, file, line, ok := runtime.Caller(2)
	if !ok {
		return
	}
	func_name := filepath.Base(runtime.FuncForPC(pc).Name())
	callerFields = append(callerFields, zap.String("func", func_name), zap.String("file", file), zap.Int("line", line))
	return
}

func Debug(msg string, fields ...zap.Field){
	callerFields := getCallerInfoForLog()
	fields = append(fields, callerFields...)
	logger.Debug(msg, fields...)
}
func Info(msg string, fields ...zap.Field){
	callerFields := getCallerInfoForLog()
	fields = append(fields, callerFields...)
	logger.Info(msg, fields...)
}

func Warn(msg string, fields ...zap.Field){
	callerFields := getCallerInfoForLog()
	fields = append(fields, callerFields...)
	logger.Warn(msg, fields...)
}

func Error(msg string, fields ...zap.Field){
	callerFields := getCallerInfoForLog()
	fields = append(fields, callerFields...)
	logger.Error(msg, fields...)
}