package fsm

import (
	"Driver-go/elevio"
	"sanntids/cmd/localElevator/config"
	"sanntids/cmd/localElevator/elevator"
	"sanntids/cmd/localElevator/requests"
	"sanntids/cmd/localElevator/timer"
)

var elevatorVar elevator.Elevator

func ElevatorInit() {
    // Create a zero-initialized fixed-size array
    var requests [config.N_FLOORS][config.N_BUTTONS]bool

    elevatorVar = elevator.Elevator{
        Floor:     0,
        MotorDirection:      elevio.MD_Down,
        Requests:  requests,
        Behaviour: elevator.EB_Idle,
    }
    elevatorVar.Config.ClearRequestVariant = elevator.CV_All // or whatever default you want
    elevatorVar.Config.DoorOpenDuration_s = config.DoorOpenDuration_s
}

func setAllLights(e elevator.Elevator) {
	for floor := e.Floor + 1; floor < config.N_FLOORS; floor++ {
		for btn := 0; btn < config.N_BUTTONS; btn++ {
			elevio.SetButtonLamp(elevio.ButtonType(elevio.BT_Cab), floor, e.Requests[floor][elevio.BT_Cab])
		}
	}
}

func fsm_onRequestButtonPress(btnFloor int, btnType elevio.ButtonType) {

	switch elevatorVar.Behaviour {
	case elevator.EB_DoorOpen:
		if requests.Requests_shouldClearImmediately(elevatorVar, btnFloor, btnType) {
			timer.TimerStart(elevatorVar.Config.DoorOpenDuration_s)
		} else {
			elevatorVar.Requests[btnFloor][btnType] = true
		}

	case elevator.EB_Moving:
		elevatorVar.Requests[btnFloor][btnType] = true

	case elevator.EB_Idle:
		elevatorVar.Requests[btnFloor][btnType] = true
		pair := requests.Requests_chooseDirection(elevatorVar)
		elevatorVar.MotorDirection = pair.MotorDirection
		elevatorVar.Behaviour = pair.Behaviour
		switch pair.Behaviour {
		case elevator.EB_DoorOpen:
			elevio.SetDoorOpenLamp(true)
			timer.TimerStart(elevatorVar.Config.DoorOpenDuration_s)
			elevatorVar = requests.Requests_clearAtCurrentFloor(elevatorVar)

		case elevator.EB_Moving:
			elevio.SetMotorDirection(elevatorVar.MotorDirection)

		case elevator.EB_Idle:
			// Do nothing
		}
	}

	setAllLights(elevatorVar)
}

func fsmOnFloorArrival(newFloor int) {

	elevatorVar.Floor = newFloor

	elevio.SetFloorIndicator(elevatorVar.Floor)

	switch elevatorVar.Behaviour {
	case elevator.EB_Moving:
		if requests.RequestsShouldStop(elevatorVar) {
			elevio.SetMotorDirection(elevio.MD_Stop)
			elevio.SetDoorOpenLamp(true)
			elevatorVar = requests.Requests_clearAtCurrentFloor(elevatorVar)
			timer.TimerStart(elevatorVar.Config.DoorOpenDuration_s)
			setAllLights(elevatorVar)
			elevatorVar.Behaviour = elevator.EB_DoorOpen
		}
	default:
		// Do nothing for other states
	}
}

func fsmOnDoorTimeout() {

	switch elevatorVar.Behaviour {
	case elevator.EB_DoorOpen:
		pair := requests.Requests_chooseDirection(elevatorVar)
		elevatorVar.MotorDirection = pair.MotorDirection
		elevatorVar.Behaviour = pair.Behaviour

		switch elevatorVar.Behaviour {
		case elevator.EB_DoorOpen:
			timer.TimerStart(elevatorVar.Config.DoorOpenDuration_s)
			elevatorVar = requests.Requests_clearAtCurrentFloor(elevatorVar)
			setAllLights(elevatorVar)

		case elevator.EB_Moving, elevator.EB_Idle:
			elevio.SetDoorOpenLamp(false)
			elevio.SetMotorDirection(elevatorVar.MotorDirection)
		}

	default:
		// Do nothing for other states
	}
}
