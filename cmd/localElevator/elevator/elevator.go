package elevator
import (
	"sanntids/cmd/localElevator/config"
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
    }

    tempFile := cabRequestsFile + ".tmp"
    
    err = os.WriteFile(tempFile, data, 0644)
    if err != nil {
        fmt.Printf("Error writing to temp file: %v\n", err)
    }

    err = os.Rename(tempFile, cabRequestsFile)
    if err != nil {
        fmt.Printf("Error renaming temp file: %v\n", err)
    }
}

// If the file doesn't exist or has an error, it returns a default cab requests slice
func LoadCabRequests() []bool {
    // Default cab requests - all false
    cabRequests := make([]bool, config.N_FLOORS)
    
    // Try to load the file
    data, err := os.ReadFile(cabRequestsFile)
    if err != nil {
        fmt.Printf("Could not read cab requests file: %v. Using default values.\n", err)
        return cabRequests
    }

    // Parse the saved requests
    err = json.Unmarshal(data, &cabRequests)
    if err != nil {
        fmt.Printf("Error parsing cab requests file: %v. Using default values.\n", err)
        return cabRequests
    }

    // Validate the loaded cab requests
    if len(cabRequests) != config.N_FLOORS {
        fmt.Printf("Invalid cab requests length: %d (expected %d). Using default values.\n", 
            len(cabRequests), config.N_FLOORS)
        return cabRequests
    }

    fmt.Println("Successfully loaded cab requests from file")
    return cabRequests
}

type Button int
const (
    BT_HallUp   Button = iota
    BT_HallDown
    BT_Cab
    N_BUTTONS // This will be 3
)

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
    // Create a zero-initialized fixed-size 2D array for Requests
    var zeros [config.N_FLOORS][config.N_BUTTONS]bool

    // Initialize elevator with default values
    e := Elevator{
        Floor:          0,
        MotorDirection: elevio.MD_Stop,  // Default motor direction
        Requests:       zeros,
        Cleared:        zeros,  
        Behaviour:      EB_Idle,         // Default behaviour
        Obstruction:    false,
    }

    // Configure additional settings
    e.Config.ClearRequestVariant = config.CV_All               // Default request clearing variant
    e.Config.DoorOpenDuration_s  = config.DoorOpenDuration_s  

    // Attempt to load saved cab requests from file
    savedCabRequests := LoadCabRequests()
    
    // Apply the loaded cab requests to the elevator
    for floor, isRequested := range savedCabRequests {
        e.Requests[floor][BT_Cab] = isRequested
    }

    return e
}