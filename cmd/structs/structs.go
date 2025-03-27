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

// Using `json:"1"`,`json:"2"`.. to save data when sending
type HallOrder struct {
	DelegatedID string   		  `json:"1"`
	Status      OrderStatus       `json:"2"`
	Floor       int			      `json:"3"`
	Dir         elevio.ButtonType `json:"4"`
}

type HRAElevState struct {
	Obstruction bool		`json:"5"`
    Behavior    string      `json:"behaviour"`
    Floor       int         `json:"floor"` 
    Direction   string      `json:"direction"`
    CabRequests []bool      `json:"cabRequests"`
}

type ElevatorDataWithID struct {
	ElevatorID string  					  `json:"6"`
	ElevatorState map[string]HRAElevState `json:"7"`
	HallOrders    []HallOrder  			  `json:"8"`
}