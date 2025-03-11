
package runHRA

import "os/exec"
import "fmt"
import "encoding/json"
import "runtime"
import "sanntids/cmd/localElevator/config"
import "sanntids/cmd/localElevator/structs"
import "Driver-go/elevio"


type HRAInput struct {
    HallRequests    [config.N_FLOORS][2]bool                   `json:"hallRequests"`
    States          map[string]structs.HRAElevState                      `json:"states"`
}

func RunHRA(elevData structs.ElevatorDataWithID) structs.ElevatorDataWithID {
	states, orders := TransformToHRA(elevData)

	hraExecutable := ""
	switch runtime.GOOS {
		case "linux":   hraExecutable = "hall_request_assigner"
		case "windows": hraExecutable = "hall_request_assigner.exe"
		default:        hraExecutable = "hall_request_assigner"
	}
	fmt.Println(hraExecutable)
	
	fmt.Printf(orders)
	//input := TransformToHRA(elevData, elevatorID)
	input := HRAInput{
		HallRequests: orders, 
		States: states,
	}
	
	jsonBytes, err := json.Marshal(input)
	if err != nil {
		fmt.Println("json.Marshal error: ", err)
		return structs.ElevatorDataWithID{}
	}
	
	ret, err := exec.Command("build/" + hraExecutable, "-i", string(jsonBytes)).CombinedOutput()
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
	
	fmt.Printf("output: \n")
	for k, v := range assignedOrders {
		fmt.Printf("%6v :  %+v\n", k, v)
	}
	
	// Transform the HRA output back to ElevatorDataWithID
	return TransformFromHRA(assignedOrders, states, elevData.ElevatorID)
}

func TransformFromHRA(assignedOrders map[string][config.N_FLOORS][2]bool, states map[string]structs.HRAElevState, elevatorID string) structs.ElevatorDataWithID {
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

func TransformToHRA(elevData structs.ElevatorDataWithID) (map[string]structs.HRAElevState, [config.N_FLOORS][2]bool) {
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
