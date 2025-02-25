package shareOrders

import (
	"Driver-go/elevio"
	"fmt"
	"sanntids/cmd/localElevator/config"
	"sanntids/cmd/localElevator/elevator"
	"sanntids/cmd/localElevator/requests"
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
    DelegatedID		string    
	time			time.Time 
    Status			OrderStatus 
    Floor			int
	Dir 			elevio.MotorDirection
}

type OrderMap map[string]map[int]OrderStatus

func HallOrderManager(netUpdates <-chan HallOrder, localRequest <-chan elevio.ButtonEvent, netOut chan <- HallOrder) {
    // orders: a mapping of OrderID to OrderState
	var ID string = "A"
    orders := make(OrderMap)

    for {
        select {
        case request := <-localRequest:
			onLocalRequest()
        case update := <-netUpdates:
			onNetworkUpdate()
		case completedRequest := <- completedReqest:
			onCompletedRequest(com)
        }
    }
}

// shouldAcceptUpdate determines if the incoming update should override the local state.
func shouldAcceptNetworkUpdate(local OrderState, incoming OrderUpdate) bool {
    return true
}


func onLocalRequest(orders *OrderMap, request requestType, netOut chan <- HallOrder){
	if orders.shouldAcceptLocalOrder(request.floor, request.Button){
		//Assign order to elevator
		var DelID string = "A"
		//Register new order:
		 newOrder := HallOrder{
			DelegatedID: 	DelID,
			orderID:		ID,
			Status:			New,
			Floor:			request.Floor,
			Dir: 			request.Direction,
		 }
		//Send new order to network module
	}
}




// handleStateTransition can trigger actions based on state changes.
func onNetworkUpdate(update HallOrder) {
	shouldAcceptNetworkUpdate()
    switch update.Status {
    case Assigned:
		fmt.Printf("New order at: Floor %d, Direction %v\n", update.Floor, update.Dir)
    case Confirmed:
        // Set all OrderLights
    case Completed:
        if allNodesComplete(update.OrderID) {
            // Create a reset update.
            // netOut <- resetUpdate, send reset message to network
        }
    }
}

// allNodesComplete checks if all nodes have reached state Completed (or counter 3) for the given order.
func allNodesComplete(orderID string) bool {
    // This function should check your global view of the order state across all nodes.
    // For now, it can be a stub returning true.
    return true
}

func setLights()