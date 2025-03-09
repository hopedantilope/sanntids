package localOrders

import (
	"Driver-go/elevio"
	"sanntids/cmd/localElevator/config"
	"sanntids/cmd/localElevator/structs"
	"sanntids/cmd/localElevator/fsm"
	"sanntids/cmd/localElevator/elevator"
)

// HallOrderManager listens for button events.
// For BT_Cab events, it updates the CabRequestList and includes it in the HRAElevState.
// For hall events, it sends back a new HallOrder via outgoingOrdersChan.
func HallOrderManager(
	localRequest <-chan elevio.ButtonEvent,
	elevatorCh <-chan elevator.Elevator,
	outgoingOrdersChan chan<- structs.HallOrder,
	outgoingElevStateChan chan<- structs.HRAElevState,
) {
	// Initialize the cab request list as an array to false
	var cabRequests structs.CabRequestList
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
					outgoingElevStateChan <- currentState
				}

			} else if e.Requests[request.Floor][request.Button] == false {
				// For hall button events, create and send a new HallOrder
					newOrder := structs.HallOrder{
						Status:      structs.New,
						DelegatedID: "undelegated",
						Floor:       request.Floor,
						Dir:         request.Button,
					}
					outgoingOrdersChan <- newOrder
				}

		case e = <- elevatorCh:
			currentState.Behavior = elevator.Eb_toString(e.Behaviour)
			currentState.Floor = e.Floor
			currentState.Direction = elevator.Md_toString(e.MotorDirection)

			outgoingElevStateChan <- currentState
		}
	}
}