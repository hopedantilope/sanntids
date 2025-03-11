
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

//func transformToElevatorState(e elevator.Elevator) structs.HRAElevState{
//	var elevstate structs.HRAElevState
//	elevstate.Behavior = elevator.Eb_toString(e.Behaviour)
//	elevstate.Floor = e.Floor
//	elevstate.Direction = elevator.Md_toString(e.MotorDirection)
//	var cab []bool
//	for i:= range e.Requests{
//		cab = append(cab, e.Requests[i][2])
//	}
//	elevstate.CabRequests = cab
//	return elevstate
//}

//hallrequests: [N_floors][2]bool (opp ned)


func runHRA(hallRequests [config.N_FLOORS][2]bool, states map[string]structs.HRAElevState) (map[string][config.N_FLOORS][2]bool){

    hraExecutable := ""
    switch runtime.GOOS {
        case "linux":   hraExecutable  = "hall_request_assigner"
        case "windows": hraExecutable  = "hall_request_assigner.exe"
        default:        panic("OS not supported")
    }
    fmt.Println(hraExecutable)
    inputMap := make (map[string]structs.HRAElevState)
    for id, state := range states {
        inputMap[id] = state
    }

    input := HRAInput{HallRequests: hallRequests, States: inputMap}

    jsonBytes, err := json.Marshal(input)
    if err != nil {
        fmt.Println("json.Marshal error: ", err)
        return nil
    }
    //path,_:=os.Getwd();
    ret, err := exec.Command("build/" + hraExecutable, "-i", string(jsonBytes)).CombinedOutput()
    if err != nil {
        fmt.Println("exec.Command error: ", err)
        fmt.Println(string(ret))
        return nil
    }
    
    output := make(map[string][config.N_FLOORS][2]bool)
    err = json.Unmarshal(ret, &output)
    if err != nil {
        fmt.Println("json.Unmarshal error: ", err)
        return nil
    }
        
    fmt.Printf("output: \n")
    for k, v := range output {
        fmt.Printf("%6v :  %+v\n", k, v)
    }

    return output
}

func transformToElevatorState(assigned_orders map[string][config.N_FLOORS][2]bool, states map[string]structs.HRAElevState) []structs.ElevatorDataWithID {
	var transformed_orders []structs.ElevatorDataWithID

	for id, arr := range assigned_orders {
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
		transformed_orders = append(transformed_orders, structs.ElevatorDataWithID{
			ElevatorID: id,
			HallOrders: hallOrders,
            ElevatorState: states,
		})
	}

	return transformed_orders
}

func transformFromElevatorState(elevData []structs.ElevatorDataWithID) (map[string][config.N_FLOORS][2]bool, map[string]structs.HRAElevState) {
	assignedOrders := make(map[string][config.N_FLOORS][2]bool)
	states := make(map[string]structs.HRAElevState)

	for _, data := range elevData {
		var orders [config.N_FLOORS][2]bool
		// Process each hall order to set the corresponding flag in the 2D orders array.
		for _, order := range data.HallOrders {
			switch order.Dir {
			case elevio.BT_HallDown:
				orders[order.Floor][0] = true
			case elevio.BT_HallUp:
				orders[order.Floor][1] = true
			}
		}
		// Map the elevator ID to its assigned orders and state.
		assignedOrders[data.ElevatorID] = orders
		states = data.ElevatorState
	}

	return assignedOrders, states
}


func Testto() []structs.ElevatorDataWithID {
    transf := map[string][4][2]bool{
        "one": {{false, false}, {true, false}, {false, false}, {false, false}},
        "two": {{false, false}, {false, false}, {false, false}, {false, true}},
        "three": {{false, true}, {false, false}, {true, false}, {false, false}},
    }
    states := map[string]structs.HRAElevState{
        "one": structs.HRAElevState{
            Behavior:       "moving",
            Floor:          2,
            Direction:      "up",
            CabRequests:   structs.CabRequestList{false, false, false, true},
        },
        "two": structs.HRAElevState{
            Behavior:       "idle",
            Floor:          0,
            Direction:      "stop",
            CabRequests:    structs.CabRequestList{false, false, false, false},
        },
        "three": structs.HRAElevState{
            Behavior:       "idle",
            Floor:          0,
            Direction:      "stop",
            CabRequests:    structs.CabRequestList{false, false, false, false},
        },
    }
    answer := transformToElevatorState(transf, states)
    return answer
}

func TestAll() {
	// Run the transform-to test.
	elevatorData := Testto()
	fmt.Println("Output of transformToElevatorState:")
	for _, data := range elevatorData {
		fmt.Printf("ElevatorID: %s, HallOrders: %+v, ElevatorState: %+v\n",
			data.ElevatorID, data.HallOrders, data.ElevatorState)
	}

	// Now run the transform-from function.
	assignedOrders, newStates := transformFromElevatorState(elevatorData)
	fmt.Println("\nOutput of transformFromElevatorState (assigned orders):")
	for id, orders := range assignedOrders {
		fmt.Printf("ElevatorID: %s, Orders: %+v\n", id, orders)
	}

	fmt.Println("\nReconstructed Elevator States:")
	for id, state := range newStates {
		fmt.Printf("ElevatorID: %s, State: %+v\n", id, state)
	}
}

func Test()map[string][config.N_FLOORS][2]bool{

    hallRequests := [config.N_FLOORS][2]bool{{false, false}, {true, false}, {false, false}, {false, true}}

    states := map[string]structs.HRAElevState{
        "one": structs.HRAElevState{
            Behavior:       "moving",
            Floor:          2,
            Direction:      "up",
            CabRequests:    structs.CabRequestList{false, false, false, true},
        },
        "two": structs.HRAElevState{
            Behavior:       "idle",
            Floor:          0,
            Direction:      "stop",
            CabRequests:    structs.CabRequestList{false, false, false, false},
        },
    }
    answer := runHRA(hallRequests, states)
    return answer
}
