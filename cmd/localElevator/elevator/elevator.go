package elevator
import (
	"sanntids/cmd/localElevator/config"
	"Driver-go/elevio"
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

type Button int

const (
    BT_HallUp   Button = iota
    BT_HallDown
    BT_Cab
    N_BUTTONS // This will be 3
)

type Elevator struct {
    Floor     int
    MotorDirection      elevio.MotorDirection
    Requests  [config.N_FLOORS][config.N_BUTTONS]bool
    Behaviour ElevatorBehaviour

    Config struct {
        ClearRequestVariant ClearRequestVariant
        DoorOpenDuration_s  float64
    }
}