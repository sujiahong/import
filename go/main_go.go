package main

import (
	"fmt"
	"os"
	"go/my_util"
)

func main(){
	p, _ := os.Getwd()
	fmt.Println("111111111 ", p)
	var li = my_util.GetLogFileLine()
	fmt.Println(li)
}