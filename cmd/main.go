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
	"sanntids/cmd/localOrders"
	"sanntids/cmd/broadcastState"
	"sanntids/cmd/networkOrders"
)

func main() {
	// Parse command-line arguments
	port := flag.String("port", "15657", "Port number for elevator simulator")
	elevatorID := flag.String("id", "", "Elevator ID (defaults to local IP if not specified)")
	broadcastPortFlag := flag.Int("broadcast", 30003, "Port for broadcasting state")
	numFloorsFlag := flag.Int("floors", config.N_FLOORS, "Number of floors")
	flag.Parse()

	// Configure the simulator
	numFloors := *numFloorsFlag
	elevPort := fmt.Sprintf("localhost:%s", *port)

	// if not set use ip adress
	if *elevatorID == "" {
		*elevatorID, _ = localip.LocalIP()
	}
	fmt.Printf("Local elevator ID: %s, Network port: %d\n", *elevatorID, *broadcastPortFlag)

	// Initialize the elevator driver
	elevio.Init(elevPort, numFloors)

	// Create channels for driver inputs
	drv_buttons := make(chan elevio.ButtonEvent)
	drv_floors := make(chan int)
	drv_obstr := make(chan bool)
	drv_stop := make(chan bool)

	// Start polling inputs concurrently
	go elevio.PollButtons(drv_buttons)
	go elevio.PollFloorSensor(drv_floors)
	go elevio.PollObstructionSwitch(drv_obstr)
	go elevio.PollStopButton(drv_stop)

	// FSM and state channels
	elevatorCh := make(chan elevator.Elevator)
	requestsToLocalChan := make(chan [config.N_FLOORS][config.N_BUTTONS]bool)
	// Local order channels
	outgoingLocalOrdersChan := make(chan structs.HallOrder)
	outgoingLocalElevStateChan := make(chan structs.HRAElevState)
	completedRequetsChan := make(chan []elevio.ButtonEvent)

	// Network communication channels
	incomingNetworkData := make(chan structs.ElevatorDataWithID)
	outgoingNetworkData := make(chan structs.ElevatorDataWithID)

	

	go fsm.Fsm(requestsToLocalChan, drv_floors, drv_obstr, drv_stop, elevatorCh)

	go localOrders.LocalStateManager(
		drv_buttons,
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
