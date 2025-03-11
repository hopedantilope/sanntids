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

// Tests the transformation from ElevatorData format to HRA format
func testTransformationToHRA() bool {
	// Create sample elevator data
	var hallOrders []structs.HallOrder
	hallOrders = append(hallOrders, structs.HallOrder{
		Floor:       1,
		Dir:         elevio.BT_HallDown,
		Status:      structs.Assigned,
		DelegatedID: localElevID,
	})
	hallOrders = append(hallOrders, structs.HallOrder{
		Floor:       3,
		Dir:         elevio.BT_HallUp,
		Status:      structs.Assigned,
		DelegatedID: otherElevID1,
	})
	
	elevatorData := structs.ElevatorDataWithID{
		ElevatorID:    localElevID,
		HallOrders:    hallOrders,
		ElevatorState: testElevatorStates,
	}
	
	// Transform to HRA format
	states, hallRequests := runHRA.TransformToHRA(elevatorData)
	
	// Check if all elevator states are preserved
	if len(states) != len(testElevatorStates) {
		fmt.Println("FAIL: TransformToHRA - Wrong number of elevator states")
		return false
	}
	
	// Check if hall requests are correctly transformed
	// Floor 1, down direction should be true
	if !hallRequests[1][0] {
		fmt.Println("FAIL: TransformToHRA - Order at floor 1, down direction not set")
		return false
	}
	
	// Floor 3, up direction should be true
	if !hallRequests[3][1] {
		fmt.Println("FAIL: TransformToHRA - Order at floor 3, up direction not set")
		return false
	}
	
	return true
}

// Tests the transformation from HRA format back to ElevatorData format
func testTransformationFromHRA() bool {
	// Transform from HRA format to ElevatorData
	result := runHRA.TransformFromHRA(testHallRequestAssignments, testElevatorStates, localElevID)
	
	// Check if ElevatorID is set correctly
	if result.ElevatorID != localElevID {
		fmt.Printf("FAIL: TransformFromHRA - Expected elevator ID %s, got %s\n", 
			localElevID, result.ElevatorID)
		return false
	}
	
	// Check if hall orders are correctly transformed
	// We should have orders for all active requests in testHallRequestAssignments
	expectedOrderCount := 0
	for _, floors := range testHallRequestAssignments {
		for _, directions := range floors {
			for _, isActive := range directions {
				if isActive {
					expectedOrderCount++
				}
			}
		}
	}
	
	if len(result.HallOrders) != expectedOrderCount {
		fmt.Printf("FAIL: TransformFromHRA - Expected %d hall orders, got %d\n", 
			expectedOrderCount, len(result.HallOrders))
		return false
	}
	
	// Check if elevator states are preserved
	if len(result.ElevatorState) != len(testElevatorStates) {
		fmt.Println("FAIL: TransformFromHRA - Wrong number of elevator states")
		return false
	}
	
	return true
}

// Tests the complete HRA flow
func testCompleteHRAFlow() bool {
	// Create elevator data with test values
	var hallOrders []structs.HallOrder
	hallOrders = append(hallOrders, structs.HallOrder{
		Floor:       1,
		Dir:         elevio.BT_HallDown,
		Status:      structs.Assigned,
		DelegatedID: localElevID,
	})
	hallOrders = append(hallOrders, structs.HallOrder{
		Floor:       3,
		Dir:         elevio.BT_HallUp,
		Status:      structs.Assigned,
		DelegatedID: otherElevID1,
	})
	
	initialData := structs.ElevatorDataWithID{
		ElevatorID:    localElevID,
		HallOrders:    hallOrders,
		ElevatorState: testElevatorStates,
	}
	
	// This test won't actually run the external HRA executable
	// but we can test that our transformations work correctly
	states, _ := runHRA.TransformToHRA(initialData)
	result := runHRA.TransformFromHRA(testHallRequestAssignments, states, localElevID)
	
	// Check that we got a valid result
	if result.ElevatorID != localElevID {
		fmt.Println("FAIL: Complete HRA flow - Invalid elevator ID in result")
		return false
	}
	
	if len(result.ElevatorState) != len(testElevatorStates) {
		fmt.Println("FAIL: Complete HRA flow - Wrong number of elevator states")
		return false
	}
	
	// Check orders were converted correctly
	expectedOrderCount := 0
	for _, floors := range testHallRequestAssignments {
		for _, directions := range floors {
			for _, isActive := range directions {
				if isActive {
					expectedOrderCount++
				}
			}
		}
	}
	
	if len(result.HallOrders) != expectedOrderCount {
		fmt.Printf("FAIL: Complete HRA flow - Expected %d hall orders, got %d\n", 
			expectedOrderCount, len(result.HallOrders))
		return false
	}
	
	return true
}

// Runs all integration tests
func runIntegrationTests() {
	fmt.Println("Running integration tests...")
	
	// Test transformations
	if testTransformationToHRA() {
		fmt.Println("PASS: Transformation to HRA format")
	}
	
	if testTransformationFromHRA() {
		fmt.Println("PASS: Transformation from HRA format")
	}
	
	if testCompleteHRAFlow() {
		fmt.Println("PASS: Complete HRA flow")
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