# Variables
SIMULATOR_DIR := lib/Simulator-v2
SIMULATOR_BINARY := build/SimElevatorServer
GO_BINARY := build/main

# Default target
all: build_dirs $(SIMULATOR_BINARY) $(GO_BINARY)

# Create build directories if they don't exist
build_dirs:
	mkdir -p build

# Compile the D simulator manually
$(SIMULATOR_BINARY):
	dmd -w -g $(SIMULATOR_DIR)/src/sim_server.d $(SIMULATOR_DIR)/src/timer_event.d -of$(SIMULATOR_BINARY)
	cp $(SIMULATOR_DIR)/simulator.con build/

# Build the Go application
$(GO_BINARY): $(wildcard cmd/*.go pkg/**/*.go)
	go build -o $(GO_BINARY) ./cmd

# Clean up build files
clean:
	rm -rf build

# Update submodules
submodule:
	git submodule update --init --recursive

.PHONY: all build_dirs clean submodule
