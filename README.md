# Elevator Control System

This project implements a distributed elevator control system that allows multiple elevators to coordinate and handle requests efficiently.

## Prerequisites

- D Compiler (for simulator and cost function)
- Go Compiler (for the main application)

## Setup

Clone the repository:

```bash
git clone git@github.com:hopedantilope/sanntids.git
cd sanntids
```

Initialize and update the submodules:
```bash
make submodule
```

Build the project:
```bash
make
```

## Running the System

To run the simulator:
```bash
./build/SimElevatorServer --port <port> --numfloors <num-floors>
```

To run the main elevator program:
```bash
./build/main --port=<port> --id=<elevator-id> --broadcast=<broadcast-port>
```

You can also run multiple elevators using the test script wich will create 3 simulated elevators:
```bash
./test.sh
```

## System Architecture

The system is designed as a distributed network of elevator controllers with no central server. Each elevator operates independently but shares information with others to optimize request handling.

### Main Components

1. **Local Elevator Control**
   - **FSM (Finite State Machine)** (`cmd/localElevator/fsm`): Controls the elevator's behavior based on its current state (idle, moving, door open)
   - **Elevator** (`cmd/localElevator/elevator`): Defines the elevator's properties and maintains its state
   - **Requests** (`cmd/localElevator/requests`): Handles button presses and decides which floor to visit next
   - **Timer** (`cmd/localElevator/timer`): Manages door timing and other time-based actions

2. **Order Management**
   - **Local States** (`cmd/localStates`): Processes button presses from this elevator and manages its state
   - **Network Orders** (`cmd/networkOrders`): Shares orders between elevators and coordinates which elevator handles which request
   - **Hall Request Assigner** (`cmd/runHRA`): Uses a cost function to optimize which elevator should handle each hall call

3. **Network Communication**
   - **Broadcast State** (`cmd/broadcastState`): Allows elevators to share their state and orders using UDP broadcasting
   - **Utility Functions** (`cmd/util`): Provides helper functions for network-related operations

4. **Configuration and Shared Structures**
   - **Config** (`cmd/config`): System-wide constants and configuration
   - **Structs** (`cmd/structs`): Data structures shared across the system

## State Management

Each elevator can be in one of the following states:
- **Idle**: The elevator is stationary with closed doors
- **Moving**: The elevator is in motion between floors
- **Door Open**: The elevator is stationary with open doors

## Request Types and Lifecycle

Requests are categorized into:
- **Hall Calls**: External buttons pressed in the hallway (up/down)
- **Cab Calls**: Internal buttons pressed inside the elevator

Request lifecycle:
1. **New**: A button is pressed and the request is created
2. **Confirmed**: All elevators acknowledge the request
3. **Assigned**: An elevator is assigned to handle the request
4. **Completed**: The request has been fulfilled (person picked up/delivered)

## Fault Tolerance

The system is designed to be fault-tolerant:
- Cab calls are saved to disk to survive system restarts
- Elevators broadcast their state to maintain system-wide consistency
- If an elevator goes offline, its assigned orders will be reassigned
- The master elevator (lowest IP) ensures consistent order assignment

## Network Communication

Elevators communicate using UDP broadcasting:
- Each elevator broadcasts its state and orders periodically
- Elevators track other elevators' state through received broadcasts
- If no updates are received from an elevator for a set period, it's considered offline

## File Structure

- `cmd/main.go`: Entry point that connects all components
- `cmd/broadcastState/`: Network communication between elevators
- `cmd/config/`: System-wide constants
- `cmd/localElevator/`: Code for controlling a single elevator
  - `elevator/`: Elevator state definition
  - `fsm/`: Finite state machine for elevator control
  - `requests/`: Logic for handling and prioritizing requests
  - `timer/`: Timing management for door operations
- `cmd/localStates/`: Local state management
- `cmd/networkOrders/`: Order distribution and management
- `cmd/runHRA/`: Hall request assignment algorithm
- `cmd/structs/`: Shared data structures
- `cmd/util/`: Helper functions
- `lib/`: External libraries for elevator hardware, simulation, and networking

## Testing and Debugging

The system includes a test script (`test.sh`) that launches multiple elevator instances, allowing you to test the coordination between elevators. Each elevator will log its actions and state changes to the terminal.