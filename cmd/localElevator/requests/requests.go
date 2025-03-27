package requests

import (
    "sanntids/cmd/localElevator/elevator"
    "sanntids/cmd/config"
	"Driver-go/elevio"
)

type dirnBehaviourPair struct {
    MotorDirection      elevio.MotorDirection
    Behaviour elevator.ElevatorBehaviour
}

func requestsFloorsAbove(e elevator.Elevator) bool {
    for f := e.Floor + 1; f < config.N_FLOORS; f++ {
        for btn := 0; btn < int(config.N_BUTTONS); btn++ {
            if e.Requests[f][btn] {
                return true
            }
        }
    }
    return false
}

func requestsFloorsBelow(e elevator.Elevator) bool {
    for f := 0; f < e.Floor; f++ {
        for btn := 0; btn < int(config.N_BUTTONS); btn++ {
            if e.Requests[f][btn] {
                return true
            }
        }
    }
    return false
}

func requestsCurrentFloor(e elevator.Elevator) bool {
    for btn := 0; btn < int(config.N_BUTTONS); btn++ {
        if e.Requests[e.Floor][btn] {
            return true
        }
    }
    return false
}

func RequestsChooseDirection(e elevator.Elevator) dirnBehaviourPair {
    switch e.MotorDirection {
    case elevio.MD_Up:
        if requestsFloorsAbove(e) {
            return dirnBehaviourPair{elevio.MD_Up, elevator.EB_Moving}
        } else if requestsCurrentFloor(e) {
            return dirnBehaviourPair{elevio.MD_Down, elevator.EB_DoorOpen}
        } else if requestsFloorsBelow(e) {
            return dirnBehaviourPair{elevio.MD_Down, elevator.EB_Moving}
        } else {
            return dirnBehaviourPair{elevio.MD_Stop, elevator.EB_Idle}
        }

    case elevio.MD_Down:
        if requestsFloorsBelow(e) {
            return dirnBehaviourPair{elevio.MD_Down, elevator.EB_Moving}
        } else if requestsCurrentFloor(e) {
            return dirnBehaviourPair{elevio.MD_Up, elevator.EB_DoorOpen}
        } else if requestsFloorsAbove(e) {
            return dirnBehaviourPair{elevio.MD_Up, elevator.EB_Moving}
        } else {
            return dirnBehaviourPair{elevio.MD_Stop, elevator.EB_Idle}
        }

    case elevio.MD_Stop:
        fallthrough
    default:
        if requestsCurrentFloor(e) {
            return dirnBehaviourPair{elevio.MD_Stop, elevator.EB_DoorOpen}
        } else if requestsFloorsAbove(e) {
            return dirnBehaviourPair{elevio.MD_Up, elevator.EB_Moving}
        } else if requestsFloorsBelow(e) {
            return dirnBehaviourPair{elevio.MD_Down, elevator.EB_Moving}
        } else {
            return dirnBehaviourPair{elevio.MD_Stop, elevator.EB_Idle}
        }
    }
}

func RequestsShouldStop(e elevator.Elevator) bool {
    switch e.MotorDirection {
    case elevio.MD_Down:
        return e.Requests[e.Floor][elevio.BT_HallDown] || 
               e.Requests[e.Floor][elevio.BT_Cab] || 
               !requestsFloorsBelow(e)

    case elevio.MD_Up:
        return e.Requests[e.Floor][elevio.BT_HallUp] || 
               e.Requests[e.Floor][elevio.BT_Cab] ||
               !requestsFloorsAbove(e)

    case elevio.MD_Stop:
        fallthrough
    default:
        return true
    }
}

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

            if !requestsFloorsAbove(e) && !e.Requests[e.Floor][elevio.BT_HallUp] {
                e.Requests[e.Floor][elevio.BT_HallDown] = false
            }

        case elevio.MD_Down:
            e.Requests[e.Floor][elevio.BT_HallDown] = false
            if !requestsFloorsBelow(e) && !e.Requests[e.Floor][elevio.BT_HallDown] {
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
			if !requestsFloorsAbove(e) && e.Requests[floor][elevio.BT_HallDown] {
				cleared[floor][elevio.BT_HallDown] = true
			}
		case elevio.MD_Down:
			if e.Requests[floor][elevio.BT_HallDown] {
				cleared[floor][elevio.BT_HallDown] = true
			}
			if !requestsFloorsBelow(e) && e.Requests[floor][elevio.BT_HallUp] {
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