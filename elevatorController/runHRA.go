
package runHRA

import "os/exec"
import "fmt"
import "encoding/json"
import "runtime"
import "elevator"
import "requests"
import "Driver-go/elevio"

// Struct members must be public in order to be accessible by json.Marshal/.Unmarshal
// This means they must start with a capital letter, so we need to use field renaming struct tags to make them camelCase

type HRAElevState struct {
    Behavior    string      `json:"behaviour"`
    Floor       int         `json:"floor"` 
    Direction   string      `json:"direction"`
    CabRequests []bool      `json:"cabRequests"`
}

type HRAInput struct {
    HallRequests    [elevator.N_FLOORS][2]bool                   `json:"hallRequests"`
    States          map[string]HRAElevState                      `json:"states"`
}

func transformToElevatorState(e elevator.Elevator) HRAElevState{
	var elevstate HRAElevState
	elevastate.Behavior = elevator.eb_toString(e.Behaviour)
	elevastate.Floor = e.Floor
	elevastate.Direction = elevator.md_toString(e.MotorDirection)
	var cab []bool
	for i:= range e.Requests{
		cab = append(cab, e.Requests[i][2])
	}
	elevastate.CabRequests = cab
	return elevastate
}

func runHRA(hallRequests [elevator.N_FLOORS][2]bool, elevators map[string]elevator.Elevator) (map[string][elevator.N_FLOORS][2]bool){

    hraExecutable := ""
    switch runtime.GOOS {
        case "linux":   hraExecutable  = "hall_request_assigner"
        case "windows": hraExecutable  = "hall_request_assigner.exe"
        default:        panic("OS not supported")
    }

    jsonBytes, err := json.Marshal(input)
    if err != nil {
        fmt.Println("json.Marshal error: ", err)
        return
    }
    
    ret, err := exec.Command("../hall_request_assigner/"+hraExecutable, "-i", string(jsonBytes)).CombinedOutput()
    if err != nil {
        fmt.Println("exec.Command error: ", err)
        fmt.Println(string(ret))
        return
    }
    
    output := make(map[string][elevator.N_FLOORS][2]bool)

    err = json.Unmarshal(ret, &output)
    if err != nil {
        fmt.Println("json.Unmarshal error: ", err)
        return
    }
        
    fmt.Printf("output: \n")
    for k, v := range *output {
        fmt.Printf("%6v :  %+v\n", k, v)
    }

    return output
}