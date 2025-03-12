// main.go
package main

import (
	"elevator-project/app"
	"elevator-project/pkg/config"
	"elevator-project/pkg/drivers"
	"elevator-project/pkg/elevator"
	"elevator-project/pkg/orders"
	"elevator-project/pkg/transport"
	"elevator-project/pkg/utils"
	"flag"
)

func main() {
	pelevatorID := flag.Int("ID", 0, "elevatorID")
	flag.Parse()

	// Set the local elevator ID.
	app.LocalElevatorID = *pelevatorID

	// Initialize roles: Elevator 1 starts as master, others as slaves.
	if *pelevatorID == 1 {
		app.IsMaster = true
		app.CurrentMasterID = 1
	} else {
		app.IsMaster = false
		app.CurrentMasterID = 1
	}

	drivers.Init(config.ElevatorAddresses[*pelevatorID], config.NumFloors)
	peerAddrs := utils.GetOtherElevatorAddresses(*pelevatorID)
	requestMatrix := orders.NewRequestMatrix(config.NumFloors)
	elevatorFSM := elevator.NewElevator(requestMatrix, *pelevatorID)

	go elevatorFSM.Run()
	go transport.StartServer(config.UDPAddresses[*pelevatorID], app.HandleMessage)

	// Every elevator sends heartbeat.
	go app.StartHeartbeat(peerAddrs, *pelevatorID)
	// Every elevator sends state updates.
	go app.StartStateSender(elevatorFSM, peerAddrs)
	go app.PrintStateStore()

	// The master monitors all elevator heartbeats.
	if *pelevatorID == 1 || app.IsMaster {
		go app.MonitorElevatorHeartbeats()
	}
	// All non-master elevators run MonitorMasterHeartbeat to detect master failure.
	if *pelevatorID != app.CurrentMasterID {
		go app.MonitorMasterHeartbeat(peerAddrs)
	}

	app.RunEventLoop(elevatorFSM, requestMatrix)
}
