package main

import (
	"elevator-project/app"
	"elevator-project/pkg/drivers"
	"elevator-project/pkg/elevator"
	"elevator-project/pkg/orders"
	"elevator-project/pkg/transport"
	"fmt"
)

func main() {
	elevatorID := 1
	numFloors := 4

	drivers.Init("localhost:15657", numFloors)

	requestMatrix:= orders.NewRequestMatrix(numFloors)

	elevatorFSM := elevator.NewElevator(requestMatrix, elevatorID)
	go elevatorFSM.Run()

	go func() {
		err := transport.StartServer("127.0.0.1:8000", app.HandleMessage)
		if err != nil {
			fmt.Println("Server error:", err)
		}
	}()

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
