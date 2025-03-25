package networkOrders

import (
	"Driver-go/elevio"
	"fmt"
	"sanntids/cmd/localElevator/config"
	"sanntids/cmd/localElevator/elevator"
	"sanntids/cmd/localElevator/structs"
	"sanntids/cmd/runHRA"
	"sanntids/cmd/util"
	"time"
)

// NetworkOrderManager handles the conversion of local orders to network-ready format
// and manages incoming orders from other elevators
func NetworkOrderManager(
	localElevatorID string,
	localElevStateChan <-chan structs.HRAElevState,
	localOrdersChan <-chan structs.HallOrder,
	completedRequetsChan <-chan []elevio.ButtonEvent,
	incomingDataChan <-chan structs.ElevatorDataWithID,
	outgoingDataChan chan<- structs.ElevatorDataWithID,
    requestsToLocalChan chan<- [config.N_FLOORS][config.N_BUTTONS]bool,
) {
	// Initialize data stores
	elevatorStates := make(map[string]structs.HRAElevState)
	hallOrders := make([]structs.HallOrder, 0)
	hallOrdersMap := make(map[string][]structs.HallOrder, 0)
	ipMap := make(map[string]time.Time, 0)
    var prevRequests [config.N_FLOORS][config.N_BUTTONS]bool
	// Create a ticker that periodically sends network data
	transmitTicker := time.NewTicker(config.TransmitTickerMs * time.Millisecond)
	defer transmitTicker.Stop()

	for {
		select {
		// Handle periodic data transmission
		case <-transmitTicker.C:
			currentTime := time.Now()
			for ip, lastSeen := range ipMap {
				// If we haven't received an update in 2 seconds, remove the IP
				if currentTime.Sub(lastSeen) > config.ElevatorTimeoutMs*time.Millisecond {
					delete(ipMap, ip)
					delete(elevatorStates, ip)
					delete(hallOrdersMap, ip)
				}
			}
			if util.IsMaster(ipMap, localElevatorID) {
				hallOrders = applyNewOrderBarrier(hallOrders, hallOrdersMap, ipMap)
			}
			sendNetworkData(localElevatorID, elevatorStates, hallOrders, outgoingDataChan, ipMap)

		// Process incoming data from other elevators
		case incomingData, ok := <-incomingDataChan:
			if !ok {
				return
			}

            //Change this:
			ipMap[incomingData.ElevatorID] = time.Now()
			hallOrdersMap[incomingData.ElevatorID] = incomingData.HallOrders

			ipList := make([]string, 0, len(ipMap))

			for key := range ipMap {
				ipList = append(ipList, key)
			}

			// Update state map with received data
			for id, state := range incomingData.ElevatorState {
				if incomingData.ElevatorID == id {
					elevatorStates[id] = state
				}
			}
			// Process incoming hall orders
			for _, newOrder := range incomingData.HallOrders {
				if !isDuplicateOrder(hallOrders, newOrder) {
					hallOrders = append(hallOrders, newOrder)
				} else {
					// Update existing order if necessary
					for i, order := range hallOrders {
						if order.Floor == newOrder.Floor && order.Dir == newOrder.Dir {
							// Accept everthing the master says:
							if util.IsLowestIP(ipList, incomingData.ElevatorID) {
								//Send FSM
								if order.Status == structs.New && newOrder.Status == structs.Completed {
									continue
								}
								hallOrders[i].Status = newOrder.Status
								hallOrders[i].DelegatedID = newOrder.DelegatedID
							}
							// The master should only accept certain orders:
							if util.IsLowestIP(ipList, localElevatorID) {
								switch newOrder.Status {
								case structs.New:
									if order.Status == structs.Completed{
										hallOrders[i].Status = newOrder.Status
										hallOrders[i].DelegatedID = newOrder.DelegatedID
									}
								case structs.Assigned:
									//Do nothing
								case structs.Confirmed:
									//Do nothing
								case structs.Completed:
									if order.Status != structs.New && order.DelegatedID == incomingData.ElevatorID{
										hallOrders[i].Status = newOrder.Status
										hallOrders[i].DelegatedID = newOrder.DelegatedID
									}
								}
							}
							break
						}
					}
				}
			}

			//Get the requests assigned to localID and send them to Elevator
			myRequests := getMyRequests(hallOrders, elevatorStates, localElevatorID)
			if myRequests != prevRequests {
				requestsToLocalChan <- myRequests
				prevRequests = myRequests
			}

		// Update local elevator state
		case localState, ok := <-localElevStateChan:
			if !ok {
				return
			}

			elevatorStates[localElevatorID] = localState

		// Process local orders
		case localOrder, ok := <-localOrdersChan:
			if !ok {
				return
			}

			if !isDuplicateOrder(hallOrders, localOrder) {
				hallOrders = append(hallOrders, localOrder)
			} else {
				// Update the existing order status if it already exists
				for i := range hallOrders {
					if hallOrders[i].Floor == localOrder.Floor && hallOrders[i].Dir == localOrder.Dir {
						hallOrders[i].Status = localOrder.Status
						hallOrders[i].DelegatedID = localOrder.DelegatedID
						break
					}
				}
			}

		// Process completed requests
		case completedReqs, ok := <-completedRequetsChan:
			if !ok {
				return
			}

			// Update order status for completed requests
			for _, req := range completedReqs {
				hallOrders = UpdateOrderStatus(hallOrders, req.Floor, int(req.Button), structs.Completed)
			}
		}
	}
}

// Two orders are considered duplicates if they have the same floor and direction
func isDuplicateOrder(orders []structs.HallOrder, newOrder structs.HallOrder) bool {
	for _, order := range orders {
		if order.Floor == newOrder.Floor && order.Dir == newOrder.Dir {
			return true
		}
	}
	return false
}

func sendNetworkData(
	localID string,
	states map[string]structs.HRAElevState,
	orders []structs.HallOrder,
	outChan chan<- structs.ElevatorDataWithID,
	ipMap map[string]time.Time,
) {
	fmt.Printf("Sending network data from elevator ID: %s\n", localID)

	// Copy states map
	statesCopy := make(map[string]structs.HRAElevState)
	for id, state := range states {
		statesCopy[id] = state
		fmt.Printf("  State for elevator %s: Floor=%d, Direction=%s, Behavior=%s\n",
			id, state.Floor, state.Direction, state.Behavior)

		// Print cab requests
		fmt.Print("    Cab Requests: [")
		for floor, hasRequest := range state.CabRequests {
			if hasRequest {
				fmt.Printf("%d ", floor)
			}
		}
		fmt.Println("]")
	}

	// Copy orders slice
	ordersCopy := make([]structs.HallOrder, len(orders))
	copy(ordersCopy, orders)

	// Log hall orders
	fmt.Printf("  Sending %d hall orders:\n", len(ordersCopy))
	for i, order := range ordersCopy {
		status := "Unknown"
		switch order.Status {
		case structs.New:
			status = "New"
		case structs.Assigned:
			status = "Assigned"
		case structs.Confirmed:
			status = "Confirmed"
		case structs.Completed:
			status = "Completed"
		}

		fmt.Printf("    Order %d: Floor=%d, Direction=%v, Status=%s, DelegatedTo=%s\n",
			i, order.Floor, order.Dir, status, order.DelegatedID)
	}

	networkData := structs.ElevatorDataWithID{
		ElevatorID:    localID,
		ElevatorState: statesCopy,
		HallOrders:    ordersCopy,
	}

	if util.IsMaster(ipMap, localID) {
		networkData = assignOrders(networkData)
	}
	setAllLights(networkData)
	// Use non-blocking send to avoid dealocks
	select {
	case outChan <- networkData:
		fmt.Println("  Successfully sent network data")
	default:
		fmt.Println("  Failed to send network data: channel full or not ready")
	}
}

// UpdateOrderStatus updates the status of a hall order in the order list
func UpdateOrderStatus(orders []structs.HallOrder, floor int, dir int, newStatus structs.OrderStatus) []structs.HallOrder {
	for i, order := range orders {
		if order.Floor == floor && int(order.Dir) == dir {
			orders[i].Status = newStatus
			break
		}
	}
	return orders
}

// assignOrders checks if the local elevator is the master and
// assigns orders accordingly using the runHRA package.
func assignOrders(data structs.ElevatorDataWithID) structs.ElevatorDataWithID {
    // Split orders into pending and completed.
    var pendingOrders []structs.HallOrder
    var nonPendingOrders []structs.HallOrder
    for _, order := range data.HallOrders {
        if order.Status == structs.Completed || order.Status == structs.New{
            nonPendingOrders = append(nonPendingOrders, order)
        } else {
            pendingOrders = append(pendingOrders, order)
        }
    }

	newElevState := make(map[string]structs.HRAElevState)
    for key, state := range data.ElevatorState {
        if !state.Obstruction {
            newElevState[key] = state
        }
    }

    // Create a new data object with only pending orders for runHRA.
    dataForHRA := data
	dataForHRA.ElevatorState = newElevState
    dataForHRA.HallOrders = pendingOrders

    // Run HRA on the pending orders.
    newData := runHRA.RunHRA(dataForHRA)
    // Merge the completed orders back into the HRA result.
    newData.HallOrders = append(newData.HallOrders, nonPendingOrders...)
    return newData
}


// orderKnownByAll returns true if every active node (from ipList)
// has an order with the same Floor and Dir that is still marked as New (or already Confirmed)
func orderKnownByAll(order structs.HallOrder, hallOrdersMap map[string][]structs.HallOrder, ipList []string) bool {
    for _, nodeID := range ipList {
        orders, exists := hallOrdersMap[nodeID]
        if !exists {
            return false
        }
        found := false
        for _, o := range orders {
            if o.Floor == order.Floor && o.Dir == order.Dir &&
                (o.Status == structs.New || o.Status == structs.Confirmed) {
                found = true
                break
            }
        }
        if !found {
            return false
        }
    }
    return true
}

func applyNewOrderBarrier(orders []structs.HallOrder, hallOrdersMap map[string][]structs.HallOrder, ipMap map[string]time.Time) []structs.HallOrder {
    // Build a list of active node IDs from ipMap
    ipList := make([]string, 0, len(ipMap))
    for nodeID := range ipMap {
        ipList = append(ipList, nodeID)
    }
    // Check each new order
    for i, order := range orders {
        if order.Status == structs.New {
            if orderKnownByAll(order, hallOrdersMap, ipList) {
                // All nodes agree the order exists – barrier passed.
                orders[i].Status = structs.Confirmed
            }
        }
    }
    return orders
}


func getMyRequests(hallOrders []structs.HallOrder, elevatorStates map[string]structs.HRAElevState, myID string) [config.N_FLOORS][config.N_BUTTONS]bool {
    var orders [config.N_FLOORS][config.N_BUTTONS]bool

    for _, order := range hallOrders {
        if order.DelegatedID == myID && order.Status == structs.Assigned {
            orders[order.Floor][order.Dir] = true
        }
    }

    if state, ok := elevatorStates[myID]; ok {
		for floorIndex := 0; floorIndex < len(state.CabRequests); floorIndex++ {
			if state.CabRequests[floorIndex] {
				orders[floorIndex][elevator.BT_Cab] = true
			}
		}
    }

    return orders
}


func setAllLights(data structs.ElevatorDataWithID) {
	var hallLightsOn [config.N_FLOORS][2]bool

	for _, order := range data.HallOrders {
		buttonType := int(order.Dir)
		if buttonType == int(elevio.BT_HallUp) || buttonType == int(elevio.BT_HallDown) {
			if order.Status == structs.Confirmed || order.Status == structs.Assigned {
				hallLightsOn[order.Floor][buttonType] = true
			}
		}
	}

	for floor := 0; floor < config.N_FLOORS; floor++ {
		elevio.SetButtonLamp(elevio.BT_HallUp, floor, hallLightsOn[floor][int(elevio.BT_HallUp)])
		elevio.SetButtonLamp(elevio.BT_HallDown, floor, hallLightsOn[floor][int(elevio.BT_HallDown)])
	}
}