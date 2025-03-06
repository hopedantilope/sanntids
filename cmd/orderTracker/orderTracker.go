package orderTracker

import (
	"fmt"
	"sanntids/cmd/localElevator/fsm"
	"sanntids/cmd/localOrders"
	"sync"
)

// OrderMap holds all orders for each elevator
type ElevatorOrderMap struct {
	// Map of elevator ID to orders
	Orders map[string][]localOrders.HallOrder
	// Mutex to protect concurrent access
	mutex sync.RWMutex
}

// Create a new OrderMap
func NewElevatorOrderMap() *ElevatorOrderMap {
	return &ElevatorOrderMap{
		Orders: make(map[string][]localOrders.HallOrder),
	}
}

// UpdateOrders updates the orders for a specific elevator
func (om *ElevatorOrderMap) UpdateOrders(elevatorID string, newOrders []localOrders.HallOrder) {
	om.mutex.Lock()
	defer om.mutex.Unlock()
	
	existingOrders, exists := om.Orders[elevatorID]
	
	if !exists {
		// First time seeing this elevator, store all its orders
		om.Orders[elevatorID] = newOrders
		return
	}
	
	// Update existing orders and add new ones
	for _, newOrder := range newOrders {
		found := false
		for i, existingOrder := range existingOrders {
			if existingOrder.OrderID == newOrder.OrderID {
				// Update existing order
				existingOrders[i] = newOrder
				found = true
				break
			}
		}
		
		if !found {
			// This is a new order, add it to our list
			existingOrders = append(existingOrders, newOrder)
		}
	}
	
	// Store updated list
	om.Orders[elevatorID] = existingOrders
	
	// Clean up completed orders if all orders for this elevator are completed
	allCompleted := true
	for _, order := range om.Orders[elevatorID] {
		if order.Status != localOrders.Completed {
			allCompleted = false
			break
		}
	}
	
	if allCompleted && len(om.Orders[elevatorID]) > 0 {
		om.Orders[elevatorID] = []localOrders.HallOrder{}
	}
}

// GetAllOrders returns a copy of all orders for all elevators
func (om *ElevatorOrderMap) GetAllOrders() map[string][]localOrders.HallOrder {
	om.mutex.RLock()
	defer om.mutex.RUnlock()
	
	// Create a deep copy of the orders map
	result := make(map[string][]localOrders.HallOrder)
	for elevID, orders := range om.Orders {
		ordersCopy := make([]localOrders.HallOrder, len(orders))
		copy(ordersCopy, orders)
		result[elevID] = ordersCopy
	}
	
	return result
}

// GetElevatorOrders returns a copy of orders for a specific elevator
func (om *ElevatorOrderMap) GetElevatorOrders(elevatorID string) []localOrders.HallOrder {
	om.mutex.RLock()
	defer om.mutex.RUnlock()
	
	orders, exists := om.Orders[elevatorID]
	if !exists {
		return []localOrders.HallOrder{}
	}
	
	// Create a deep copy
	ordersCopy := make([]localOrders.HallOrder, len(orders))
	copy(ordersCopy, orders)
	
	return ordersCopy
}

// PrintOrderMap prints the entire map of orders with elevator IDs
func (om *ElevatorOrderMap) PrintOrderMap() {
	om.mutex.RLock()
	defer om.mutex.RUnlock()
	
	fmt.Println("--------- Current Order Map ---------")
	for elevID, orders := range om.Orders {
		fmt.Printf("Elevator ID: %s, Total Orders: %d\n", elevID, len(orders))
		for i, order := range orders {
			fmt.Printf("  [%d] OrderID: %s, Floor: %d, Dir: %v, Status: %v, Delegated to: %s\n", 
				i, order.OrderID, order.Floor, order.Dir, order.Status, order.DelegatedID)
		}
	}
	fmt.Println("-------------------------------------")
}

// Combined tracking of elevator states and orders for HRA input
type SystemState struct {
	ElevatorStates map[string]fsm.ElevatorState
	orderMap       *ElevatorOrderMap
	mutex          sync.RWMutex
}

// Create a new SystemState
func NewSystemState(orderMap *ElevatorOrderMap) *SystemState {
	return &SystemState{
		ElevatorStates: make(map[string]fsm.ElevatorState),
		orderMap:       orderMap,
	}
}

// UpdateElevatorState updates the state for a specific elevator
func (ss *SystemState) UpdateElevatorState(elevatorID string, state fsm.ElevatorState) {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()
	
	ss.ElevatorStates[elevatorID] = state
}

// GetElevatorStates returns a copy of all elevator states
func (ss *SystemState) GetElevatorStates() map[string]fsm.ElevatorState {
	ss.mutex.RLock()
	defer ss.mutex.RUnlock()
	
	// Create a deep copy
	result := make(map[string]fsm.ElevatorState)
	for elevID, state := range ss.ElevatorStates {
		result[elevID] = state
	}
	
	return result
}

// PrintSystemState prints the current state of all elevators and their orders
func (ss *SystemState) PrintSystemState() {
	ss.mutex.RLock()
	states := ss.ElevatorStates
	ss.mutex.RUnlock()
	
	fmt.Println("======= Current System State =======")
	fmt.Println("Elevator States:")
	for elevID, state := range states {
		fmt.Printf("  Elevator: %s, Floor: %d, Direction: %v, Behavior: %v\n", 
			elevID, state.Floor, state.MotorDirection, state.Behaviour)
	}
	
	// Print orders using the OrderMap's function
	ss.orderMap.PrintOrderMap()
	fmt.Println("===================================")
}

// ProcessUpdate processes an update received from broadcastState
func (ss *SystemState) ProcessUpdate(update struct {
	ElevatorID string
	State      fsm.ElevatorState
	Orders     []localOrders.HallOrder
}) {
	// Update elevator state
	ss.UpdateElevatorState(update.ElevatorID, update.State)
	
	// Update orders
	ss.orderMap.UpdateOrders(update.ElevatorID, update.Orders)
	
	// Print the current system state
	ss.PrintSystemState()
}

// GetMergedOrderMap combines local and network orders for HRA
func GetMergedOrderMap(localOrdersMap localOrders.OrderMap, networkOrdersMap localOrders.OrderMap) localOrders.OrderMap {
    mergedOrders := make(localOrders.OrderMap)
    
    // Add all local orders
    for id, order := range localOrdersMap {
        mergedOrders[id] = order
    }
    
    // Add network orders, skipping any that conflict with local ones
    for id, order := range networkOrdersMap {
        if _, exists := mergedOrders[id]; !exists {
            mergedOrders[id] = order
        }
    }
    
    return mergedOrders
}