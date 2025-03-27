package broadcastState

import (
	"Network-go/network/bcast"
	"sanntids/cmd/structs"
)

func BroadcastState(dataChan <-chan structs.ElevatorDataWithID, port int) {
	broadcastChan := make(chan structs.ElevatorDataWithID)
	go bcast.Transmitter(port, broadcastChan)
	for dataWithID := range dataChan {
		broadcastChan <- dataWithID
	}
}

func ReceiveState(dataChan chan<- structs.ElevatorDataWithID, port int) {
	receiveChan := make(chan structs.ElevatorDataWithID)
	go bcast.Receiver(port, receiveChan)
	for receivedDataWithID := range receiveChan {
		dataChan <- receivedDataWithID
	}
}
