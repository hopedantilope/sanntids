package elevator
import (
	"sanntids/cmd/config"
	"Driver-go/elevio"
	"encoding/json"
	"fmt"
	"os"
)

const cabRequestsFile = "cab_requests.json"

type ElevatorBehaviour int
const (
    EB_Idle ElevatorBehaviour = iota
    EB_DoorOpen
    EB_Moving
)

func Eb_toString(eb ElevatorBehaviour) string {
    switch eb {
    case EB_Idle:
        return "idle"
    case EB_DoorOpen:
        return "doorOpen"
    case EB_Moving:
        return "moving"
    default:
        return "unknown"
    }
}

func Md_toString(md elevio.MotorDirection) string {
    switch md {
    case elevio.MD_Up:
        return "up"
    case elevio.MD_Down:
        return "down"
    case elevio.MD_Stop:
        return "stop"
    default:
        return "unknown"
    }
}

func GetCabRequests(requests [config.N_FLOORS][config.N_BUTTONS]bool) []bool {
    var cabOrders []bool
    for floor := 0; floor < config.N_FLOORS; floor++ {
        cabOrders = append(cabOrders, requests[floor][elevio.BT_Cab])
    }
    saveCabRequests(cabOrders)
    return cabOrders
}

func saveCabRequests(cabOrders []bool) {
	data, err := json.Marshal(cabOrders)
	if err != nil {
		fmt.Printf("Error marshalling cab requests: %v\n", err)
		return
	}

	err = os.WriteFile(cabRequestsFile, data, 0644)
	if err != nil {
		fmt.Printf("Error writing to cab requests file: %v\n", err)
	}
}

func loadCabRequests() []bool {
	cabRequests := make([]bool, config.N_FLOORS)

	data, err := os.ReadFile(cabRequestsFile)
	if err != nil {
		fmt.Printf("Could not read cab requests file: %v. Creating a new file with default values.\n", err)
		saveCabRequests(cabRequests)
		return cabRequests
	}

	err = json.Unmarshal(data, &cabRequests)
	if err != nil || len(cabRequests) != config.N_FLOORS {
		fmt.Printf("Invalid cab requests data, resetting to default. Error: %v\n", err)
		cabRequests = make([]bool, config.N_FLOORS)
		saveCabRequests(cabRequests)
	}

	return cabRequests
}


type Elevator struct {
    Floor     int
    MotorDirection      elevio.MotorDirection
    Requests  [config.N_FLOORS][config.N_BUTTONS]bool
    Cleared   [config.N_FLOORS][config.N_BUTTONS]bool
    Behaviour ElevatorBehaviour
    Obstruction bool

    Config struct {
        ClearRequestVariant config.ClearRequestVariant
        DoorOpenDuration_s  float64
    }
}


func ElevatorInit() Elevator {
    var zeros [config.N_FLOORS][config.N_BUTTONS]bool

    e := Elevator{
        Floor:          0,
        MotorDirection: elevio.MD_Stop,
        Requests:       zeros,
        Cleared:        zeros,  
        Behaviour:      EB_Idle,
        Obstruction:    false,
    }

    e.Config.ClearRequestVariant = config.CV_All
    e.Config.DoorOpenDuration_s  = config.DoorOpenDuration_s  

    savedCabRequests := loadCabRequests()
    
    for floor, isRequested := range savedCabRequests {
        e.Requests[floor][elevio.BT_Cab] = isRequested
    }

    return e
}