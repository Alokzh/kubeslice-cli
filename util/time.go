package util

import (
	"time"
)

func Sleep(d time.Duration) {
	SystemClock.Sleep(d)
}

func Now() time.Time {
	return SystemClock.Now()
}
