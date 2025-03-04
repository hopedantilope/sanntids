package main

import (
	"Driver-go/elevio"
	"fmt"
	"sanntids/cmd/localElevator/fsm"
	"sanntids/cmd/shareOrders"
	"sanntids/cmd/network/broadcastState"
	"sanntids/cmd/elevatorController/runHRA"
	"time"
	"Network-go/network/localip"
)



func main() {
	runHRA.Test()
	numFloors := 4

	elevio.Init("localhost:15657", numFloors)
	id, _ := localip.LocalIP()
	fmt.Println("Elevator started with id: ", id)

	drv_buttons := make(chan elevio.ButtonEvent)
	drv_floors := make(chan int)
	drv_obstr := make(chan bool)
	drv_stop := make(chan bool)

	go elevio.PollButtons(drv_buttons)
	go elevio.PollFloorSensor(drv_floors)
	go elevio.PollObstructionSwitch(drv_obstr)
	go elevio.PollStopButton(drv_stop)

	// Channel for FSM state requests
	stateRequestTx := make(chan chan fsm.ElevatorState)
	
	// Channels for hall order handling
	hallOrderChan := make(chan elevio.ButtonEvent)
	completedOrderChan := make(chan shareOrders.HallOrder)
	
	// Split the button events between FSM and hall order manager
	buttonSplitter := make(chan elevio.ButtonEvent)
	go func() {
		for btn := range drv_buttons {
			// Send to FSM
			buttonSplitter <- btn
			
			// Also send hall orders to hall order manager
			if btn.Button != elevio.BT_Cab {
				hallOrderChan <- btn
			}
		}
	}()

	// Start the FSM with the original parameters
	go fsm.Fsm(buttonSplitter, drv_floors, drv_obstr, drv_stop, stateRequestTx)

	// Setup network communication
	broadcastPort := 30003
	broadcastDataChan := make(chan broadcastState.ElevatorDataWithID)
	receiveDataChan := make(chan broadcastState.ElevatorDataWithID)
	
	// Internal channels for hall order manager
	incomingOrdersChan := make(chan []shareOrders.HallOrder)
	outgoingOrdersChan := make(chan []shareOrders.HallOrder)
	
	// Start hall order manager
	go shareOrders.HallOrderManager(incomingOrdersChan, hallOrderChan, outgoingOrdersChan, completedOrderChan)

	// Start network communication
	go broadcastState.BroadcastState(broadcastDataChan, broadcastPort)
	go broadcastState.ReceiveState(receiveDataChan, broadcastPort)

	// Setup periodic broadcasting
	broadcastTicker := time.NewTicker(1 * time.Second)
	defer broadcastTicker.Stop()

	// Track local hall orders
	var localHallOrders []shareOrders.HallOrder

	for {
		select {
		case <-broadcastTicker.C:
			// Get current elevator state
			requestChan := make(chan fsm.ElevatorState)
			stateRequestTx <- requestChan
			currentState := <-requestChan
			
			// Create combined data packet with state and orders
			dataWithID := broadcastState.ElevatorDataWithID{
				ElevatorID:    id,
				ElevatorState: currentState,
				HallOrders:    localHallOrders,
			}
			
			// Broadcast
			broadcastDataChan <- dataWithID

		case receivedDataWithID := <-receiveDataChan:
			if receivedDataWithID.ElevatorID != id {
				incomingOrdersChan <- receivedDataWithID.HallOrders
			}
			
		case updatedOrders := <-outgoingOrdersChan:
			localHallOrders = updatedOrders
		}
	}
}