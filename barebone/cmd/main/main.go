package main

import (
	"barebone/pkg/drivers"
	"barebone/pkg/elevator"
	//"barebone/pkg/utils"
	//"fmt"
)

func main() {
	numFloors := 4

	drivers.Init("localhost:15657", numFloors)

	// Opprett kanaler
	newOrders := make(chan drivers.ButtonEvent)
	elevatorReady := make(chan bool)
	elevatorOrders := make(chan drivers.ButtonEvent)

	// Start QueueManager i egen goroutine
	go elevator.QueueManager(newOrders, elevatorReady, elevatorOrders)

	// Opprett Elevator-instans (FSM)
	elevatorFSM := elevator.NewElevator(elevatorOrders, elevatorReady)

	// Start ElevatorFSM i egen goroutine
	go elevatorFSM.Run()

	// I utgangspunktet er heisen "ledig"
	go func() {
		elevatorReady <- true
	}()

	drv_buttons := make(chan drivers.ButtonEvent)
	drv_floors := make(chan int)
	drv_obstr := make(chan bool)
	drv_stop := make(chan bool)
	go drivers.PollButtons(drv_buttons)
	go drivers.PollFloorSensor(drv_floors)
	go drivers.PollObstructionSwitch(drv_obstr)
	go drivers.PollStopButton(drv_stop)

	for {
		select {
		case a := <-drv_buttons:
			//fmt.Printf("%+v\n", a)
			newOrders <- a

		case <-drv_floors:
			//fmt.Printf("%+v\n", a)
			elevatorFSM.UpdateElevatorState(elevator.EventArrivedAtFloor)

		case a := <-drv_obstr:
			//fmt.Printf("%+v\n", a)
			if a {
				elevatorFSM.UpdateElevatorState(elevator.EventDoorObstructed)
			} else {
				elevatorFSM.UpdateElevatorState(elevator.EventDoorReleased)
			}

		case <-drv_stop:
			//fmt.Printf("%+v\n", a)
			for f := 0; f < numFloors; f++ {
				for b := drivers.ButtonType(0); b < 3; b++ {
					drivers.SetButtonLamp(b, f, false)
				}
			}
		}
	}
}
