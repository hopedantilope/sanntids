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
	ipMap := make(map[string]time.Time, 0)
    var prevRequests [config.N_FLOORS][config.N_BUTTONS]bool
	// Create a ticker that periodically sends network data
	transmitTicker := time.NewTicker(500 * time.Millisecond)
	defer transmitTicker.Stop()

	for {
		select {
		// Handle periodic data transmission
		case <-transmitTicker.C:
			sendNetworkData(localElevatorID, elevatorStates, hallOrders, outgoingDataChan, ipMap)

		// Process incoming data from other elevators
		case incomingData, ok := <-incomingDataChan:
			if !ok {
				return
			}

            //Change this:
			ipMap[incomingData.ElevatorID] = time.Now()
			ipList := make([]string, 0, len(ipMap))

			for key := range ipMap {
				ipList = append(ipList, key)
			}
			//

			// Skip processing our own messages
			if incomingData.ElevatorID == localElevatorID {
				continue
			}

			// Update state map with received data
			for id, state := range incomingData.ElevatorState {
				if incomingData.ElevatorID == id {
					elevatorStates[id] = state
				}
			}

            //Get the requests assigned to localID and send them to Elevator
            myRequests := getMyRequests(incomingData, localElevatorID)
            if myRequests != prevRequests {
                requestsToLocalChan <- myRequests
                prevRequests = myRequests
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
								hallOrders[i].Status = newOrder.Status
								hallOrders[i].DelegatedID = newOrder.DelegatedID
							}
							// The master should only accept certain orders:
							if util.IsLowestIP(ipList, localElevatorID) {
								switch newOrder.Status {
								case structs.New:
									if order.Status == structs.Completed {
										hallOrders[i].Status = newOrder.Status
										hallOrders[i].DelegatedID = newOrder.DelegatedID
									}
								case structs.Assigned:
									//Maybe do something like this?
									//Add id to list of elevetors that knows about the order
									//If every alive elevator knows about the order, set status to confirmed
									//Turn on lights

								case structs.Confirmed:
									//Do nothing
								case structs.Completed:
									if order.Status != structs.New {
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

			// Remove completed orders after a reasonable delay
			// Consider implementing a cleanup mechanism here or periodically
			hallOrders = RemoveCompletedOrders(hallOrders)
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

	//The master assigns orders here
	networkData = assignOrders(networkData, ipMap, localID)

	// Use non-blocking send to avoid deadlocks
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

// RemoveCompletedOrders removes orders with Completed status from the order list
func RemoveCompletedOrders(orders []structs.HallOrder) []structs.HallOrder {
	result := make([]structs.HallOrder, 0)
	removed := 0

	for _, order := range orders {
		if order.Status != structs.Completed {
			result = append(result, order)
		} else {
			removed++
		}
	}

	if removed > 0 {
		fmt.Printf("Removed %d completed orders\n", removed)
	}

	return result
}

// assignOrders checks if the local elevator is the master and
// assigns orders accordingly using the runHRA package.
func assignOrders(data structs.ElevatorDataWithID, ipMap map[string]time.Time, localID string) structs.ElevatorDataWithID {
	ipList := make([]string, 0, len(ipMap))
	for key := range ipMap {
		ipList = append(ipList, key)
	}
	if util.IsLowestIP(ipList, localID) {
		data = runHRA.RunHRA(data)
	}
	return data
}


func getMyRequests(elevatorData structs.ElevatorDataWithID, myID string) [config.N_FLOORS][config.N_BUTTONS]bool {
    var orders [config.N_FLOORS][config.N_BUTTONS]bool

    for _, order := range elevatorData.HallOrders {
        if order.DelegatedID == myID && order.Status == structs.Confirmed {
            orders[order.Floor][order.Dir] = true
        }
    }

    if state, ok := elevatorData.ElevatorState[myID]; ok {
        for floor, requested := range state.CabRequests {
            if requested {
                orders[floor][elevator.BT_Cab] = true
            }
        }
    }

    return orders
}

