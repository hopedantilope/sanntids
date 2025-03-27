package timer

import (
	"time"
)

var timeoutChan = make(chan bool)
var doorTimer *time.Timer
var disabled bool

// TimerStart starts a timer that will send a timeout event after the given duration,
// but if the timer is disabled, it simply stops any running timer and does nothing.
func TimerStart(duration float64) {
	if disabled {
		if doorTimer != nil {
			doorTimer.Stop()
		}
		return
	}
	if doorTimer != nil {
		doorTimer.Stop()
	}
	doorTimer = time.AfterFunc(time.Duration(duration)*time.Second, func() {
		if !disabled {
			timeoutChan <- true
		}
	})
}

func TimerStop() {
	if doorTimer != nil {
		doorTimer.Stop()
	}
}

func TimerDisable() {
	disabled = true
	if doorTimer != nil {
		doorTimer.Stop()
	}
}

func TimerEnable() {
	disabled = false
}

func TimeoutChan() <-chan bool {
	return timeoutChan
}
