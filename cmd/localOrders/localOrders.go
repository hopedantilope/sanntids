package localOrders

import (
	"Driver-go/elevio"
	"fmt"
	"sanntids/cmd/localElevator/config"
	"time"
)

// OrderStatus represents the order lifecycle state.
type OrderStatus int

const (
	Unknown OrderStatus = iota - 1
	New
	Assigned
	Confirmed
	Completed
)

// HallOrder structure for order management
type HallOrder struct {
	DelegatedID string
	OrderID     string
	Status      OrderStatus
	Floor       int
	Dir         elevio.MotorDirection
	Time        time.Time
}

// OrderMap maps OrderID to HallOrder
type OrderMap map[string]HallOrder

// Convert to HRA compatible format
func ConvertToHRAHallRequests(orders OrderMap) [config.N_FLOORS][2]bool {
	var hallRequests [config.N_FLOORS][2]bool
	
	for _, order := range orders {
		if order.Status != Completed {
			dirIndex := 0
			if order.Dir == elevio.MD_Up {
				dirIndex = 0
			} else {
				dirIndex = 1
			}
			hallRequests[order.Floor][dirIndex] = true
		}
	}
	
	return hallRequests
}

// HallOrderManager manages the hall orders and communicates with the network
func HallOrderManager(
	netUpdates <-chan []HallOrder, 
	localRequest <-chan elevio.ButtonEvent, 
	netOut chan<- []HallOrder,
	completedRequest <-chan HallOrder) {
	
	// orders: a mapping of OrderID to HallOrder
	orders := make(OrderMap)
	var localOrders []HallOrder
	
	// Use timestamp and counter for unique IDs
	var orderCounter uint64 = 0
	
	for {
		select {
		case request := <-localRequest:
			// Create unique order ID using timestamp and counter
			timestamp := time.Now().UnixNano()
			orderID := fmt.Sprintf("%d-%d", timestamp, orderCounter)
			orderCounter++
			
			onLocalRequest(&orders, &localOrders, request, orderID, netOut)
			
		case completed := <-completedRequest:
			onCompletedRequest(&orders, &localOrders, completed, netOut)
		}
	}
}

func shouldAcceptLocalOrder(orders *OrderMap, floor int, button elevio.ButtonType) bool {
	// Check if this is a hall order and if it's already being handled
	if button == elevio.BT_Cab {
		return false 
	}
	
	// Convert button type to direction for checking
	var dir elevio.MotorDirection
	if button == elevio.BT_HallUp {
		dir = elevio.MD_Up
	} else {
		dir = elevio.MD_Down
	}
	
	// Check if this order already exists and is not completed
	for _, order := range *orders {
		if order.Floor == floor && order.Dir == dir && order.Status != Completed {
			return false // Order already exists and is active
		}
	}
	
	return true
}

func onLocalRequest(orders *OrderMap, localOrders *[]HallOrder, request elevio.ButtonEvent, orderID string, netOut chan<- []HallOrder) {
	if request.Button == elevio.BT_Cab {
		return // Don't share cab orders
	}
	
	if shouldAcceptLocalOrder(orders, request.Floor, request.Button) {
		// Convert button type to direction
		var dir elevio.MotorDirection
		if request.Button == elevio.BT_HallUp {
			dir = elevio.MD_Up
		} else {
			dir = elevio.MD_Down
		}
		
		// Register new order with "undelegated" as the delegated ID
		newOrder := HallOrder{
			OrderID:     orderID,
			Status:      New,
			DelegatedID: "undelegated", // Set as undelegated initially
			Floor:       request.Floor,
			Dir:         dir,
			Time:        time.Now(),
		}
		
		// Add to local orders map
		(*orders)[orderID] = newOrder
		
		// Add to broadcast list
		*localOrders = append(*localOrders, newOrder)
		
		// Send updated order list to network module
		netOut <- *localOrders
		
		fmt.Printf("New local order created: Floor %d, Direction %v, Status: %v, DelegatedID: %s\n", 
			newOrder.Floor, newOrder.Dir, newOrder.Status, newOrder.DelegatedID)
	}
}

func onCompletedRequest(orders *OrderMap, localOrders *[]HallOrder, completed HallOrder, netOut chan<- []HallOrder) {
	// Find the order in our local map
	if order, exists := (*orders)[completed.OrderID]; exists {
		// Update status
		order.Status = Completed
		(*orders)[completed.OrderID] = order
		
		// Update in broadcast list
		for i, o := range *localOrders {
			if o.OrderID == completed.OrderID {
				(*localOrders)[i].Status = Completed
				break
			}
		}
		
		// Send updated order list to network module
		netOut <- *localOrders
		
		// Turn off the corresponding button light
		var btnType elevio.ButtonType
		if order.Dir == elevio.MD_Up {
			btnType = elevio.BT_HallUp
		} else {
			btnType = elevio.BT_HallDown
		}
		
		elevio.SetButtonLamp(btnType, order.Floor, false)
		
		fmt.Printf("Order completed: Floor %d, Direction %v\n", order.Floor, order.Dir)
	}
}

// Helper function to set all lights based on current orders
func SetLights(orders OrderMap) {
	// Clear all lights first
	for f := 0; f < config.N_FLOORS; f++ {
		elevio.SetButtonLamp(elevio.BT_HallUp, f, false)
		elevio.SetButtonLamp(elevio.BT_HallDown, f, false)
	}
	
	// Set lights based on active orders
	for _, order := range orders {
		if order.Status != Completed {
			var btnType elevio.ButtonType
			if order.Dir == elevio.MD_Up {
				btnType = elevio.BT_HallUp
			} else {
				btnType = elevio.BT_HallDown
			}
			elevio.SetButtonLamp(btnType, order.Floor, true)
		}
	}
}