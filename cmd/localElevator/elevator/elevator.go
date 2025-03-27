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

func GetCabRequests(requests [config.N_FLOORS][config.N_BUTTONS]bool) []bool {
    var cabOrders []bool
    for floor := 0; floor < config.N_FLOORS; floor++ {
        cabOrders = append(cabOrders, requests[floor][elevio.BT_Cab])
    }
    saveCabRequestsToFile(cabOrders)
    return cabOrders
}

func saveCabRequestsToFile(cabOrders []bool) {
	data, err := json.Marshal(cabOrders)
	if err != nil {
		fmt.Println("Error marshalling cab requests:", err)
		return
	}

	err = os.WriteFile(cabRequestsFile, data, 0644)
	if err != nil {
		fmt.Println("Error writing to cab requests file:", err)
	}
}

func loadCabRequestsFromFile() []bool {
	cabRequests := make([]bool, config.N_FLOORS)

	data, err := os.ReadFile(cabRequestsFile)
	if err != nil {
		fmt.Println("Could not read cab requests file: ", err)
		saveCabRequestsToFile(cabRequests)
		return cabRequests
	}

	err = json.Unmarshal(data, &cabRequests)
	if err != nil || len(cabRequests) != config.N_FLOORS {
		fmt.Println("Invalid cab requests data, resetting to default. Error:", err)
		cabRequests = make([]bool, config.N_FLOORS)
		saveCabRequestsToFile(cabRequests)
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

    savedCabRequests := loadCabRequestsFromFile()
    
    for floor, isRequested := range savedCabRequests {
        e.Requests[floor][elevio.BT_Cab] = isRequested
    }

    return e
}