package elevator

import (
	"sanntids/cmd/localElevator/config"
)

type ElevatorBehaviour int

const (
	EB_Idle ElevatorBehaviour = iota
	EB_DoorOpen
	EB_Moving
)

type ClearRequestVariant int

const (
	// Assume everyone waiting for the elevator gets on the elevator, even if
	// they will be traveling in the "wrong" direction for a while
	CV_All ClearRequestVariant = iota

	// Assume that only those that want to travel in the current direction
	// enter the elevator, and keep waiting outside otherwise
	CV_InDirn
)

type Dirn int

type Elevator struct {
	floor     int
	dirn      Dirn
	requests  [config.N_FLOORS][config.N_BUTTONS]int
	behaviour ElevatorBehaviour

	Config struct {
		clearRequestVariant ClearRequestVariant
		doorOpenDuration_s  float64
	}
}
