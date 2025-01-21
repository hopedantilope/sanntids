package main

import (
	"Driver-go/elevio"
	"fmt"
	"time"
)

func main() {
	numFloors := 4
	targetFloor := 3

	// Initialize the elevator system
	elevio.Init("localhost:15657", numFloors)

	floorChan := make(chan int)

	go elevio.PollFloorSensor(floorChan)

	elevio.SetMotorDirection(elevio.MD_Up)
	fmt.Println("Elevator is moving...")

	for {
		currentFloor := <-floorChan
		fmt.Printf("Current floor: %d\n", currentFloor)

		if currentFloor == targetFloor {
			elevio.SetMotorDirection(elevio.MD_Stop)
			fmt.Printf("Elevator has arrived at floor %d\n", targetFloor)
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	elevio.SetDoorOpenLamp(true)
	fmt.Println("Door is open. Please enter/exit.")

	time.Sleep(3 * time.Second)

	elevio.SetDoorOpenLamp(false)
	fmt.Println("Door is closed.")
}
