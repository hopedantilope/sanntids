package runHRA

import(
	"os/exec"
	"fmt"
	"encoding/json"
	"runtime"
	"sanntids/cmd/config"
	"sanntids/cmd/structs"
	"Driver-go/elevio"
	"path/filepath"
	"os"
)

type hraInput struct {
    HallRequests    [config.N_FLOORS][2]bool                   `json:"hallRequests"`
    States          map[string]structs.HRAElevState            `json:"states"`
}

func RunHRA(elevData structs.ElevatorDataWithID) structs.ElevatorDataWithID {
	states, orders := transformToHRA(elevData)

	hraExecutable := ""
	switch runtime.GOOS {
		case "linux":   hraExecutable = "hall_request_assigner"
		case "windows": hraExecutable = "hall_request_assigner.exe"
		default:        hraExecutable = "hall_request_assigner"
	}

	input := hraInput{
		HallRequests: orders, 
		States: states,
	}
	
	jsonBytes, err := json.Marshal(input)
	if err != nil {
		fmt.Println("json.Marshal error: ", err)
		return structs.ElevatorDataWithID{}
	}
	
	executablePath, err := os.Executable()
	if err != nil {
		fmt.Println("Error getting executable path:", err)
		return structs.ElevatorDataWithID{}
	}

	execDir := filepath.Dir(executablePath)
	execPath := filepath.Join(execDir, hraExecutable)
	ret, err := exec.Command(execPath, "-i", string(jsonBytes)).CombinedOutput()

	if err != nil {
		fmt.Println("exec.Command error: ", err)
		fmt.Println(string(ret))
		return structs.ElevatorDataWithID{}
	}
	
	assignedOrders := make(map[string][config.N_FLOORS][2]bool)
	err = json.Unmarshal(ret, &assignedOrders)
	if err != nil {
		fmt.Println("json.Unmarshal error: ", err)
		return structs.ElevatorDataWithID{}
	}
	
	return transformFromHRA(assignedOrders, states, elevData.ElevatorID)
}

func transformFromHRA(assignedOrders map[string][config.N_FLOORS][2]bool, states map[string]structs.HRAElevState, elevatorID string) structs.ElevatorDataWithID {
	var transformedOrder structs.ElevatorDataWithID
	var hallOrders []structs.HallOrder
	for id, arr := range assignedOrders {
		for floor, directions := range arr {
			for dir, isActive := range directions {
				if isActive {
                    if dir == 0{
					order := structs.HallOrder{
						Floor: floor,
						Dir:   elevio.BT_HallDown,
                        Status: structs.Assigned,
                        DelegatedID: id,
					}
					hallOrders = append(hallOrders, order)
                    } else {
                        order := structs.HallOrder{
                            Floor: floor,
                            Dir:   elevio.BT_HallUp,
                            Status: structs.Assigned,
                            DelegatedID: id,
                        }
                        hallOrders = append(hallOrders, order)
                    }
				}
			}
		}
	}
    transformedOrder = structs.ElevatorDataWithID{
        ElevatorID: elevatorID,
        HallOrders: hallOrders,
        ElevatorState: states,
    }
	return transformedOrder
}

func transformToHRA(elevData structs.ElevatorDataWithID) (map[string]structs.HRAElevState, [config.N_FLOORS][2]bool) {
	hallrequests := [config.N_FLOORS][2]bool{}

	for _,order := range elevData.HallOrders{
		orderFloor := order.Floor
		orderDirection := order.Dir
		if orderDirection == elevio.BT_HallDown {
			hallrequests[orderFloor][0] = true
		}	
		if orderDirection == elevio.BT_HallUp {
			hallrequests[orderFloor][1] = true
		}
	}
	return elevData.ElevatorState, hallrequests

}
