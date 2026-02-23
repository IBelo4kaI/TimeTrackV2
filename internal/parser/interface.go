package parser

import "strconv"

// Helper function to convert interface{} to int64
func InterfaceToInt64(val interface{}) int64 {
	if val == nil {
		return 0
	}
	switch v := val.(type) {
	case int64:
		return v
	case int32:
		return int64(v)
	case int:
		return int64(v)
	case float64:
		return int64(v)
	case float32:
		return int64(v)
	case []byte:
		// MySQL values might be returned as []byte
		str := string(v)
		f, err := strconv.ParseFloat(str, 64)
		if err != nil {
			return 0
		}
		return int64(f)
	default:
		return 0
	}
}

// Helper function to convert interface{} to float32
func InterfaceToFloat32(val interface{}) float32 {
	if val == nil {
		return 0
	}
	switch v := val.(type) {
	case float32:
		return v
	case float64:
		return float32(v)
	case int64:
		return float32(v)
	case int32:
		return float32(v)
	case int:
		return float32(v)
	case []byte:
		// MySQL DECIMAL values are returned as []byte
		str := string(v)
		f, err := strconv.ParseFloat(str, 32)
		if err != nil {
			return 0
		}
		return float32(f)
	default:
		return 0
	}
}

func InterfaceToInt(val interface{}) int {
	if val == nil {
		return 0
	}
	switch v := val.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case int32:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	case []byte:
		str := string(v)
		f, err := strconv.ParseFloat(str, 64)
		if err != nil {
			return 0
		}
		return int(f)
	default:
		return 0
	}
}
