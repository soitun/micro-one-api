package safecast

import (
	"fmt"
	"math"
)

func IntToInt32(v int) (int32, error) {
	if v < math.MinInt32 || v > math.MaxInt32 {
		return 0, fmt.Errorf("value %d overflows int32", v)
	}
	return int32(v), nil // #nosec G115 -- range checked above.
}

func Int64ToInt32(v int64) (int32, error) {
	if v < math.MinInt32 || v > math.MaxInt32 {
		return 0, fmt.Errorf("value %d overflows int32", v)
	}
	return int32(v), nil // #nosec G115 -- range checked above.
}

func Int64ToUint(v int64) (uint, error) {
	if v < 0 || uint64(v) > uint64(math.MaxUint) {
		return 0, fmt.Errorf("value %d overflows uint", v)
	}
	return uint(v), nil // #nosec G115 -- range checked above.
}

func Int32ToInt8(v int32) (int8, error) {
	if v < math.MinInt8 || v > math.MaxInt8 {
		return 0, fmt.Errorf("value %d overflows int8", v)
	}
	return int8(v), nil // #nosec G115 -- range checked above.
}

func UintToInt64(v uint) (int64, error) {
	if uint64(v) > uint64(math.MaxInt64) {
		return 0, fmt.Errorf("value %d overflows int64", v)
	}
	return int64(v), nil // #nosec G115 -- range checked above.
}

func UintToUint32(v uint) (uint32, error) {
	if v > math.MaxUint32 {
		return 0, fmt.Errorf("value %d overflows uint32", v)
	}
	return uint32(v), nil // #nosec G115 -- range checked above.
}
