package fsm

import (
	"Driver-go/elevio"
	"sanntids/cmd/localElevator/config"
	"sanntids/cmd/localElevator/elevator"
	"sanntids/cmd/localElevator/requests"
)

var elevatorvar elevator.Elevator

func elevatorInit() {
	elevatorvar = elevator.Elevator{
		Floor:     0,
		Dirn:      elevator.MD_Down, // Ensure MD_Down is defined in the elevator package
		Requests:  [config.N_FLOORS][config.N_BUTTONS]int{},
		Behaviour: elevator.EB_Idle, // Ensure EB_Idle is defined in the elevator package
		Config: struct {
			ClearRequestVariant elevator.ClearRequestVariant
			DoorOpenDuration_s  float64
		}{
			ClearRequestVariant: config.ClearRequestVariant, // Ensure ClearRequestVariant is exported
			DoorOpenDuration_s:  config.DoorOpenDuration_s, // Ensure DoorOpenDuration_s is exported
		},
	}
}
func setAllLights(e elevator) {
	for f := e.floor + 1; f < N_FLOORS; f++ {
		for btn := 0; btn < N_BUTTONS; btn++ {
			elevio.SetButtonLamp(floor, btn, e.requests[floor][btn])
		}
	}
}

func fsm_onRequestButtonPress(btnFloor int, btnType elevio.ButtonType) {

	switch elevator.behaviour {
	case EB_DoorOpen:
		if requests.requests_shouldClearImmediately(elevatorVar, btnFloor, btnType) {
			timerStart(elevator.config.doorOpenDuration_s)
		} else {
			elevator.requests[btnFloor][btnType] = 1
		}

	case EB_Moving:
		elevator.requests[btnFloor][btnType] = 1

	case EB_Idle:
		elevator.requests[btnFloor][btnType] = 1
		pair := requests_chooseDirection(elevator)
		elevator.dirn = pair.dirn
		elevator.behaviour = pair.behaviour
		switch pair.behaviour {
		case EB_DoorOpen:
			elevio.SetDoorOpenLamp(1)
			timerStart(elevator.config.doorOpenDuration_s)
			elevator = requests_clearAtCurrentFloor(elevator)

		case EB_Moving:
			elevio.SetMotorDirection(elevator.dirn)

		case EB_Idle:
			// Do nothing
		}
	}

	setAllLights(elevator)
}

func fsmOnFloorArrival(newFloor int) {

	elevator.floor = newFloor

	elevio.floorIndicator(elevator.floor)

	switch elevator.behaviour {
	case EB_Moving:
		if requestsShouldStop(elevator) {
			SetMotorDirection(D_Stop)
			SetDoorOpenLamp(1)
			elevator = requests_clearAtCurrentFloor(elevator)
			timerStart(elevator.config.doorOpenDuration_s)
			setAllLights(elevator)
			elevator.behaviour = EB_DoorOpen
		}
	default:
		// Do nothing for other states
	}
}

func fsmOnDoorTimeout() {

	switch elevator.behaviour {
	case EB_DoorOpen:
		pair := requests_chooseDirection(elevator)
		elevator.dirn = pair.dirn
		elevator.behaviour = pair.behaviour

		switch elevator.behaviour {
		case EB_DoorOpen:
			timerStart(elevator.config.doorOpenDuration_s)
			elevator = requests_clearAtCurrentFloor(elevator)
			setAllLights(elevator)

		case EB_Moving, EB_Idle:
			elevio.SetDoorOpenLamp()(0)
			elevio.SetMotorDirection(elevator.dirn)
		}

	default:
		// Do nothing for other states
	}
}
