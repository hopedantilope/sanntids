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
	"sanntids/cmd/runHRA"
	"sanntids/cmd/localOrders"
)

// Test data structures defined at package level for reuse
var (
	// Use IP addresses as elevator IDs
	localElevID  = "127.0.0.1"
	otherElevID1 = "127.0.0.2"
	otherElevID2 = "127.0.0.3"
	
	// Hall request assignments for test
	testHallRequestAssignments = map[string][config.N_FLOORS][2]bool{
		localElevID:  {{false, false}, {true, false}, {false, false}, {false, false}},
		otherElevID1: {{false, false}, {false, false}, {false, false}, {false, true}},
		otherElevID2: {{false, true}, {false, false}, {true, false}, {false, false}},
	}
	
	// Elevator states for test
	testElevatorStates = map[string]structs.HRAElevState{
		localElevID: {
			Behavior:    "moving",
			Floor:       2,
			Direction:   "up",
			CabRequests: structs.CabRequestList{false, false, false, true},
		},
		otherElevID1: {
			Behavior:    "idle",
			Floor:       0,
			Direction:   "stop",
			CabRequests: structs.CabRequestList{false, false, false, false},
		},
		otherElevID2: {
			Behavior:    "idle",
			Floor:       0,
			Direction:   "stop",
			CabRequests: structs.CabRequestList{false, false, false, false},
		},
	}
	
	// Sample hall requests for HRA testing
	testHallRequests = [config.N_FLOORS][2]bool{
		{false, false}, 
		{true, false}, 
		{false, false}, 
		{false, true},
	}
)

// Tests the transformation from HRA format to internal ElevatorData format
func testTransformationToElevatorData() bool {
	// Transform data to elevator format
	result := runHRA.TransformToElevatorState(testHallRequestAssignments, testElevatorStates, )
	
	// Check if ElevatorID is set - this should be the ID of the local elevator
	if result.ElevatorID == "" {
		fmt.Println("FAIL: TransformToElevatorData - Missing elevator ID in result")
		return false
	}
	
	// Check if hall orders are correctly transformed
	// We expect only orders for the local elevator ID to be included
	for _, order := range result.HallOrders {
		if order.DelegatedID != result.ElevatorID {
			fmt.Printf("FAIL: TransformToElevatorData - Order delegated to %s but local elevator is %s\n", 
				order.DelegatedID, result.ElevatorID)
			return false
		}
	}
	
	// Check if elevator states for all elevators are present
	if len(result.ElevatorState) != 3 {
		fmt.Println("FAIL: TransformToElevatorData - Expected 3 elevator states, got", len(result.ElevatorState))
		return false
	}
	
	return true
}

// Tests the transformation from internal ElevatorData format back to HRA format
func testTransformationFromElevatorData() bool {
	// Create elevator data with test values for local elevator
	var hallOrders []structs.HallOrder
	hallOrders = append(hallOrders, structs.HallOrder{
		Floor:       1,
		Dir:         elevio.BT_HallDown,
		Status:      structs.Assigned,
		DelegatedID: localElevID,
	})
	
	elevatorData := structs.ElevatorDataWithID{
		ElevatorID:    localElevID,
		HallOrders:    hallOrders,
		ElevatorState: testElevatorStates,
	}
	
	// Transform back
	reconvertedAssignments, reconvertedStates := runHRA.TransformFromElevatorState(elevatorData)
	
	// Check if the local elevator ID exists in reconverted assignments
	if _, exists := reconvertedAssignments[localElevID]; !exists {
		fmt.Println("FAIL: TransformFromElevatorData - Missing local elevator ID in assignments")
		return false
	}
	
	// Verify the order at floor 1, direction down is correctly set for local elevator
	if !reconvertedAssignments[localElevID][1][0] {
		fmt.Println("FAIL: TransformFromElevatorData - Order at floor 1, down direction not set for local elevator")
		return false
	}
	
	// Validate states
	if len(reconvertedStates) != len(testElevatorStates) {
		fmt.Println("FAIL: TransformFromElevatorData - Wrong number of elevator states")
		return false
	}
	
	// Check that each elevator state from input is present in output
	for id := range testElevatorStates {
		if _, exists := reconvertedStates[id]; !exists {
			fmt.Printf("FAIL: TransformFromElevatorData - Missing elevator state for %s\n", id)
			return false
		}
	}
	
	return true
}

// Runs all integration tests
func runIntegrationTests() {
	fmt.Println("Running integration tests...")
	
	// Test transformations
	if testTransformationToElevatorData() {
		fmt.Println("PASS: Transformation to ElevatorData")
	}
	
	if testTransformationFromElevatorData() {
		fmt.Println("PASS: Transformation from ElevatorData")
	}
	
	fmt.Println("Integration tests completed")
}

func main() {
	// Parse command-line arguments
	port := flag.String("port", "15657", "Port number for elevator simulator")
	elevatorID := flag.String("id", "", "Elevator ID (defaults to local IP if not specified)")
	broadcastPortFlag := flag.Int("broadcast", 30003, "Port for broadcasting state")
	numFloorsFlag := flag.Int("floors", config.N_FLOORS, "Number of floors")
	flag.Parse()

	// Run integration tests
	runIntegrationTests()

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