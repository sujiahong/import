package main

import (
	"fmt"
	"go/my_util"
	slog "go/su_log"
	"os"
	"runtime"
	"time"
	//"strings"
	"context"
	"sync"
	"strconv"
	"hash/crc32"
	"math"
	// "go/my_util/go_pool"

	// "github.com/panjf2000/gnet"
	// "github.com/panjf2000/gnet/pkg/pool/goroutine"
	"go.uber.org/zap"
	//"go/su_net"
	"github.com/golang/protobuf/proto"
	"go/proto/Test"
	sredis "go/su_da/redis"
	"github.com/garyburd/redigo/redis"
	smysql "go/su_da/su_sql"
	skafka "go/su_da/kafka"
)

/*
#include <stdio.h>

void hi()
{
	printf("hello world!\n");
}
*/
import "C"

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
	height int    
}

type PINFO struct {
	Id int  `db:"id"`
	Name string   `db:"name"`
}

var wg sync.WaitGroup

func worker(ctx context.Context){
	go worker2(ctx)
LOOP:
	for {
		fmt.Println("worker")
		time.Sleep(time.Second)
		select {
		case <- ctx.Done():
			break LOOP
		default:
		}
	}
	wg.Done()
}

func worker2(ctx context.Context){
LOOP:
	for {
		fmt.Println("worker2")
		time.Sleep(time.Second)
		select {
		case <- ctx.Done():
			break LOOP
		default:
		}
	}
}

func worker3(ctx context.Context){
LOOP:
	for {
		fmt.Println("db connecting ...")
		time.Sleep(time.Millisecond*10)
		select {
		case <- ctx.Done():
			break LOOP
		default:
		}
	}
	fmt.Println("workder3 done")
	wg.Done()
}

func GetTodayZeroTime() int64 {
	now := time.Now()
	zero_time := time.Date(now.Year(), now.Month(), now.Day(),0,0,0,0,now.Location())
	return zero_time.Unix()
}

func GetCrc32Sum(userId, peerId uint64, zero_time uint32) uint32 {
	str := strconv.FormatUint(userId, 10) + strconv.FormatUint(peerId, 10) + strconv.FormatUint(uint64(zero_time), 10)
	return crc32.ChecksumIEEE([]byte(str))
}
func GetMiniNeedPower(totalForce uint64) (uint64, uint64) {
	if totalForce == 0 {
		return 0, 0
	}
	if totalForce > 0 && totalForce < 10 {
		return 0, 10
	}
	str := strconv.FormatUint(totalForce, 10)
	n, _ := strconv.ParseUint(str[0:1], 10, 64) 
	m := uint64(math.Pow(10, float64(len(str)-1)))
	fmt.Println(str, m, n, len(str))
	ret := n * m
	return ret, ret + m
}
func main() {
	slog.Init("client.log")
	tn := time.Now()
	runtime.GOMAXPROCS(5)
	p, _ := os.Getwd()
	tm_str := tn.String()
	C.hi();
	fmt.Println("111111111 ", p, tm_str, tm_str[0:27], my_util.GetTimePrintString())
	fmt.Println(GetMiniNeedPower(1066))
	var li = my_util.GetLogFileLine()
	fmt.Println(li, math.Ceil(0.03))
	my_util.Classifier(li)
	var itemCount float64 = 51240
	var at_rate float64 = 2500.0
	var MAX_PRO float64 = 10000.0
	fmt.Println("22222222   ", itemCount * (at_rate + MAX_PRO) / MAX_PRO * float64(15000) / MAX_PRO)
	// go func(s string) {
	// 	for i := 0; i < 2; i++ {
	// 		fmt.Println(s)
	// 	}
	// }("world")

	for i := 0; i < 2; i++ {
		runtime.Gosched()
		fmt.Println("hello")
	}

	fmt.Println(my_util.GetTodayDate())
	fmt.Println("1231  ", my_util.GetZeroTime(0), )
	fmt.Println(GetCrc32Sum(1, 2, 3))
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
	
	// nano := uint64(tn.UnixNano())
	// my_util.DelayRun(1000, func(){
	// 	zero := 0
	// 	x := nano / uint64(zero)
	// 	slog.Info("delay run", zap.Any("nano", nano), zap.Any("x", x))
	// })
	st := &student{
		name: "pprof.cn",
		age: 18,
		height: 100,
	}
	fmt.Println("student=", st)
	slog.Info("student ", zap.Any("st", st))
	my_util.IntervalRun(100, 0, func(){
		slog.Info("interval run", zap.Any("q", 3))
	})
	time.Sleep(time.Second*400)

	// my_util.CopyFile("./t.log", "1.log")

	// gp := my_util.NewGoPool(3, 3)
	// for i := 0; i < 10; i++ {
	// 	nano1 := time.Now().UnixNano()
	// 	gp.SendTask(uint64(nano1), func(){
	// 		slog.Info("执行", zap.Any("nano1=",nano1))
	// 	})
	// 	//time.Sleep(time.Second)
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
	// gts.RegisterHandler(10000, &Test.TestRQ{}, 10001, &Test.TestRS{}, func(gnc *su_net.GNetConn,a_shardingid uint64, a_rq proto.Message, a_rs proto.Message){
	// 	rq := a_rq.(*Test.TestRQ)
	// 	rs := a_rs.(*Test.TestRS)
	// 	slog.Info(" recv ", zap.Any("rq", rq))
	// 	rs.Test1 = rq.Test1
	// 	rs.Test2 = rq.Test2
	// 	slog.Info(" finish ", zap.Any("rs", rs))
	// })
	// gts.Run()
	
	// gtc := su_net.CreateClient("127.0.0.1:9990",2)
	// gtc.RegisterHandler(10000, &Test.TestRQ{}, 10001, &Test.TestRS{}, func(gnc *su_net.GNetConn,a_shardingid uint64, a_rq proto.Message, a_rs proto.Message){
	// 	rq := a_rq.(*Test.TestRQ)
	// 	rs := a_rs.(*Test.TestRS)
	// 	slog.Info(" client recv ", zap.Any("rq", rq))
	// 	rs.Test1 = rq.Test1
	// 	rs.Test2 = rq.Test2
	// 	slog.Info(" client finish ", zap.Any("rs", rs))
	// })
	rq := &Test.TestRQ{}
	rq.Test1 = proto.Uint32(12367864)
	rq.Test2 = proto.String("测试 一下福建省佛教螺蛳粉放松拼接翻领萨法贾发泡剂阿里发放弗拉索夫骄傲卷发福建省佛教啊加热看见啊发饿就饿了就让了；发来送积分啦叠加多怕卷发；‘发发；封疆大吏放假诶下")
	// //time.Sleep()
	// var i uint32 = 0
	// for i = 0; i < 10000; i++ {
	// 	rq.Test1 = proto.Uint32(rq.GetTest1() + i)
	// 	gtc.Send(10000, 10001, rq)
	// }
	// time.Sleep(time.Second*3600)

	// ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*50)
	// wg.Add(1)
	// go worker3(ctx)
	// time.Sleep(time.Second*5)
	// cancel()
	// wg.Wait()
	// fmt.Println("over")
	
	// data_slice := make([]byte, 0, 0)
	// str := "88888888"
	// slice := []byte(str)
	// fmt.Println(data_slice, str, slice)	
	// for i := 0; i < 100; i++ {
	// 	data_slice = append(data_slice, slice...)
	// 	data_slice = data_slice[7:]
	// }
	// fmt.Println(len(data_slice), cap(data_slice), data_slice)

	slog.Info("redis 相关测试")
	sd := sredis.NewRedisClient("localhost:8379", 1)
	sd.Connect()
	_, err := sd.Do("set", "1", 234231)
	slog.Info("redis  set", zap.Error(err))
	r, err := redis.Int(sd.Do("get", "1"))
	slog.Info("redis  get", zap.Any("r",r), zap.Error(err))
	slog.Info("mysql 相关测试")
	sq := smysql.NewMysqlClient("root", "root", "localhost:6306", "tt1", 3, 1)
	sq.Connect()
	//sq.Insert("insert into t1(id,name) values (?,?)",100,"uosufje")
	sq.Update("update t1 set name=? where id=?", "dhsbd", 100)
	var info []PINFO
	sq.Select(&info, "select id, name from t1 where id=?", 100)
	slog.Info("mysql  select ", zap.Any("info",info))

	slog.Info("kafka 相关测试")
	kp := skafka.NewKafkaProducer([]string{"127.0.0.1:7000"}, "test", true)
	kp.Send("1234234233")
	kc := skafka.NewKafkaConsumer([]string{"127.0.0.1:7000"}, "test", "test1", func(a_pa_id int32){
		slog.Info("kafka 消费", zap.Int32("partion_id", a_pa_id))
	})
	time.Sleep(time.Second)
	kc.ConsumeAllPartion()
	time.Sleep(time.Second*10)
	// var testMap sync.Map
	// testMap.Store(34, "3943")
	// testMap.Store(3, "34")
	// testMap.Store(34, "87889")
	// val, _ := testMap.Load(34)
	// slog.Info("打印", zap.Any("val", val))
}
