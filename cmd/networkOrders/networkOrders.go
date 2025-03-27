package networkOrders

import (
	"Driver-go/elevio"
	"sanntids/cmd/config"
	"sanntids/cmd/structs"
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
	elevatorStates := make(map[string]structs.HRAElevState)
	hallOrders := make([]structs.HallOrder, 0)
	hallOrdersMap := make(map[string][]structs.HallOrder, 0)
	ipMap := make(map[string]time.Time, 0)

	transmitTicker := time.NewTicker(config.TransmitTickerMs * time.Millisecond)
	defer transmitTicker.Stop()

	for {
		select {
		case <-transmitTicker.C:
			currentTime := time.Now()
			for ip, lastSeen := range ipMap {

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

			//Get the requests assigned to localID and send them to Elevator
			myRequests := getMyRequests(hallOrders, elevatorStates, localElevatorID)
			requestsToLocalChan <- myRequests

		case incomingData := <-incomingDataChan:

			ipMap[incomingData.ElevatorID] = time.Now()
			hallOrdersMap[incomingData.ElevatorID] = incomingData.HallOrders
			if len(incomingData.ElevatorState) == 1 {
				hallOrders = incomingData.HallOrders
			}
			for id, state := range incomingData.ElevatorState {
				if incomingData.ElevatorID == id {
					elevatorStates[id] = state
				}
			}
			for _, newOrder := range incomingData.HallOrders {
				if !isDuplicateOrder(hallOrders, newOrder) {
					hallOrders = append(hallOrders, newOrder)
					continue
				}
				
				for i, order := range hallOrders {
					if order.Floor != newOrder.Floor || order.Dir != newOrder.Dir {
						continue
					}
					
					if util.IsMaster(ipMap, incomingData.ElevatorID) {
						if (order.Status == structs.New && newOrder.Status == structs.Completed) || 
						(order.Status == structs.Completed && newOrder.Status == structs.Assigned) {
							break
						}
						hallOrders[i].Status = newOrder.Status
						hallOrders[i].DelegatedID = newOrder.DelegatedID
					}
					
					// The master should only accept certain orders
					if util.IsMaster(ipMap, localElevatorID) {
						switch newOrder.Status {
						case structs.New:
							if order.Status == structs.Completed {
								hallOrders[i].Status = newOrder.Status
								hallOrders[i].DelegatedID = newOrder.DelegatedID
							}
						case structs.Assigned:
						case structs.Confirmed:
						case structs.Completed:
							if order.Status != structs.New && order.DelegatedID == incomingData.ElevatorID {
								hallOrders[i].Status = newOrder.Status
								hallOrders[i].DelegatedID = newOrder.DelegatedID
							}
						}
					}
					break
				}
			}
		case localState, ok := <-localElevStateChan:
			if !ok {
				return
			}

			elevatorStates[localElevatorID] = localState

		case localOrder, ok := <-localOrdersChan:
			if !ok {
				return
			}

			if !isDuplicateOrder(hallOrders, localOrder) {
				hallOrders = append(hallOrders, localOrder)
			} else {
				for i := range hallOrders {
					if hallOrders[i].Floor == localOrder.Floor && hallOrders[i].Dir == localOrder.Dir {
						hallOrders[i].Status = localOrder.Status
						hallOrders[i].DelegatedID = localOrder.DelegatedID
						break
					}
				}
			}
		case completedReqs:= <-completedRequetsChan:
			for _, req := range completedReqs {
				hallOrders = updateOrderStatus(hallOrders, req.Floor, int(req.Button), structs.Completed)
			}
		}
	}
}

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
	// Copy states map
	statesCopy := make(map[string]structs.HRAElevState)
	for id, state := range states {
		statesCopy[id] = state
	}

	// Copy orders slice
	ordersCopy := make([]structs.HallOrder, len(orders))
	copy(ordersCopy, orders)

	networkData := structs.ElevatorDataWithID{
		ElevatorID:    localID,
		ElevatorState: statesCopy,
		HallOrders:    ordersCopy,
	}

	if util.IsMaster(ipMap, localID) {
		networkData = assignOrders(networkData)
	}
	setAllLights(networkData)

	select {
	case outChan <- networkData:
	default:
	}
}

func updateOrderStatus(orders []structs.HallOrder, floor int, dir int, newStatus structs.OrderStatus) []structs.HallOrder {
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
        if !(state.Obstruction || state.Stop){
            newElevState[key] = state
        }
    }

    dataForHRA := data
	dataForHRA.ElevatorState = newElevState
    dataForHRA.HallOrders = pendingOrders
    newData := runHRA.RunHRA(dataForHRA)
    newData.HallOrders = append(newData.HallOrders, nonPendingOrders...)
	newData.ElevatorState = data.ElevatorState

    return newData
}


// orderKnownByAll returns true if every active node
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
    ipList := make([]string, 0, len(ipMap))

    for nodeID := range ipMap {
        ipList = append(ipList, nodeID)
    }

    for i, order := range orders {
        if order.Status == structs.New {
            if orderKnownByAll(order, hallOrdersMap, ipList) {
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
				orders[floorIndex][elevio.BT_Cab] = true
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