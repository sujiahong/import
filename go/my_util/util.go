package my_util

import (
	"errors"
	"fmt"
	slog "go.local/su_log"
	"go.uber.org/zap"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

/*
channel的注意事项
channel 在 Golang 中是一等公民，它是线程安全的，面对并发问题，应首先想到 channel
关闭一个未初始化的 channel 会产生 panic
重复关闭同一个 channel 会产生 panic
向一个已关闭的 channel 发送消息会产生 panic
从已关闭的 channel 读取消息不会产生 panic，且能读出 channel 中还未被读取的消息，若消息均已被读取，则会读取到该类型的零值。
从已关闭的 channel 读取消息永远不会阻塞，并且会返回一个为 false 的值，用以判断该 channel 是否已关闭（x,ok := <- ch）
关闭 channel 会产生一个广播机制，所有向 channel 读取消息的 goroutine 都会收到消息
*/

func GetLogFileLine() string {
	pc, name, line, ok := runtime.Caller(1)
	if !ok {
		return ""
	}
	fun := runtime.FuncForPC(pc)
	return fmt.Sprintf("%s %d %s", filepath.Base(name), line, filepath.Base(fun.Name()))
}

func Classifier(items ...interface{}) { /////类型分类函数
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
	zero_time := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return zero_time.Unix()
}
func GetZeroTime(timestamp uint32) int64 {
	var t time.Time
	if timestamp == 0 {
		t = time.Now()
	} else {
		t = time.Unix(int64(timestamp), 0)
	}
	zero_time := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	return zero_time.Unix()
}

func GetTodayDate() uint32 {
	now := time.Now()
	year := now.Year()
	month := now.Month()
	day := now.Day()
	date := year*10000 + int(month)*100 + day
	return uint32(date)
}

func GetTimePrintString() string {
	tn := time.Now()
	tm_str := tn.String()
	return tm_str[0:27]
}

func RecoverPanic() {
	if err := recover(); err != nil {
		timeStr := GetTimePrintString()
		fmt.Println(timeStr+" panic: recovered err=", err, string(debug.Stack()))
		slog.Error(timeStr+" panic: recovered err", zap.Any("err:", err), zap.Any("stack:", string(debug.Stack())))
	}
}

// //毫秒延时执行
func DelayRun(a_dealy uint32, a_f func()) {
	go func() {
		defer RecoverPanic()
		timer := time.NewTimer(time.Duration(a_dealy) * time.Millisecond)
		defer timer.Stop()
		select {
		case <-timer.C:
			a_f()
			return
		}
	}()
}

// 定时执行毫秒，限定执行次数
func IntervalRun(a_interval, a_times uint32, a_f func()) {
	var count uint32 = 0
	go func() {
		defer RecoverPanic()
		ticker := time.NewTicker(time.Duration(a_interval) * time.Millisecond)
		defer ticker.Stop()
		for {
			if a_times > 0 {
				if count >= a_times {
					return
				}
				count++
			}
			select {
			case <-ticker.C:
				a_f()
			}
		}
	}()
}

// 文件拷贝，从a拷到b
func CopyFile(a_src_file, a_dst_file string) {
	if err := CopyFileE(a_src_file, a_dst_file); err != nil {
		slog.Error("CopyFile err=", zap.Error(err))
	}
}

func CopyFileE(aSrcFile, aDstFile string) error {
	srcFile, err := os.Open(aSrcFile)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	dstFile, err := os.OpenFile(aDstFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer dstFile.Close()
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}
	return dstFile.Sync()
}

// //判断管道是否关闭
func IsChanClosed(ch chan int) bool {
	select {
	case _, received := <-ch:
		return !received
	default:
	}
	return false
}

func WaitGroupWithTimeout(wg *sync.WaitGroup, timeout uint32) error {
	if wg == nil {
		return nil
	}
	c := make(chan struct{})
	go func() {
		defer RecoverPanic()
		wg.Wait()
		close(c)
	}()
	select {
	case <-c:
		// fmt.Println("wait group finish!!!")
		return nil
	case <-time.After(time.Duration(timeout) * time.Millisecond):
		fmt.Println("warn: timeout waiting for wait group!!!")
		return errors.New("timeout waiting for wait group")
	}
}

func SyncMustSuccessIO(f func() error) {
	for {
		err := f()
		if err != nil {
			fmt.Println("SyncMustSuccessIO err=", zap.Error(err))
			time.Sleep(time.Second * 5)
		} else {
			return /////结束
		}
	}
}

func AsyncMustSuccessIO(f func() error) {
	go func() {
		defer RecoverPanic()
		for {
			err := f()
			if err != nil {
				fmt.Println("AsyncMustSuccessIO err=", zap.Error(err))
				time.Sleep(time.Second * time.Duration(RandRange(10, 20)))
			} else {
				return /////结束协程
			}
		}
	}()
}

var incrUniqId uint32 ///////自增唯一ID   此进程内唯一
func GetIncrUUID() uint64 { ///获取累加进程内唯一ID
	next := atomic.AddUint32(&incrUniqId, 1)
	if next == 0 {
		next = atomic.AddUint32(&incrUniqId, 1)
	}
	return uint64(next)
}
