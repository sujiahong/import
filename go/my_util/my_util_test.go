package my_util_test

import (
	"go/my_util"
	"fmt"
	"testing"
	"errors"
	"time"
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

func TestSyncMustSuccessIO(t *testing.T) {
	fmt.Println("111111111")
	my_util.SyncMustSuccessIO(func() error {
		return errors.New("test error")
	})
	fmt.Println("22222222")
}

func TestAsyncMustSuccessIO(t *testing.T) {
	fmt.Println("111111111")
	my_util.AsyncMustSuccessIO(func() error {
		return errors.New("test error")
	})
	time.Sleep(time.Second*50)
	fmt.Println("22222222")
}