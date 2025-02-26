package main

import (
	"elevator-project/app"
	"elevator-project/pkg/drivers"
	"elevator-project/pkg/elevator"
	"elevator-project/pkg/orders"
	"elevator-project/pkg/transport"
	"flag"
	"fmt"
)

func main() {
	//Collecting flags and parsing them to handle correct setup of multiple elevators running on the same computer
	elevatorID := flag.Int("ID", 0, "elevatorID")
	UDPlistenAddr := flag.String("UDPaddr", "", "Listen UDP address (e.g., 127.0.0.1:8001)")
	elevatorAddrPtr := flag.String("elevAddr", "", "Elevator address")
	flag.Parse()

	elevatorAddr := "localhost:" + *elevatorAddrPtr
	numFloors := 4

	drivers.Init(elevatorAddr, numFloors)

	requestMatrix:= orders.NewRequestMatrix(numFloors)

	elevatorFSM := elevator.NewElevator(requestMatrix, *elevatorID)
	go elevatorFSM.Run()

	go func() {
		err := transport.StartServer(*UDPlistenAddr, app.HandleMessage, elevatorFSM)
		if err != nil {
			fmt.Println("Server error:", err)
		}
	}()

	go app.StartHeartbeat(*UDPlistenAddr, *elevatorID)

	drvButtons := make(chan drivers.ButtonEvent)
	drvFloors := make(chan int)
	drvObstr := make(chan bool)
	drvStop := make(chan bool)

	go drivers.PollButtons(drvButtons)
	go drivers.PollFloorSensor(drvFloors)
	go drivers.PollObstructionSwitch(drvObstr)
	go drivers.PollStopButton(drvStop)

	for {
		select {
		case be := <-drvButtons:
			switch be.Button {
			case drivers.BT_Cab:
				_ = requestMatrix.SetCabRequest(be.Floor, true)
			case drivers.BT_HallUp:
				_ = requestMatrix.SetHallRequest(be.Floor, 0, true)
			case drivers.BT_HallDown:
				_ = requestMatrix.SetHallRequest(be.Floor, 1, true)
			}
			drivers.SetButtonLamp(be.Button, be.Floor, true)

		case <-drvFloors:
			elevatorFSM.UpdateElevatorState(elevator.EventArrivedAtFloor)

		case obstr := <-drvObstr:
			if obstr {
				elevatorFSM.UpdateElevatorState(elevator.EventDoorObstructed)
			} else {
				elevatorFSM.UpdateElevatorState(elevator.EventDoorReleased)
			}

		case <-drvStop:
			// Clear all button lamps. Open for further implementation
			for f := 0; f < numFloors; f++ {
				for b := drivers.ButtonType(0); b < 3; b++ {
					drivers.SetButtonLamp(b, f, false)
				}
			}
		}
	}
}
