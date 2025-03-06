package broadcastState

import (
	"Network-go/network/bcast"
	"fmt"
	"math"
	"sanntids/cmd/localElevator/config"
	"sanntids/cmd/localElevator/elevator"
	"sanntids/cmd/localElevator/fsm"
	"sanntids/cmd/localOrders"
)

// HRACompatible structure for use with the Hall Request Assigner
type HRAElevState struct {
	Behavior    string  `json:"behaviour"`
	Floor       int     `json:"floor"`
	Direction   string  `json:"direction"`
	CabRequests []bool  `json:"cabRequests"`
}

// Combined structure for broadcasting both elevator state and hall orders
type ElevatorDataWithID struct {
	ElevatorID    string
	ElevatorState fsm.ElevatorState
	HallOrders    []localOrders.HallOrder
}

// BroadcastState transmits elevator state and hall orders in chunks
func BroadcastState(dataChan <-chan ElevatorDataWithID, port int) {
	fmt.Printf("Starting data broadcaster on port %d\n", port)
	broadcastChan := make(chan ElevatorDataWithID)
	
	go bcast.Transmitter(port, broadcastChan)
	
	for dataWithID := range dataChan {
		// Split orders into chunks of max 3 per message
		if len(dataWithID.HallOrders) <= 3 {
			// Send as-is if 3 or fewer orders
			broadcastChan <- dataWithID
		} else {
			// Split orders into chunks of 3
			totalOrders := len(dataWithID.HallOrders)
			chunkSize := 3
			numChunks := int(math.Ceil(float64(totalOrders) / float64(chunkSize)))
			
			for i := 0; i < numChunks; i++ {
				start := i * chunkSize
				end := (i + 1) * chunkSize
				
				if end > totalOrders {
					end = totalOrders
				}
				
				// Create new data packet with just this chunk of orders
				chunkData := ElevatorDataWithID{
					ElevatorID:    dataWithID.ElevatorID,
					ElevatorState: dataWithID.ElevatorState,
					HallOrders:    dataWithID.HallOrders[start:end],
				}
				
				broadcastChan <- chunkData
			}
		}
	}
}

// ReceiveState receives elevator state and hall orders
func ReceiveState(dataChan chan<- ElevatorDataWithID, port int) {
	fmt.Printf("Starting data receiver on port %d\n", port)
	receiveChan := make(chan ElevatorDataWithID)
	
	go bcast.Receiver(port, receiveChan)
	
	for receivedDataWithID := range receiveChan {
		// Simply forward the data to the main application
		dataChan <- receivedDataWithID
	}
}

// ConvertToHRAInput converts from our internal state to the HRA input format
func ConvertToHRAInput(elevStates map[string]fsm.ElevatorState, hallOrders localOrders.OrderMap) HRAInput {
	hraInput := HRAInput{
		States: make(map[string]HRAElevState),
	}
	
	hraInput.HallRequests = localOrders.ConvertToHRAHallRequests(hallOrders)
	
	for id, state := range elevStates {
		hraElev := HRAElevState{
			Behavior:    elevator.Eb_toString(state.Behaviour),
			Floor:       state.Floor,
			Direction:   elevator.Md_toString(state.MotorDirection),
			CabRequests: make([]bool, config.N_FLOORS),
		}
		
		// Copy cab requests
		for i, req := range state.CabRequests {
			hraElev.CabRequests[i] = req
		}
		
		hraInput.States[id] = hraElev
	}
	
	return hraInput
}

// Define HRAInput struct to match the format expected by Hall Request Assigner
type HRAInput struct {
	HallRequests [config.N_FLOORS][2]bool    `json:"hallRequests"`
	States       map[string]HRAElevState     `json:"states"`
}