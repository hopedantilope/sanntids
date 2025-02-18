// broadcastState.go
package broadcastState

import (
	"Network-go/network/bcast"
	"fmt"
	"sanntids/cmd/localElevator/fsm"
)


type ElevatorStateWithID struct {
	ElevatorID   string
	ElevatorState fsm.ElevatorState
}


func BroadcastState(stateChan <-chan ElevatorStateWithID, port int) {
	fmt.Printf("Starting state broadcaster on port %d\n", port)
	broadcastChan := make(chan ElevatorStateWithID)

	go bcast.Transmitter(port, broadcastChan)

	for stateWithID := range stateChan {
		broadcastChan <- stateWithID
	}
}

func ReceiveState(stateChan chan<- ElevatorStateWithID, port int) {
	fmt.Printf("Starting state receiver on port %d\n", port)
	receiveChan := make(chan ElevatorStateWithID)

	go bcast.Receiver(port, receiveChan)

	for receivedStateWithID := range receiveChan {
		stateChan <- receivedStateWithID
	}
}