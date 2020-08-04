package util

import (
	"time"
)

func Timestamp() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}
