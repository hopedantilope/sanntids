package fsm

import (
	"Driver-go/elevio"
	"fmt"
	"sanntids/cmd/localElevator/config"
	"sanntids/cmd/localElevator/elevator"
	"sanntids/cmd/localElevator/requests"
	"sanntids/cmd/localElevator/timer"
)

func setAllCabLights(e elevator.Elevator) {
	for floor := 0; floor < config.N_FLOORS; floor++ {
		elevio.SetButtonLamp(elevio.ButtonType(elevator.BT_Cab), floor, e.Requests[floor][elevator.BT_Cab])
	}
}

func moveToFirstFloor(floor <-chan int) {
	for {
		elevio.SetMotorDirection(elevio.MD_Down)

		currentFloor := <-floor
		fmt.Printf("moveToFirstFloor: got floor = %d\n", currentFloor)
		if currentFloor == 0 {
			elevio.SetMotorDirection(elevio.MD_Stop)
			break
		}
	}
}


func onRequestsUpdate(el *elevator.Elevator, newRequests [config.N_FLOORS][config.N_BUTTONS]bool) {
	el.Requests = newRequests
	switch el.Behaviour {
	case elevator.EB_DoorOpen:
		var zeros [config.N_FLOORS][config.N_BUTTONS]bool
		el.Cleared = zeros
        for floor := 0; floor < config.N_FLOORS; floor++ {
            for btnType := 0; btnType < config.N_BUTTONS; btnType++ {
                if newRequests[floor][btnType] {
                    if requests.Requests_shouldClearImmediately(*el, floor, elevio.ButtonType(btnType)) {
                        el.Cleared[floor][btnType] = true
                        timer.TimerStart(el.Config.DoorOpenDuration_s)
                    }
                }
            }
        }

	case elevator.EB_Moving:
		// Do nothing

	case elevator.EB_Idle:
		pair := requests.Requests_chooseDirection(*el)
		el.MotorDirection = pair.MotorDirection
		el.Behaviour = pair.Behaviour
		switch pair.Behaviour {
		case elevator.EB_DoorOpen:
			elevio.SetDoorOpenLamp(true)
			timer.TimerStart(el.Config.DoorOpenDuration_s)
			cleared := requests.Requests_getClearedAtCurrentFloor(*el)
			el.Cleared = cleared
			*el = requests.Requests_clearAtCurrentFloor(*el)

		case elevator.EB_Moving:
			elevio.SetMotorDirection(el.MotorDirection)

		case elevator.EB_Idle:
			// Do nothing
		}
	}

	setAllCabLights(*el)
}


func onFloorArrival(el *elevator.Elevator, newFloor int) {

	el.Floor = newFloor

	elevio.SetFloorIndicator(el.Floor)

	switch el.Behaviour {
	case elevator.EB_Moving:
		if requests.RequestsShouldStop(*el) {
			elevio.SetMotorDirection(elevio.MD_Stop)
			elevio.SetDoorOpenLamp(true)
			cleared := requests.Requests_getClearedAtCurrentFloor(*el)
			el.Cleared = cleared
			*el = requests.Requests_clearAtCurrentFloor(*el)
			timer.TimerStart(el.Config.DoorOpenDuration_s)
			setAllCabLights(*el)
			el.Behaviour = elevator.EB_DoorOpen
		}
	default:
		// Do nothing for other states
	}

}

func onDoorTimeout(el *elevator.Elevator) {

	switch el.Behaviour {
	case elevator.EB_DoorOpen:
		pair := requests.Requests_chooseDirection(*el)
		el.MotorDirection = pair.MotorDirection
		el.Behaviour = pair.Behaviour

		switch el.Behaviour {
		case elevator.EB_DoorOpen:
			timer.TimerStart(el.Config.DoorOpenDuration_s)
			*el = requests.Requests_clearAtCurrentFloor(*el)
			setAllCabLights(*el)

		case elevator.EB_Moving, elevator.EB_Idle:
			elevio.SetDoorOpenLamp(false)
			elevio.SetMotorDirection(el.MotorDirection)
		}

	default:
		// Do nothing for other states
	}
}

func onObstruction(el *elevator.Elevator, obstruction bool) {
	fmt.Printf("Obstuction: %v,behavior: %v \n", obstruction, el.Behaviour)
	switch {
	case el.Behaviour == elevator.EB_DoorOpen && obstruction:
		fmt.Println("Stopping timer")
		timer.TimerStop()
	case el.Behaviour == elevator.EB_DoorOpen && !obstruction:
		fmt.Println("Starting timer")
		timer.TimerStart(el.Config.DoorOpenDuration_s)
	}
}


func Fsm(
    drv_buttons chan [config.N_FLOORS][config.N_BUTTONS]bool,
    drv_floors chan int,
    drv_obstr chan bool,
    drv_stop chan bool,
	elevatorCh chan <- elevator.Elevator) {

    e := elevator.ElevatorInit()
	elevatorCh <- e

    setAllCabLights(e)
    elevio.SetFloorIndicator(0)
    elevio.SetDoorOpenLamp(false)
    moveToFirstFloor(drv_floors)


    for {
        select {
        case newRequests := <-drv_buttons:
            fmt.Println("Got some new requests")
			onRequestsUpdate(&e, newRequests)
			elevatorCh <- e

        case floor := <-drv_floors:
            fmt.Printf("Arrived at floor: %v \n", floor)
            onFloorArrival(&e, floor)
			elevatorCh <- e

        case <-timer.TimeoutChan():
            fmt.Println("Timeout")
            onDoorTimeout(&e)
			elevatorCh <- e

        case obstruction := <-drv_obstr:
            onObstruction(&e, obstruction)
			elevatorCh <- e

        case <-drv_stop:
            //Optional - if stop button causes a state change
        }
    }
}
