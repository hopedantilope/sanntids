package fsm

import (
	"Driver-go/elevio"
	"sanntids/cmd/config"
	"sanntids/cmd/localElevator/elevator"
	"sanntids/cmd/localElevator/requests"
	"sanntids/cmd/localElevator/timer"
	"time"
)

var lastMovingFloor int = -1
var movingStartTime time.Time

func setAllCabLights(e elevator.Elevator) {
	for floor := 0; floor < config.N_FLOORS; floor++ {
		elevio.SetButtonLamp(elevio.ButtonType(elevio.BT_Cab), floor, e.Requests[floor][elevio.BT_Cab])
	}
}

func moveToFirstFloor(floor <-chan int) {
	for {
		elevio.SetMotorDirection(elevio.MD_Down)

		currentFloor := <-floor
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
                    if requests.RequestsShouldClearImmediately(*el, floor, elevio.ButtonType(btnType)) {
                        el.Cleared[floor][btnType] = true
						el.Requests[floor][btnType] = false
                        timer.TimerStart(el.Config.DoorOpenDuration_s)
                    }
                }
            }
        }
	case elevator.EB_Moving:
		if lastMovingFloor == -1 {
			lastMovingFloor = el.Floor
			movingStartTime = time.Now()
		}
	case elevator.EB_Idle:
		lastMovingFloor = -1
		pair := requests.RequestsChooseDirection(*el)
		el.MotorDirection = pair.MotorDirection
		el.Behaviour = pair.Behaviour
		switch pair.Behaviour {
		case elevator.EB_DoorOpen:
			elevio.SetDoorOpenLamp(true)
			timer.TimerStart(el.Config.DoorOpenDuration_s)
			cleared := requests.RequestsGetClearedAtCurrentFloor(*el)
			el.Cleared = cleared
			*el = requests.RequestsClearAtCurrentFloor(*el)

		case elevator.EB_Moving:
			elevio.SetMotorDirection(el.MotorDirection)
			lastMovingFloor = el.Floor
			movingStartTime = time.Now()

		case elevator.EB_Idle:
		}
	}

	setAllCabLights(*el)
	
	if el.Behaviour == elevator.EB_Moving && 
		lastMovingFloor == el.Floor && 
		time.Since(movingStartTime) > 4*time.Second {
		el.Stop = true

	} else{
		el.Stop = false
	}
}

func onFloorArrival(el *elevator.Elevator, newFloor int) {
	// Update the last moving floor when we arrive at a new floor
	lastMovingFloor = newFloor

	el.Floor = newFloor

	elevio.SetFloorIndicator(el.Floor)

	switch el.Behaviour {
	case elevator.EB_Moving:
		if requests.RequestsShouldStop(*el) {
			elevio.SetMotorDirection(elevio.MD_Stop)
			elevio.SetDoorOpenLamp(true)
			cleared := requests.RequestsGetClearedAtCurrentFloor(*el)
			el.Cleared = cleared
			*el = requests.RequestsClearAtCurrentFloor(*el)
			timer.TimerStart(el.Config.DoorOpenDuration_s)
			setAllCabLights(*el)
			el.Behaviour = elevator.EB_DoorOpen
		}
	default:
	}

}

func onDoorTimeout(el *elevator.Elevator) {

	switch el.Behaviour {
	case elevator.EB_DoorOpen:
		pair := requests.RequestsChooseDirection(*el)
		el.MotorDirection = pair.MotorDirection
		el.Behaviour = pair.Behaviour

		switch el.Behaviour {
		case elevator.EB_DoorOpen:
			timer.TimerStart(el.Config.DoorOpenDuration_s)
			cleared := requests.RequestsGetClearedAtCurrentFloor(*el)
			el.Cleared = cleared
			*el = requests.RequestsClearAtCurrentFloor(*el)
			setAllCabLights(*el)

		case elevator.EB_Moving, elevator.EB_Idle:
			elevio.SetDoorOpenLamp(false)
			elevio.SetMotorDirection(el.MotorDirection)
		}

	default:
	}
}

func onObstruction(el *elevator.Elevator, obstruction bool) {
	el.Obstruction = obstruction
	switch {
	case obstruction:
		timer.TimerDisable()
	case !obstruction:
		timer.TimerEnable()
		if el.Behaviour == elevator.EB_DoorOpen {
			timer.TimerStart(config.DoorOpenDuration_s)
		}
	}
}

func Fsm(
    drvButtons chan [config.N_FLOORS][config.N_BUTTONS]bool,
    drvFloors chan int,
    drvObstr chan bool,
    drvStop chan bool,
	elevatorCh chan <- elevator.Elevator) {

    e := elevator.ElevatorInit()
	elevatorCh <- e

    setAllCabLights(e)
    elevio.SetFloorIndicator(0)
    elevio.SetDoorOpenLamp(false)
    moveToFirstFloor(drvFloors)


    for {
        select {
        case newRequests := <-drvButtons:
			onRequestsUpdate(&e, newRequests)
			elevatorCh <- e

        case floor := <-drvFloors:
            onFloorArrival(&e, floor)
			elevatorCh <- e

        case <-timer.TimeoutChan():
            onDoorTimeout(&e)
			elevatorCh <- e

        case obstruction := <-drvObstr:
            onObstruction(&e, obstruction)
			elevatorCh <- e

        case <-drvStop:
            //Optional - if stop button causes a state change
        }
    }
}
