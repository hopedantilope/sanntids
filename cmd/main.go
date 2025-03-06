package main

import (
	"Driver-go/elevio"
	"flag"
	"fmt"
	"os"
	"sanntids/cmd/localElevator/fsm"
	"sanntids/cmd/localOrders"
	"sanntids/cmd/networkOrders"
	"sanntids/cmd/network/broadcastState"
	"sanntids/cmd/elevatorController/runHRA"
	"sanntids/cmd/orderTracker" 
	"time"
	"Network-go/network/localip"
)

func main() {
	// Parse command-line arguments
	port := flag.String("port", "15657", "Port number for elevator simulator")
	elevatorID := flag.String("id", "", "Elevator ID (defaults to local IP if not specified)")
	broadcastPortFlag := flag.Int("broadcast", 30003, "Port for broadcasting state")
	numFloorsFlag := flag.Int("floors", 4, "Number of floors")
	flag.Parse()

	// Configure the simulator
	numFloors := *numFloorsFlag
	elevPort := fmt.Sprintf("localhost:%s", *port)

	runHRA.Test()

	elevio.Init(elevPort, numFloors)
	
	// Use provided ID or default to local IP
	var id string
	if *elevatorID != "" {
		id = *elevatorID
	} else {
		var err error
		id, err = localip.LocalIP()
		if err != nil {
			fmt.Println("Failed to get local IP:", err)
			os.Exit(1)
		}
	}
	
	fmt.Println("Elevator started with id:", id)
	fmt.Println("Connected to elevator on port:", *port)

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
	
	// Channel for completed orders
	completedOrderChan := make(chan localOrders.HallOrder)

	// not sending buttons yet as this is going through statemachine
	go fsm.Fsm(nil, drv_floors, drv_obstr, drv_stop, stateRequestTx)

	// Setup network communication
	broadcastPort := *broadcastPortFlag
	broadcastDataChan := make(chan broadcastState.ElevatorDataWithID)
	receiveDataChan := make(chan broadcastState.ElevatorDataWithID)
	
	// Internal channels for order managers
	incomingOrdersChan := make(chan []localOrders.HallOrder)
	outgoingOrdersChan := make(chan []localOrders.HallOrder)
	orderAssignmentChan := make(chan localOrders.HallOrder)
	
	// Create shared order maps
	localOrderMap := make(localOrders.OrderMap)
	networkOrderMap := make(localOrders.OrderMap)

	// Create the order tracker
	elevOrderMap := orderTracker.NewElevatorOrderMap()
	systemState := orderTracker.NewSystemState(elevOrderMap)

	// Start order managers - pass drv_buttons directly
	go localOrders.HallOrderManager(
		incomingOrdersChan,
		drv_buttons,
		outgoingOrdersChan,
		completedOrderChan)
	
	go networkOrders.NetworkOrderManager(
		id, 
		incomingOrdersChan, 
		&localOrderMap, 
		orderAssignmentChan)

	// Start network communication
	go broadcastState.BroadcastState(broadcastDataChan, broadcastPort)
	go broadcastState.ReceiveState(receiveDataChan, broadcastPort)

	// Setup periodic broadcasting
	broadcastTicker := time.NewTicker(1 * time.Second)
	defer broadcastTicker.Stop()

	// Setup periodic system state printing
	statePrintTicker := time.NewTicker(5 * time.Second)
	defer statePrintTicker.Stop()

	// Track local hall orders
	var localHallOrders []localOrders.HallOrder

	for {
		select {
		case <-broadcastTicker.C:
			// Get current elevator state
			requestChan := make(chan fsm.ElevatorState)
			stateRequestTx <- requestChan
			currentState := <-requestChan
			
			// Update our local system state
			systemState.UpdateElevatorState(id, currentState)
			
			// Also make sure local orders are in the order tracker
			elevOrderMap.UpdateOrders(id, localHallOrders)
			
			// Create combined data packet with state and orders
			dataWithID := broadcastState.ElevatorDataWithID{
				ElevatorID:    id,
				ElevatorState: currentState,
				HallOrders:    localHallOrders,
			}
			
			// Broadcast - the BroadcastState function will handle chunking
			broadcastDataChan <- dataWithID

		case <-statePrintTicker.C:
			// Print the complete system state periodically
			fmt.Println("\n======= PERIODIC SYSTEM STATUS =======")
			systemState.PrintSystemState()
			fmt.Println("======================================\n")

		case receivedDataWithID := <-receiveDataChan:
			if receivedDataWithID.ElevatorID != id {
				// Update the order tracker with received data
				update := struct {
					ElevatorID string
					State      fsm.ElevatorState
					Orders     []localOrders.HallOrder
				}{
					ElevatorID: receivedDataWithID.ElevatorID,
					State:      receivedDataWithID.ElevatorState,
					Orders:     receivedDataWithID.HallOrders,
				}
				
				// Process update through order tracker
				systemState.ProcessUpdate(update)
				
				// Process their hall orders through the network order manager
				incomingOrdersChan <- receivedDataWithID.HallOrders
				
				fmt.Printf("Received update from elevator %s with %d orders\n", 
					receivedDataWithID.ElevatorID, len(receivedDataWithID.HallOrders))
			}
			
		case updatedOrders := <-outgoingOrdersChan:
			// Store orders for broadcasting
			localHallOrders = updatedOrders
			
			// Update the order tracker with our own orders
			elevOrderMap.UpdateOrders(id, localHallOrders)
			
			// Update local order map
			for _, order := range updatedOrders {
				localOrderMap[order.OrderID] = order
			}
			
			// Merge local and network orders for HRA
			mergedOrderMap := orderTracker.GetMergedOrderMap(localOrderMap, networkOrderMap)
			elevatorStates := systemState.GetElevatorStates()
			hraInput := broadcastState.ConvertToHRAInput(elevatorStates, mergedOrderMap)
			fmt.Printf("Generated HRA input with %d elevator states and hall requests\n", 
			len(hraInput.States))
			
			// Print updated system state
			fmt.Println("\n======= UPDATED SYSTEM STATUS =======")
			systemState.PrintSystemState()
			fmt.Println("======================================\n")
		}
	}
}