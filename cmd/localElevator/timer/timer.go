package timer

import (
	"time"
)

var (
	timerrEndTime float64
	timerActive   bool
)

func getWallTime() float64 {
	now := time.Now()
	return float64(now.Unix()) + float64(now.Nanosecond())*1e-9
}

func Timerstart(duration float64) {
	timerEndTime = getWallTime() + duration
	timerActive = 1
}
func TimerStop() {
	timerAcive = 0
}

func TimerTimedOut() {
	return timerActive && getWallTime() > timerEndTime
}
