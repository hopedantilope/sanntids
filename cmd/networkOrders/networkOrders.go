package networkOrders

import (
	"Driver-go/elevio"
	"fmt"
	"sanntids/cmd/localOrders"
)

// NetworkOrderManager handles orders received from the network
func NetworkOrderManager(
	localID string,
	netUpdates <-chan []localOrders.HallOrder,
	localOrderMap *localOrders.OrderMap,
	orderAssignmentChan chan<- localOrders.HallOrder) {
	
	// Network order map keeps track of all orders from other elevators
	networkOrders := make(localOrders.OrderMap)
	
	for {
		select {
		case updateList := <-netUpdates:
			// Process orders from other elevators
			for _, update := range updateList {
				onNetworkUpdate(localID, &networkOrders, localOrderMap, update, orderAssignmentChan)
			}
		}
	}
}

// shouldAcceptNetworkUpdate determines if the incoming update should override the local state.
func shouldAcceptNetworkUpdate(local localOrders.HallOrder, incoming localOrders.HallOrder) bool {
	switch {
	case local.Status == localOrders.Unknown && incoming.Status == localOrders.Completed:
		return false
	case local.Status == localOrders.Completed && incoming.Status == localOrders.Unknown:
		return true
	case incoming.Status > local.Status:
		return true
	default:
		return false
	}
}

func onNetworkUpdate(
	localID string,
	networkOrders *localOrders.OrderMap,
	localOrderMap *localOrders.OrderMap,
	update localOrders.HallOrder,
	orderAssignmentChan chan<- localOrders.HallOrder) {
	
	// First check if this order is in our local order map
	if localOrder, existsLocal := (*localOrderMap)[update.OrderID]; existsLocal {
		// Handle updates for our locally initiated orders
		if update.Status == localOrders.Assigned && update.DelegatedID != "" {
			// The order has been assigned by the system
			orderAssignmentChan <- update
			
			// Only set light if this order is assigned to us
			if update.DelegatedID == localID {
				var btnType elevio.ButtonType
				if update.Dir == elevio.MD_Up {
					btnType = elevio.BT_HallUp
				} else {
					btnType = elevio.BT_HallDown
				}
				
				elevio.SetButtonLamp(btnType, update.Floor, true)
				fmt.Printf("Setting light ON for assigned order: Floor %d, Direction %v\n", 
					update.Floor, update.Dir)
			}
		} else if shouldAcceptNetworkUpdate(localOrder, update) {
			// Update our local record
			(*localOrderMap)[update.OrderID] = update
			
			// If completed and was assigned to us, turn off the light
			if update.Status == localOrders.Completed && update.DelegatedID == localID {
				var btnType elevio.ButtonType
				if update.Dir == elevio.MD_Up {
					btnType = elevio.BT_HallUp
				} else {
					btnType = elevio.BT_HallDown
				}
				
				elevio.SetButtonLamp(btnType, update.Floor, false)
				fmt.Printf("Setting light OFF for completed order: Floor %d, Direction %v\n", 
					update.Floor, update.Dir)
			}
		}
	} else {
		// Check if we already have this network order
		if existingOrder, exists := (*networkOrders)[update.OrderID]; exists {
			// Only update if the network version has higher priority
			if shouldAcceptNetworkUpdate(existingOrder, update) {
				(*networkOrders)[update.OrderID] = update
				
				// Handle light status based on assignment and completion
				if update.Status == localOrders.Assigned && update.DelegatedID == localID {
					// Assigned to us, turn on light
					var btnType elevio.ButtonType
					if update.Dir == elevio.MD_Up {
						btnType = elevio.BT_HallUp
					} else {
						btnType = elevio.BT_HallDown
					}
					
					elevio.SetButtonLamp(btnType, update.Floor, true)
					fmt.Printf("Setting light ON for newly assigned order: Floor %d, Direction %v\n", 
						update.Floor, update.Dir)
				} else if update.Status == localOrders.Completed && update.DelegatedID == localID {
					// Completed and was assigned to us, turn off light
					var btnType elevio.ButtonType
					if update.Dir == elevio.MD_Up {
						btnType = elevio.BT_HallUp
					} else {
						btnType = elevio.BT_HallDown
					}
					
					elevio.SetButtonLamp(btnType, update.Floor, false)
					fmt.Printf("Setting light OFF for completed order: Floor %d, Direction %v\n", 
						update.Floor, update.Dir)
				}
			}
		} else {
			// New order from network, add it
			(*networkOrders)[update.OrderID] = update
			
			// Set light ONLY if assigned to us
			if update.Status == localOrders.Assigned && update.DelegatedID == localID {
				var btnType elevio.ButtonType
				if update.Dir == elevio.MD_Up {
					btnType = elevio.BT_HallUp
				} else {
					btnType = elevio.BT_HallDown
				}
				
				elevio.SetButtonLamp(btnType, update.Floor, true)
				fmt.Printf("Setting light ON for new assigned order: Floor %d, Direction %v\n", 
					update.Floor, update.Dir)
			}
		}
	}
}
