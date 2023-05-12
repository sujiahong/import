package main

import (
	"fmt"
	"go/my_util"
	"os"
	"runtime"
	"time"
)

func a() {
	for i := 1; i < 10; i++ {
		fmt.Println("A:", i)
	}
}

func b() {
	for i := 1; i < 10; i++ {
		fmt.Println("B:", i)
	}
}

type student struct {
	name string
	age int
}

func main() {
	runtime.GOMAXPROCS(3)
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

	
	go a()
	go b()
	time.Sleep(time.Second)

	m := make(map[string]*student)
	stus := []student{
		{name: "pprof.cn", age: 18},
		{name: "测试", age: 23},
		{name: "博客", age: 34},
	}

	for _, stu := range stus {
		fmt.Printf(" %p\n", &stu)
		m[stu.name] = &stu
	}
	for k, v := range m {
		fmt.Println(k, "===>", v.name)
	}
}
