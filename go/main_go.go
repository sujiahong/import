package main

import (
	"fmt"
	"go/my_util"
	slog "go/su_log"
	"os"
	"runtime"
	"time"
	// "go/my_util/go_pool"

	// "github.com/panjf2000/gnet"
	// "github.com/panjf2000/gnet/pkg/pool/goroutine"
	"go.uber.org/zap"
	"go/su_net"
)

func a() {
	aa := "aaaa"
	for i := 1; i < 10; i++ {
		fmt.Println("A:", i)
	}
	go func(){
		for i := 1; i < 1000; i++ {
			fmt.Println("aa:", i, aa)
		}
	}()
	fmt.Println("end end end")
}

func b() {
	for i := 1; i < 100; i++ {
		fmt.Println("B:", i)
	}
}

type student struct {
	name string
	age int
}


func GetTodayZeroTime() int64 {
	now := time.Now()
	zero_time := time.Date(now.Year(), now.Month(), now.Day(),0,0,0,0,now.Location())
	return zero_time.Unix()
}
func main() {
	runtime.GOMAXPROCS(5)
	p, _ := os.Getwd()
	fmt.Println("111111111 ", p)
	var li = my_util.GetLogFileLine()
	fmt.Println(li)
	my_util.Classifier(li)

	go func(s string) {
		for i := 0; i < 2; i++ {
			fmt.Println(s)
		}
	}("world")

	for i := 0; i < 2; i++ {
		runtime.Gosched()
		fmt.Println("hello")
	}

	
	// go a()
	// time.Sleep(time.Second)

	// m := make(map[string]*student)
	// stus := []student{
	// 	{name: "pprof.cn", age: 18},
	// 	{name: "测试", age: 23},
	// 	{name: "博客", age: 34},
	// }

	// for _, stu := range stus {
	// 	fmt.Printf(" %p\n", &stu)
	// 	m[stu.name] = &stu
	// }
	// for k, v := range m {
	// 	fmt.Println(k, "===>", v.name)
	// }
	// for j:= 0; j < 10; j++ {
	// 	go func(t int){
	// 		for i := 0; i <1000; i++ {
	// 			slog.Info("test log", zap.Int("uid", 321323+i), zap.Int("j=", t))
	// 		}
	// 	}(j)
	// }
	
	// for i := 0; i <1000; i++ {
	// 	slog.Info("test log", zap.Int("uid", 321323+i))
	// }

	// p := goroutine.Default()
	// defer p.Rlease()

	// echo := &echoServer{pool: p}
	// gnet.Serve(echo, "tcp:://:9000", gnet.WithMulticore(true))
	//fmt.Println("@@@@@", GetTodayZeroTime())

	my_util.DelayRun(3000, func(){
		slog.Info("delay run")
	})
	my_util.IntervalRun(1000, 3, func(){
		slog.Info("interval run", zap.Any("q", 3))
	})
	time.Sleep(time.Second*5)

	// my_util.CopyFile("./t.log", "1.log")

	// gp := my_util.NewGoPool(3, 3)
	// for i := 0; i < 10; i++ {
	// 	nano1 := time.Now().UnixNano()
	// 	gp.SendTask(uint64(nano1), func(){
	// 		slog.Info("执行", zap.Any("nano1=",nano1))
	// 	})
	// 	time.Sleep(time.Second)
	// }
	// gp.Stop()
	// nano2 := time.Now().UnixNano()
	// gp.SendTask(uint64(nano2), func(){
	// 	slog.Info("执行", zap.Any("nano2=",nano2))
	// })
	// nano3 := time.Now().UnixNano()
	// gp.SendTask(uint64(nano3), func(){
	// 	slog.Info("执行", zap.Any("nano3=",nano3))
	// })
	// time.Sleep(time.Second*5)

	// gts := su_net.CreateServer("9990")
	// gts.RegisterHandler()
	gtc := su_net.CreateClient("127.0.0.1:9990",1)
}
