package main

import (
	"Driver-go/elevio"
	"Network-go/network/localip"
	"flag"
	"fmt"
	"sanntids/cmd/config"
	"sanntids/cmd/localElevator/elevator"
	"sanntids/cmd/localElevator/fsm"
	"sanntids/cmd/structs"
	"sanntids/cmd/localStates"
	"sanntids/cmd/broadcastState"
	"sanntids/cmd/networkOrders"
)

func main() {
	// Parse command-line arguments
	port := flag.String("port", "15657", "Port number for elevator")
	elevatorID := flag.String("id", "", "Elevator ID (defaults to local IP if not specified)")
	broadcastPortFlag := flag.Int("broadcast", 30003, "Port for broadcasting state")
	flag.Parse()

	numFloors := config.N_FLOORS
	elevPort := fmt.Sprintf("localhost:%s", *port)

	// if not set use ip adress
	if *elevatorID == "" {
		*elevatorID, _ = localip.LocalIP()
	}
	fmt.Printf("Local elevator ID: %s, Network port: %d\n", *elevatorID, *broadcastPortFlag)

	// Initialize the elevator driver
	elevio.Init(elevPort, numFloors)

	// Create channels for driver inputs
	drvButtons := make(chan elevio.ButtonEvent)
	drvFloors := make(chan int)
	drvObstr := make(chan bool)
	drvStop := make(chan bool)

	// Start polling inputs concurrently
	go elevio.PollButtons(drvButtons)
	go elevio.PollFloorSensor(drvFloors)
	go elevio.PollObstructionSwitch(drvObstr)
	go elevio.PollStopButton(drvStop)

	// FSM and state channels
	elevatorCh := make(chan elevator.Elevator)
	requestsToLocalChan := make(chan [config.N_FLOORS][config.N_BUTTONS]bool)

	// Local channels
	outgoingLocalOrdersChan := make(chan structs.HallOrder)
	outgoingLocalElevStateChan := make(chan structs.HRAElevState)
	completedRequetsChan := make(chan []elevio.ButtonEvent)

	// Network communication channels
	incomingNetworkData := make(chan structs.ElevatorDataWithID)
	outgoingNetworkData := make(chan structs.ElevatorDataWithID)

	go fsm.Fsm(requestsToLocalChan, drvFloors, drvObstr, drvStop, elevatorCh)

	go localStates.LocalStateManager(
		drvButtons,
		elevatorCh,
		outgoingLocalOrdersChan,
		outgoingLocalElevStateChan,
		completedRequetsChan,
	)

	go networkOrders.NetworkOrderManager(
		*elevatorID,
		outgoingLocalElevStateChan,
		outgoingLocalOrdersChan,
		completedRequetsChan,
		incomingNetworkData,
		outgoingNetworkData,
		requestsToLocalChan,
	)

	go broadcastState.BroadcastState(outgoingNetworkData, *broadcastPortFlag)
	go broadcastState.ReceiveState(incomingNetworkData, *broadcastPortFlag)

	select {}
}
