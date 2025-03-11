# Elevator Control System

This project implements a distributed elevator control system that allows multiple elevators to coordinate and handle requests efficiently.

## Prerequisites

- D Compiler (for simulator and cost function)
- Go Compiler (for the main application)

## Setup

Clone the repository with all submodules:

```bash
git clone git@github.com:hopedantilope/sanntids.git
cd sanntids
```

Build the project:
```bash
make submodule
make
cd build/
```

## Running the System

To run the simulator:
```bash
./SimElevatorServer --port <port> --numfloors <num-floors>
```

To run the main elevator program:
```bash
./main
```

You can also run multiple elevators using the test script:
```bash
./test.sh
```

## How the System Works

### Main Components

1. **Local Elevator Control** (`cmd/localElevator/`):
   - **FSM (Finite State Machine)**: Controls the elevator's behavior (moving, doors open, idle)
   - **Elevator**: Defines the elevator's properties and state
   - **Requests**: Handles button presses and decides where to go next
   - **Timer**: Manages door timing and other time-based actions

2. **Order Management** (`cmd/localOrders/` and `cmd/networkOrders/`):
   - **Local Orders**: Processes button presses from this elevator
   - **Network Orders**: Shares orders between elevators and coordinates which elevator handles which request

3. **Network Communication** (`cmd/broadcastState/`):
   - Allows elevators to share their state and orders with each other
   - Uses UDP broadcasting to communicate between elevators

4. **Order Assignment** (`cmd/runHRA/`):
   - Determines which elevator should handle each hall call
   - Uses a cost function to optimize elevator assignments

## Design Features

- **Distributed System**: No central controller - elevators coordinate between themselves
- **Fault Tolerance**: System continues to work even if some elevators fail
- **Cost Optimization**: Assigns orders to minimize wait time and energy usage

## File Structure

- `cmd/main.go`: Entry point that connects all components
- `cmd/localElevator/`: Code for controlling a single elevator
- `cmd/localOrders/` and `cmd/networkOrders/`: Order processing and distribution
- `cmd/network/`: Network communication between elevators
- `lib/`: External libraries for elevator hardware, simulation, and networking