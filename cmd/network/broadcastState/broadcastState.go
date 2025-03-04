package broadcastState

import (
	"Network-go/network/bcast"
	"fmt"
	"sanntids/cmd/localElevator/fsm"
	"sanntids/cmd/shareOrders"
)

// Combined structure for broadcasting both elevator state and hall orders
type ElevatorDataWithID struct {
	ElevatorID    string
	ElevatorState fsm.ElevatorState
	HallOrders    []shareOrders.HallOrder
}

func BroadcastState(dataChan <-chan ElevatorDataWithID, port int) {
	fmt.Printf("Starting data broadcaster on port %d\n", port)
	broadcastChan := make(chan ElevatorDataWithID)

	go bcast.Transmitter(port, broadcastChan)

	for dataWithID := range dataChan {
		broadcastChan <- dataWithID
	}
}

func ReceiveState(dataChan chan<- ElevatorDataWithID, port int) {
	fmt.Printf("Starting data receiver on port %d\n", port)
	receiveChan := make(chan ElevatorDataWithID)

	go bcast.Receiver(port, receiveChan)

	for receivedDataWithID := range receiveChan {
		dataChan <- receivedDataWithID
	}
}