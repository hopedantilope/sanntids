package localOrders

import (
	"Driver-go/elevio"
	"sanntids/cmd/localElevator/config"
	"sanntids/cmd/localElevator/elevator"
	"sanntids/cmd/localElevator/structs"
	"fmt"
)

// LocalStateManager listens for button events.
// For BT_Cab events, it updates the CabRequestList and includes it in the HRAElevState.
// For hall events, it sends back a new HallOrder via outgoingOrdersChan.
func LocalStateManager(
	localRequest <-chan elevio.ButtonEvent,
	elevatorCh <-chan elevator.Elevator,
	outgoingOrdersChan chan<- structs.HallOrder,
	outgoingElevStateChan chan<- structs.HRAElevState,
	completedRequetsChan chan<- []elevio.ButtonEvent) {
	// Initialize the cab request list as an array to false
	cabRequests := make([]bool, config.N_FLOORS)
	for i := range cabRequests {
		cabRequests[i] = false
	}

	// Initialize with default values
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
			// Handle cab button press
			if request.Button == elevio.BT_Cab {
				if request.Floor >= 0 && request.Floor < config.N_FLOORS {
					cabRequests[request.Floor] = true
					currentState.CabRequests = cabRequests
					fmt.Println("Sending updated elevator state (cab request updated):", currentState)
					outgoingElevStateChan <- currentState
				}

			} else if !e.Requests[request.Floor][request.Button] {
				// For hall button events, create and send a new HallOrder
				newOrder := structs.HallOrder{
					Status:      structs.New,
					DelegatedID: "undelegated",
					Floor:       request.Floor,
					Dir:         request.Button,
				}
				fmt.Println("Sending new hall order:", newOrder)
				outgoingOrdersChan <- newOrder
			}

		case e = <-elevatorCh:
			currentState.Behavior = elevator.Eb_toString(e.Behaviour)
			currentState.Floor = e.Floor
			currentState.Direction = elevator.Md_toString(e.MotorDirection)
			// To make sure completed cab requests gets reset to false:
			currentState.CabRequests = elevator.GetCabRequests(e.Requests)
			//Sending completed requets to a channel
			completedRequests := getClearedHallRequests(e.Cleared)
			if len(completedRequests) > 0 {
				fmt.Println("Sending completed hall requests:", completedRequests)
				completedRequetsChan <- completedRequests
			}
			fmt.Println("Sending updated elevator state (elevator update):", currentState)
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
