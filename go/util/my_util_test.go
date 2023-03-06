package my_util_test

import (
	"my_util"
	"fmt"
)

func TestGetLogFileLine(){
	string line = my_util.GetLogFileLine()
	fmt.Println(line)
}