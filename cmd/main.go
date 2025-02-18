// main.go
package main

import (
	"Driver-go/elevio"
	"fmt"
	"sanntids/cmd/localElevator/fsm"
	"sanntids/cmd/network/broadcastState"
	"time"
	"Network-go/network/localip"
)

func main() {

	numFloors := 4

	elevio.Init("localhost:15657", numFloors)
	id, _ := localip.LocalIP()
	fmt.Println("Elevator started with id: ", id)

	drv_buttons := make(chan elevio.ButtonEvent)
	drv_floors := make(chan int)
	drv_obstr := make(chan bool)
	drv_stop := make(chan bool)

	go elevio.PollButtons(drv_buttons)
	go elevio.PollFloorSensor(drv_floors)
	go elevio.PollObstructionSwitch(drv_obstr)
	go elevio.PollStopButton(drv_stop)

	stateRequestTx := make(chan chan fsm.ElevatorState)

	go fsm.Fsm(drv_buttons, drv_floors, drv_obstr, drv_stop, stateRequestTx)

	broadcastPort := 30003
	broadcastStateChan := make(chan broadcastState.ElevatorStateWithID)
	receiveStateChan := make(chan broadcastState.ElevatorStateWithID)

	go broadcastState.BroadcastState(broadcastStateChan, broadcastPort)
	go broadcastState.ReceiveState(receiveStateChan, broadcastPort)


	broadcastTicker := time.NewTicker(1 * time.Second) 
	defer broadcastTicker.Stop()

	for {
		select {
		case <-broadcastTicker.C: 
			requestChan := make(chan fsm.ElevatorState) 
			stateRequestTx <- requestChan           
			currentState := <-requestChan           

			fmt.Printf("Timed broadcast: Floor %d, Direction %v, Behaviour %v\n", currentState.Floor, currentState.MotorDirection, currentState.Behaviour)

			stateWithID := broadcastState.ElevatorStateWithID{
				ElevatorID:   id,
				ElevatorState: currentState,
			}
			broadcastStateChan <- stateWithID

		case receivedStateWithID := <-receiveStateChan:
			fmt.Printf("Received broadcast state (ID: %s): Floor %d, Direction %v, Behaviour %v\n",
				receivedStateWithID.ElevatorID, receivedStateWithID.ElevatorState.Floor, receivedStateWithID.ElevatorState.MotorDirection, receivedStateWithID.ElevatorState.Behaviour)

		}
	}

	select {}

}