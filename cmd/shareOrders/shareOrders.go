package shareOrders

import (
	"Driver-go/elevio"
	"fmt"
	"time"
)

// OrderStatus represents the order lifecycle state.
type OrderStatus int

const (
	Unknown OrderStatus = iota - 1
	Assigned
	Confirmed
	Completed
)

type HallOrder struct {
	DelegatedID string
	OrderID     string
	Status      OrderStatus
	Floor       int
	Dir         elevio.MotorDirection
	Time        time.Time
}

type OrderMap map[string]HallOrder

// HallOrderManager manages the hall orders and communicates with the network
func HallOrderManager(
	netUpdates <-chan []HallOrder, 
	localRequest <-chan elevio.ButtonEvent, 
	netOut chan<- []HallOrder,
	completedRequest <-chan HallOrder) {
	
	// orders: a mapping of OrderID to HallOrder
	orders := make(OrderMap)
	ID := 'A'
	var localOrders []HallOrder
	
	for {
		select {
		case request := <-localRequest:
			onLocalRequest(&orders, &localOrders, request, string(ID), netOut)
			ID++ // Increment order ID
			
		case updateList := <-netUpdates:
			// Process orders from other elevators
			for _, update := range updateList {
				onNetworkUpdate(update)
			}
			
		case completed := <-completedRequest:
			onCompletedRequest(&orders, &localOrders, completed, netOut)
		}
	}
}

// shouldAcceptNetworkUpdate determines if the incoming update should override the local state.
func shouldAcceptNetworkUpdate(local HallOrder, incoming HallOrder) bool {
	switch {
	case local.Status == Unknown && incoming.Status == Completed:
		return false
	case local.Status == Completed && incoming.Status == Unknown:
		return true
	case incoming.Status > local.Status:
		return true
	default:
		return false
	}
	return true
}

func shouldAcceptLocalOrder(orders *OrderMap, floor int, button elevio.ButtonType) bool {
	// Check if this is a hall order and if it's already being handled
	if button == elevio.BT_Cab {
		return false 
	}
	
	// Add logic to check if order already exists
	return true
}

func onLocalRequest(orders *OrderMap, localOrders *[]HallOrder, request elevio.ButtonEvent, orderID string, netOut chan<- []HallOrder) {
	if request.Button == elevio.BT_Cab {
		return // Don't share cab orders
	}
	
	if shouldAcceptLocalOrder(orders, request.Floor, request.Button) {
		// Assign order to elevator
		var delID string = "A" // Local ID
		
		// Convert button type to direction
		var dir elevio.MotorDirection
		if request.Button == elevio.BT_HallUp {
			dir = elevio.MD_Up
		} else {
			dir = elevio.MD_Down
		}
		
		// Register new order
		newOrder := HallOrder{
			DelegatedID: delID,
			OrderID:     orderID,
			Status:      Assigned,
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
		
		// Set the corresponding button light
		elevio.SetButtonLamp(request.Button, request.Floor, true)
		
		fmt.Printf("New local order created: Floor %d, Direction %v\n", newOrder.Floor, newOrder.Dir)
	}
}

func onNetworkUpdate(update HallOrder) {
	// Simply print the orders coming from other elevators
	fmt.Printf("Received network order: ID %s, Floor %d, Direction %v, Status %v\n", 
		update.OrderID, update.Floor, update.Dir, update.Status)
		
	// Set light for the hall order
	var btnType elevio.ButtonType
	if update.Dir == elevio.MD_Up {
		btnType = elevio.BT_HallUp
	} else {
		btnType = elevio.BT_HallDown
	}
	
	elevio.SetButtonLamp(btnType, update.Floor, true)
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
	for f := 0; f < 4; f++ { // Assuming 4 floors
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