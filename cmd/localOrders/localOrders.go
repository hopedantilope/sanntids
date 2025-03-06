package localOrders

import (
	"Driver-go/elevio"
	"sanntids/cmd/localElevator/config"
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
	Status      OrderStatus
	Floor       int
	Dir         elevio.ButtonType
}

type OrderMap map[string]HallOrder


// HallOrderManager manages the hall orders and communicates with the network
func HallOrderManager(
	localRequest <-chan elevio.ButtonEvent, 
	netOut chan<- HallOrder) {
	
	orders := make(OrderMap)
	var localOrders HallOrder
	
	for {
		select {
		case request := <-localRequest:
			
			onLocalRequest(&orders, &localOrders, request, netOut)
		}
	}
}

func onLocalRequest(localOrders *HallOrder, request elevio.ButtonEvent, netOut chan<- HallOrder) {
	if request.Button == elevio.BT_Cab {
		return // Don't share cab orders
	}
	
	// Register new order with "undelegated" as the delegated ID
	newOrder := HallOrder{
		Status:      New,
		DelegatedID: "undelegated", // Set as undelegated initially
		Floor:       request.Floor,
		Dir:         request.Button,
	}

	
	// Add to broadcast list
	*localOrders = append(*localOrders, newOrder)
	
	// Send updated order list to network module
	netOut <- *localOrders
}

func onCompletedRequest(localOrders *HallOrder, completed HallOrder, netOut chan<- HallOrder) {

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
			elevio.SetButtonLamp(btnType, order.Floor, true)
		}
	}
}