package main

import (
	"Network-go/network/peers"
	"flag"
	"fmt"
	"time"
)

func main() {
	// Parse command-line arguments
	// Run with: go run peers.go --id=Elevator1
	elevatorID := flag.String("id", "", "Elevator ID (defaults to local IP if not specified)")
	peerPortFlag := flag.Int("peer", 30001, "Port for peer discovery")
	flag.Parse()

	transmitEnableChan := make(chan bool)
	peerUpdatesChan := make(chan peers.PeerUpdate)

	go peers.Transmitter(*peerPortFlag, *elevatorID, transmitEnableChan)
	go peers.Receiver(*peerPortFlag, peerUpdatesChan)

	for {
		select {
		case update := <-peerUpdatesChan:
			// Print the updated list of peers
			fmt.Printf("Current peers: %v\n", update.Peers)
			if update.New != "" {
				fmt.Printf("New peer detected: %s\n", update.New)
			}
			if len(update.Lost) > 0 {
				fmt.Printf("Lost peers: %v\n", update.Lost)
			}
		case <-time.After(15 * time.Second):
			// Example: periodically disable transmission for testing purposes
			fmt.Println("Toggling transmitter off for 10 seconds")
			//We can maybe use this chan to signal that we are unavailable during an obstruction.
			transmitEnableChan <- false
			time.Sleep(10 * time.Second)
			transmitEnableChan <- true
		}
	}
}
