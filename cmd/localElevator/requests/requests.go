// File: cmd/localElevator/requests/requests.go

package requests

import (
    "sanntids/cmd/localElevator/elevator"
    "sanntids/cmd/config"
	"Driver-go/elevio"
)

// Use elevator package types
type DirnBehaviourPair struct {
    MotorDirection      elevio.MotorDirection
    Behaviour elevator.ElevatorBehaviour
}

// Check if there are any requests above the current floor
func requestsAbove(e elevator.Elevator) bool {
    for f := e.Floor + 1; f < config.N_FLOORS; f++ {
        for btn := 0; btn < int(config.N_BUTTONS); btn++ {
            if e.Requests[f][btn] {
                return true
            }
        }
    }
    return false
}

// Check if there are any requests below the current floor
func requestsBelow(e elevator.Elevator) bool {
    for f := 0; f < e.Floor; f++ {
        for btn := 0; btn < int(config.N_BUTTONS); btn++ {
            if e.Requests[f][btn] {
                return true
            }
        }
    }
    return false
}

// Check if there is a request on the current floor
func requestsHere(e elevator.Elevator) bool {
    for btn := 0; btn < int(config.N_BUTTONS); btn++ {
        if e.Requests[e.Floor][btn] {
            return true
        }
    }
    return false
}

// Decide which direction to move next, based on current requests and direction
func RequestsChooseDirection(e elevator.Elevator) DirnBehaviourPair {
    switch e.MotorDirection {
    case elevio.MD_Up:
        if requestsAbove(e) {
            return DirnBehaviourPair{elevio.MD_Up, elevator.EB_Moving}
        } else if requestsHere(e) {
            return DirnBehaviourPair{elevio.MD_Down, elevator.EB_DoorOpen}
        } else if requestsBelow(e) {
            return DirnBehaviourPair{elevio.MD_Down, elevator.EB_Moving}
        } else {
            return DirnBehaviourPair{elevio.MD_Stop, elevator.EB_Idle}
        }

    case elevio.MD_Down:
        if requestsBelow(e) {
            return DirnBehaviourPair{elevio.MD_Down, elevator.EB_Moving}
        } else if requestsHere(e) {
            return DirnBehaviourPair{elevio.MD_Up, elevator.EB_DoorOpen}
        } else if requestsAbove(e) {
            return DirnBehaviourPair{elevio.MD_Up, elevator.EB_Moving}
        } else {
            return DirnBehaviourPair{elevio.MD_Stop, elevator.EB_Idle}
        }

    case elevio.MD_Stop:
        fallthrough
    default:
        if requestsHere(e) {
            return DirnBehaviourPair{elevio.MD_Stop, elevator.EB_DoorOpen}
        } else if requestsAbove(e) {
            return DirnBehaviourPair{elevio.MD_Up, elevator.EB_Moving}
        } else if requestsBelow(e) {
            return DirnBehaviourPair{elevio.MD_Down, elevator.EB_Moving}
        } else {
            return DirnBehaviourPair{elevio.MD_Stop, elevator.EB_Idle}
        }
    }
}

// Determine if the elevator should stop at the current floor
func RequestsShouldStop(e elevator.Elevator) bool {
    switch e.MotorDirection {
    case elevio.MD_Down:
        return e.Requests[e.Floor][elevio.BT_HallDown] || 
               e.Requests[e.Floor][elevio.BT_Cab] || 
               !requestsBelow(e)

    case elevio.MD_Up:
        return e.Requests[e.Floor][elevio.BT_HallUp] || 
               e.Requests[e.Floor][elevio.BT_Cab] ||
               !requestsAbove(e)

    case elevio.MD_Stop:
        fallthrough
    default:
        return true
    }
}

// Decide if a request should be cleared immediately based on variant
func RequestsShouldClearImmediately(e elevator.Elevator, btnFloor int, btnType elevio.ButtonType) bool {
    switch e.Config.ClearRequestVariant {
    case config.CV_All:
        return e.Floor == btnFloor

    case config.CV_InDirn:
        return e.Floor == btnFloor &&
            ((e.MotorDirection == elevio.MD_Up && btnType == elevio.BT_HallUp) ||
                (e.MotorDirection == elevio.MD_Down && btnType == elevio.BT_HallDown) ||
                e.MotorDirection == elevio.MD_Stop ||
                btnType == elevio.BT_Cab)

    default:
        return false
    }
}

func RequestsClearAtCurrentFloor(e elevator.Elevator) elevator.Elevator {
    switch e.Config.ClearRequestVariant {
    case config.CV_All:
        for btn := 0; btn < int(config.N_BUTTONS); btn++ {
            e.Requests[e.Floor][btn] = false
        }

    case config.CV_InDirn:
        e.Requests[e.Floor][elevio.BT_Cab] = false

        switch e.MotorDirection {
        case elevio.MD_Up:
            e.Requests[e.Floor][elevio.BT_HallUp] = false

            if !requestsAbove(e) && !e.Requests[e.Floor][elevio.BT_HallUp] {
                e.Requests[e.Floor][elevio.BT_HallDown] = false
            }

        case elevio.MD_Down:
            e.Requests[e.Floor][elevio.BT_HallDown] = false
            if !requestsBelow(e) && !e.Requests[e.Floor][elevio.BT_HallDown] {
                e.Requests[e.Floor][elevio.BT_HallUp] = false
            }

        case elevio.MD_Stop:
            fallthrough
        default:
            e.Requests[e.Floor][elevio.BT_HallUp] = false
            e.Requests[e.Floor][elevio.BT_HallDown] = false
        }
    }
    return e
}

func RequestsGetClearedAtCurrentFloor(e elevator.Elevator) [config.N_FLOORS][config.N_BUTTONS]bool {
	var cleared [config.N_FLOORS][config.N_BUTTONS]bool
	floor := e.Floor
	switch e.Config.ClearRequestVariant {
	case config.CV_All:
		for btn := 0; btn < config.N_BUTTONS; btn++ {
			if e.Requests[floor][btn] {
				cleared[floor][btn] = true
			}
		}
	case config.CV_InDirn:
		if e.Requests[floor][elevio.BT_Cab] {
			cleared[floor][elevio.BT_Cab] = true
		}
		switch e.MotorDirection {
		case elevio.MD_Up:
			if e.Requests[floor][elevio.BT_HallUp] {
				cleared[floor][elevio.BT_HallUp] = true
			}
			if !requestsAbove(e) && e.Requests[floor][elevio.BT_HallDown] {
				cleared[floor][elevio.BT_HallDown] = true
			}
		case elevio.MD_Down:
			if e.Requests[floor][elevio.BT_HallDown] {
				cleared[floor][elevio.BT_HallDown] = true
			}
			if !requestsBelow(e) && e.Requests[floor][elevio.BT_HallUp] {
				cleared[floor][elevio.BT_HallUp] = true
			}
		default:
			if e.Requests[floor][elevio.BT_HallUp] {
				cleared[floor][elevio.BT_HallUp] = true
			}
			if e.Requests[floor][elevio.BT_HallDown] {
				cleared[floor][elevio.BT_HallDown] = true
			}
		}
	}
	return cleared
}


