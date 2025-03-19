package broadcastState

import (
	"Network-go/network/bcast"
	"fmt"
	"math/rand"
	"time"
	"sanntids/cmd/localElevator/structs"
)

// BroadcastState transmits elevator state and hall orders in chunks
// with a 30% chance of packet loss
func BroadcastState(dataChan <-chan structs.ElevatorDataWithID, port int) {
	fmt.Printf("Starting data broadcaster on port %d (with 30%% packet loss simulation)\n", port)
	broadcastChan := make(chan structs.ElevatorDataWithID)
	
	// Initialize random number generator
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	
	go bcast.Transmitter(port, broadcastChan)
	
	for dataWithID := range dataChan {
		// Simulate 30% packet loss
		if r.Float64() > 0.3 {
			broadcastChan <- dataWithID
		} else {
			fmt.Println("Simulating packet loss - dropping broadcast packet")
		}
	}
}

// ReceiveState receives elevator state and hall orders
func ReceiveState(dataChan chan<- structs.ElevatorDataWithID, port int) {
	fmt.Printf("Starting data receiver on port %d\n", port)
	receiveChan := make(chan structs.ElevatorDataWithID)
	
	go bcast.Receiver(port, receiveChan)
	
	for receivedDataWithID := range receiveChan {
		dataChan <- receivedDataWithID
	}
}
