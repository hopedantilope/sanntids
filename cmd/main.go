package main

import "Driver-go/elevio"
func main() {
    numFloors := 4

    elevio.Init("localhost:15657", numFloors)
}
