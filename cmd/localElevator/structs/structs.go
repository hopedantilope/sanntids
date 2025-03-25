package structs

import (
	"Driver-go/elevio"
)

type OrderStatus int

const (
	Unknown OrderStatus = iota - 1
	New
	Confirmed
	Assigned
	Completed
)

// HallOrder structure for order management
type HallOrder struct {
	DelegatedID string
	Status      OrderStatus
	Floor       int
	Dir         elevio.ButtonType
}

type HRAElevState struct {
	Obstruction bool
    Behavior    string      `json:"behaviour"`
    Floor       int         `json:"floor"` 
    Direction   string      `json:"direction"`
    CabRequests []bool      `json:"cabRequests"`
}

type ElevatorDataWithID struct {
	ElevatorID string
	ElevatorState map[string]HRAElevState
	HallOrders    []HallOrder
}