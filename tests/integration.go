package main

import (
	"Driver-go/elevio"
	"Network-go/network/localip"
	"flag"
	"fmt"
	"sanntids/cmd/localElevator/config"
	"sanntids/cmd/localElevator/elevator"
	"sanntids/cmd/localElevator/fsm"
	"sanntids/cmd/localElevator/structs"
	"sanntids/cmd/localOrders"
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

	// if not set use IP address
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

	// We want to duplicate drv_buttons for two consumers:
	// one for the local state machine and one for the FSM.
	localStateButtons := make(chan elevio.ButtonEvent)
	fsmButtons := make(chan elevio.ButtonEvent)

	// Start polling inputs concurrently
	go elevio.PollButtons(drv_buttons)
	go elevio.PollFloorSensor(drv_floors)
	go elevio.PollObstructionSwitch(drv_obstr)
	go elevio.PollStopButton(drv_stop)

	// Tee the events from drv_buttons into two channels.
	go func() {
		for event := range drv_buttons {
			// Send the event to both consumers.
			localStateButtons <- event
			fsmButtons <- event
		}
	}()

	// FSM and state channels
	elevatorCh := make(chan elevator.Elevator)

	// Local order channels
	outgoingOrdersChan := make(chan structs.HallOrder)
	outgoingElevStateChan := make(chan structs.HRAElevState)
	completedRequetsChan := make(chan []elevio.ButtonEvent)

	// Start the FSM using fsmButtons.
	go fsm.Fsm(fsmButtons, drv_floors, drv_obstr, drv_stop, elevatorCh)

	// Start the local state machine using localStateButtons.
	go localOrders.LocalStateManager(
		localStateButtons,
		elevatorCh,
		outgoingOrdersChan,
		outgoingElevStateChan,
		completedRequetsChan,
	)

	// Drain the outgoing channels if you don't need their data.
	go func() {
		for range outgoingOrdersChan {
			// Optionally log or discard.
		}
	}()
	go func() {
		for range outgoingElevStateChan {
			// Optionally log or discard.
		}
	}()
	go func() {
		for range completedRequetsChan {
			// Optionally log or discard.
		}
	}()

	select {}
}
