package su_util_test

import (
	"fmt"
	"go.local/su_util"
	"testing"
)

func TestGetLogFileLine(t *testing.T) {
	line := su_util.GetLogFileLine()
	fmt.Println(line)
}

func TestGetZeroTime(t *testing.T) {
	fmt.Println("@@@@@", su_util.GetTodayZeroTime())
}

func TestToString(t *testing.T) {
	// v := 20102.9390394
	var v1 uint32 = 343
	fmt.Println(su_util.ToString(v1))
}

func TestSyncMustSuccessIO(t *testing.T) {
	fmt.Println("111111111")
	su_util.SyncMustSuccessIO(func() error {
		return nil
	})
	fmt.Println("22222222")
}

func TestAsyncMustSuccessIO(t *testing.T) {
	fmt.Println("111111111")
	done := make(chan struct{})
	su_util.AsyncMustSuccessIO(func() error {
		close(done)
		return nil
	})
	<-done
	fmt.Println("22222222")
}
