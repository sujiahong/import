package my_util

import (
	"runtime"
    "runtime/debug"
	"fmt"
    "time"
    "sync"
    "os"
    "io"
    "math/rand"
	"path/filepath"
    slog "go/su_log"
    "go.uber.org/zap"
    "errors"
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
        fmt.Println("panic: recovered err=", err, string(debug.Stack()))
        slog.Error("panic: recovered err", zap.Any("err:", err), zap.Any("stack:", string(debug.Stack())))
    }
}

// //毫秒延时执行
func DelayRun(a_dealy uint32, a_f func()) {
	go func() {
		defer RecoverPanic()
		select {
		case <-time.After(time.Duration(a_dealy) * time.Millisecond):
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
        for {
            if a_times > 0 {
                if count >= a_times {
                    return
                }
                count++
            }
		    select {
		    case <-time.Tick(time.Duration(a_interval) * time.Millisecond):
		    	a_f()
		    }
        }
    }()
}

//文件拷贝，从a拷到b
func CopyFile(a_src_file, a_dst_file string) {
    var err error
    var srcFileST *os.File
    srcFileST, err = os.Open(a_src_file)
    if err != nil {
        slog.Error("src os.Open err=", zap.Error(err))
        if !os.IsExist(err) {
            srcFileST, err = os.Create(a_src_file)
            if err != nil {
                slog.Error("src os.Create err=", zap.Error(err))
            }
        }
        return
    }
    var dstFileST *os.File
    dstFileST, err = os.OpenFile(a_dst_file, os.O_RDWR|os.O_CREATE, 0777)
    if err != nil {
        slog.Error("dst os.OpenFile err=", zap.Error(err))
        return
    }
    defer srcFileST.Close()
    defer dstFileST.Close()
    buf := make([]byte, 4098)
    for {
        n, err := srcFileST.Read(buf)
        //slog.Info("打印 ", zap.Any("n=",n), zap.Any("a_src_file",a_src_file))
        if err == io.EOF && n == 0{
            slog.Info("srcFileST.Read读取完毕", zap.Any("read n", n))
            break
        }
        if err != nil {
            slog.Error("srcFileST.Read err=", zap.Error(err))
            break
        }
        wn, err := dstFileST.Write(buf[:n])
        if err != nil {
            slog.Error("dstFileST.Write err=", zap.Error(err), zap.Any("wn=", wn))
            break
        }
    }
}

/////[min,max]范围内随机
func RandRange(min, max int64) int64 {
	if max < min  {
		return 0
	}
	return rand.Int63() % (max-min+1) + min
}

func WaitGroupWithTimeout(wg *sync.WaitGroup, timeout uint32) error{
	c := make(chan int)
	defer close(c)
	go func(){
		defer RecoverPanic()
		wg.Wait()
		c <- 0
	}()
	select {
	case <-c:
		fmt.Println("wait group finish!!!")
		return nil
	case <-time.After(time.Duration(timeout)*time.Millisecond):
		fmt.Println("warn: timeout waiting for wait group!!!")
		return errors.New("timeout waiting for wait group")
	}
}