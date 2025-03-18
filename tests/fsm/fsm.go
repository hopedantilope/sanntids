package main

import (
	"Driver-go/elevio"
	"flag"
	"fmt"
	"sanntids/cmd/localElevator/config"
	"sanntids/cmd/localElevator/elevator"
	"sanntids/cmd/localElevator/fsm"
	"time"
)

// buttonRequestsHandler accumulates button events into a 2D array and periodically
// sends the current snapshot on the requestsToLocalChan.
func buttonRequestsHandler(
	drv_buttons <-chan elevio.ButtonEvent,
	requestsToLocalChan chan<- [config.N_FLOORS][config.N_BUTTONS]bool,
	eCh <-chan elevator.Elevator,
) {
	var localRequests [config.N_FLOORS][config.N_BUTTONS]bool
	var prevRequests [config.N_FLOORS][config.N_BUTTONS]bool // holds last sent snapshot
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case btn := <-drv_buttons:
			// Mark the button as pressed in the snapshot.
			localRequests[btn.Floor][int(btn.Button)] = true

		case elev := <-eCh:
			// Clear only those orders that are marked as cleared in elevator.Cleared.
			fmt.Println("Received elevator update, Cleared:", elev.Cleared)
			for floor := 0; floor < config.N_FLOORS; floor++ {
				for btn := 0; btn < config.N_BUTTONS; btn++ {
					if elev.Cleared[floor][btn] {
						localRequests[floor][btn] = false
						fmt.Println("Cleared:", floor, btn)
					}
				}
			}

		case <-ticker.C:
			// Only send the snapshot if it's different from the previous one.
			if localRequests != prevRequests {
				requestsToLocalChan <- localRequests
				fmt.Println("Sending new localRequests snapshot:", localRequests)
				prevRequests = localRequests
			}
		}
	}
}



func main() {
	// Parse command-line arguments.
	port := flag.String("port", "15657", "Port number for elevator simulator")
	numFloorsFlag := flag.Int("floors", config.N_FLOORS, "Number of floors")
	flag.Parse()

	// Configure the simulator.
	numFloors := *numFloorsFlag
	elevPort := fmt.Sprintf("localhost:%s", *port)

	// Initialize the elevator driver.
	elevio.Init(elevPort, numFloors)

	// Create channels for driver inputs.
	drv_buttons := make(chan elevio.ButtonEvent)
	drv_floors := make(chan int)
	drv_obstr := make(chan bool)
	drv_stop := make(chan bool)

	// Start polling inputs concurrently.
	go elevio.PollButtons(drv_buttons)
	go elevio.PollFloorSensor(drv_floors)
	go elevio.PollObstructionSwitch(drv_obstr)
	go elevio.PollStopButton(drv_stop)

	// Create channels for the elevator FSM and orders.
	elevatorCh := make(chan elevator.Elevator)
	// Channel carrying the complete 2D array of button requests.
	requestsToLocalChan := make(chan [config.N_FLOORS][config.N_BUTTONS]bool)

	// Start the button request aggregator.
	go buttonRequestsHandler(drv_buttons, requestsToLocalChan, elevatorCh)
	// Start the FSM with the new requests channel.
	go fsm.Fsm(requestsToLocalChan, drv_floors, drv_obstr, drv_stop, elevatorCh)

	// Block forever.
	select {}
}
