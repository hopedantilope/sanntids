
package runHRA

import "os/exec"
import "fmt"
import "encoding/json"
import "runtime"
import "sanntids/cmd/localElevator/config"
import "sanntids/cmd/localElevator/structs"
import "Driver-go/elevio"
import "Network-go/network/localip"
import "cmd/util"


type HRAInput struct {
    HallRequests    [config.N_FLOORS][2]bool                   `json:"hallRequests"`
    States          map[string]structs.HRAElevState                      `json:"states"`
}

func RunHRA(elevData structs.ElevatorDataWithID, elevatorID string) structs.ElevatorDataWithID {
	orders, states := TransformFromElevatorState(elevData)
    keys := make([]string, 0, len(states))

    for key := range states {
        keys = append(keys, key)
    }

    if(util.IsLowestIP(keys, elevatorID))
	hraExecutable := ""
	switch runtime.GOOS {
		case "linux":   hraExecutable = "hall_request_assigner"
		case "windows": hraExecutable = "hall_request_assigner.exe"
		default:        panic("OS not supported")
	}
	fmt.Println(hraExecutable)
	
	// Prepare the input for the HRA executable
	input := HRAInput{
		HallRequests: orders[elevData.ElevatorID], 
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
	err = json.Unmarshal(ret, &output)
	if err != nil {
		fmt.Println("json.Unmarshal error: ", err)
		return structs.ElevatorDataWithID{}
	}
	
	fmt.Printf("output: \n")
	for k, v := range output {
		fmt.Printf("%6v :  %+v\n", k, v)
	}
	
	// Transform the HRA output back to ElevatorDataWithID
	return TransformToElevatorState(assignedOrders, states)
}

func TransformToElevatorState(assignedOrders map[string][config.N_FLOORS][2]bool, states map[string]structs.HRAElevState, elevatorID string) structs.ElevatorDataWithID {
	var transformedOrder structs.ElevatorDataWithID

	for id, arr := range assignedOrders {
		var hallOrders []structs.HallOrder
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

func TransformFromElevatorState(elevData structs.ElevatorDataWithID) (map[string][config.N_FLOORS][2]bool, map[string]structs.HRAElevState) {
	assignedOrders := make(map[string][config.N_FLOORS][2]bool)
	states := make(map[string]structs.HRAElevState)

    var orders [config.N_FLOORS][2]bool
    // Process each hall order to set the corresponding flag in the 2D orders array.
    for _, order := range elevData.HallOrders {
        switch order.Dir {
        case elevio.BT_HallDown:
            orders[order.Floor][0] = true
        case elevio.BT_HallUp:
            orders[order.Floor][1] = true
        }
    }
    // Map the elevator ID to its assigned orders and state.
    assignedOrders[elevData.ElevatorID] = orders
    states = elevData.ElevatorState

	return assignedOrders, states
}
