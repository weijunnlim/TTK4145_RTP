package main

import (
	"elevator-project/app"
	"elevator-project/pkg/config"
	"elevator-project/pkg/drivers"
	"elevator-project/pkg/elevator"
	"elevator-project/pkg/orders"
	"elevator-project/pkg/transport"
	"flag"
)

func main() {
	//Collecting flags and parsing them to handle correct setup of multiple elevators running on the same computer
	pelevatorID := flag.Int("ID", 0, "elevatorID")
	flag.Parse()

	drivers.Init(config.ElevatorAddresses[*pelevatorID], config.NumFloors)

	requestMatrix := orders.NewRequestMatrix(config.NumFloors)
	elevatorFSM := elevator.NewElevator(requestMatrix, *pelevatorID)

	go elevatorFSM.Run()
	go transport.StartServer(*pelevatorID, app.HandleMessage)
	go app.StartHeartbeat(*pelevatorID)
	go app.StartStateSender(elevatorFSM, *pelevatorID)
	go app.PrintStateStore()

	app.RunEventLoop(elevatorFSM, requestMatrix)
}
