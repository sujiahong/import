package my_util

import (
	"strconv"
	"fmt"
)

func ToString(v interface{}) string {
	switch v.(type) {
	case string:
		return v.(string)
	case int:
		return strconv.Itoa(v.(int))
	case int64:
		return strconv.FormatInt(v.(int64), 10)
	case int32:
		return strconv.FormatInt(int64(v.(int32)), 10)
	case uint64:
		return strconv.FormatUint(v.(uint64), 10)
	case uint32:
		return strconv.FormatUint(uint64(v.(uint32)), 10)
	case float64:
		return strconv.FormatFloat(v.(float64), 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(v.(float32)), 'f', -1, 32)
	case bool:
		return strconv.FormatBool(v.(bool))
	default:
		fmt.Println("ToString err: 不处理的类型", "time: ", GetTimePrintString())
	}
	return ""
}

func I64ToString(v int64) string {
	return strconv.FormatInt(v, 10)
}

func U64ToString(v uint64) string {
	return strconv.FormatUint(v, 10)
}

func ToBool(v string) bool {
	if v == "" {
		return false
	}
	ret, err := strconv.ParseBool(v)
	if err != nil {
		fmt.Println("ToBool err:", err, "time: ", GetTimePrintString())
		return false
	}
	return ret
}
func ToI32(v string) int32 {
	return int32(ToI64(v))
}

func ToU32(v string) uint32 {
	return uint32(ToU64(v))
}

func ToInt(v string) int {
	ret, err := strconv.Atoi(v)
	if err != nil {
		fmt.Println("ToInt err:", err, "time: ", GetTimePrintString())
		return 0
	}
	return ret
}

func ToI64(v string) int64 {
	if v == "" {
		return 0
	}
	ret, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		fmt.Println("ToI64 err:", err, "time: ", GetTimePrintString())
		return 0
	}
	return ret
}

func ToU64(v string) uint64 {
	if v == "" {
		return 0
	}
	ret, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		fmt.Println("ToU64 err:", err, "time: ", GetTimePrintString())
		return 0
	}
	return ret
}

func ToF64(v string) float64 {
	if v == "" {
		return 0.0
	}
	ret, err := strconv.ParseFloat(v, 64)
	if err != nil {
		fmt.Println("ToF64 err:", err, "time: ", GetTimePrintString())
		return 0.0
	}
	return ret
}