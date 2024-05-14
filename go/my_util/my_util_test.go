package my_util_test

import (
	"go/my_util"
	"fmt"
	"testing"
)

func TestGetLogFileLine(t *testing.T){
	line := my_util.GetLogFileLine()
	fmt.Println(line)
}

func TestGetZeroTime(t *testing.T){
	fmt.Println("@@@@@", my_util.GetTodayZeroTime())
}

func TestToString(t *testing.T) {
	// v := 20102.9390394
	var v1 uint32 = 343
	fmt.Println(my_util.ToString(v1))
}