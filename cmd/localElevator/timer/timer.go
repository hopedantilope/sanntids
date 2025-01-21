package timer

import (
	"time"
)

var (
	timerEndTime float64
	timerActive  bool
)

func getWallTime() float64 {
	now := time.Now()
	return float64(now.Unix()) + float64(now.Nanosecond())*1e-9
}

func TimerStart(duration float64) {
	timerEndTime = getWallTime() + duration
	timerActive = true
}

func TimerStop() {
	timerActive = false
}

func TimerTimedOut() bool {
	return timerActive && getWallTime() > timerEndTime
}
