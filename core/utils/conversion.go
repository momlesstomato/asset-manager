package utils

import (
	"fmt"
	"strconv"
	"strings"
)

// ToInt converts various types to int using explicit type switching.
// It handles standard integer types, floats, strings, and byte slices.
func ToInt(val any) int {
	switch v := val.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case int32:
		return int(v)
	case int16:
		return int(v)
	case int8:
		return int(v)
	case uint:
		return int(v)
	case uint64:
		return int(v)
	case uint32:
		return int(v)
	case uint16:
		return int(v)
	case uint8:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	case string:
		i, _ := strconv.Atoi(v)
		return i
	case []byte:
		i, _ := strconv.Atoi(string(v))
		return i
	default:
		s := fmt.Sprintf("%v", v)
		i, _ := strconv.Atoi(s)
		return i
	}
}

// ToString converts various types to string.
func ToString(val any) string {
	switch v := val.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// ToBool converts various types to bool.
// It handles bool, numeric types (1=true), and strings ("1", "true").
func ToBool(val any) bool {
	switch v := val.(type) {
	case bool:
		return v
	case int, int64, int32, int16, int8, uint, uint64, uint32, uint16, uint8:
		return ToInt(v) == 1
	case string:
		return v == "1" || strings.ToLower(v) == "true"
	case []byte:
		s := string(v)
		return s == "1" || strings.ToLower(s) == "true"
	default:
		return false
	}
}
