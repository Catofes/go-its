package gateway

import "time"

func int2Time(milliSecond uint64) time.Duration {
	return time.Duration(milliSecond) * time.Millisecond
}
