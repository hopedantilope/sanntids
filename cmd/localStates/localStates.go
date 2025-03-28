package localStates

import (
	"Driver-go/elevio"
	"sanntids/cmd/config"
	"sanntids/cmd/localElevator/elevator"
	"sanntids/cmd/structs"
)

func LocalStateManager(
	localRequest <-chan elevio.ButtonEvent,
	elevatorCh <-chan elevator.Elevator,
	outgoingOrdersChan chan<- structs.HallOrder,
	outgoingElevStateChan chan<- structs.HRAElevState,
	completedRequetsChan chan<- []elevio.ButtonEvent) {

	cabRequests := make([]bool, config.N_FLOORS)
	for i := range cabRequests {
		cabRequests[i] = false
	}

	currentState := structs.HRAElevState{
		Behavior:    "idle",
		Floor:       0,
		Direction:   "stop",
		CabRequests: cabRequests,
	}

	e := elevator.ElevatorInit()

	for {
		select {
		case request := <-localRequest:
			if request.Button == elevio.BT_Cab {
				if request.Floor >= 0 && request.Floor < config.N_FLOORS {
					currentState.CabRequests[request.Floor] = true
					outgoingElevStateChan <- currentState
				}

			} else if !e.Requests[request.Floor][request.Button] {
				newOrder := structs.HallOrder{
					Status:      structs.New,
					DelegatedID: "undelegated",
					Floor:       request.Floor,
					Dir:         request.Button,
				}
				outgoingOrdersChan <- newOrder
			}

		case e = <-elevatorCh:
			currentState.Behavior = elevatorBehaviourToString(e.Behaviour)
			currentState.Floor = e.Floor
			currentState.Direction = motorDirectionToString(e.MotorDirection)
			currentState.Obstruction = e.Obstruction
			currentState.Stop = e.Stop
			currentState.CabRequests = elevator.GetCabRequests(e.Requests)
			completedRequests := getClearedHallRequests(e.Cleared)
			if len(completedRequests) > 0 {
				completedRequetsChan <- completedRequests
			}
			outgoingElevStateChan <- currentState
		}
	}
}

func getClearedHallRequests(cleared [config.N_FLOORS][config.N_BUTTONS]bool) []elevio.ButtonEvent {
	var requests []elevio.ButtonEvent
	for floor := 0; floor < config.N_FLOORS; floor++ {
		for btn := 0; btn < config.N_BUTTONS; btn++ {
			if (cleared[floor][btn]) && (elevio.ButtonType(btn) != elevio.BT_Cab) {
				requests = append(requests, elevio.ButtonEvent{
					Floor:  floor,
					Button: elevio.ButtonType(btn),
				})
			}
		}
	}
	return requests
}


func elevatorBehaviourToString(eb elevator.ElevatorBehaviour) string {
    switch eb {
    case elevator.EB_Idle:
        return "idle"
    case elevator.EB_DoorOpen:
        return "doorOpen"
    case elevator.EB_Moving:
        return "moving"
    default:
        return "unknown"
    }
}

func motorDirectionToString(md elevio.MotorDirection) string {
    switch md {
    case elevio.MD_Up:
        return "up"
    case elevio.MD_Down:
        return "down"
    case elevio.MD_Stop:
        return "stop"
    default:
        return "unknown"
    }
}