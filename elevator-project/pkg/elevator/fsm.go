package elevator

import (
	"elevator-project/pkg/drivers"
	"fmt"
	"sync"
	"time"
)

// Elevator states
type State int

const (
	Idle State = iota
	DoorOpen
	Moving
	ErrorState
)

type ElevatorFSM struct {
	CurrentState State
	CurrentFloor int
	Direction    drivers.MotorDirection
	RequestQueue [4][3]bool //Currently hardcoded
	Mutex        sync.Mutex
	Processing   bool
	NextOrder    chan int
	Queue		 *OrderQueue
}

func NewElevatorFSM(queue *OrderQueue) *ElevatorFSM {
	fsm := &ElevatorFSM{
		CurrentState: Idle,
		CurrentFloor: -1, // Unknown at startup
		Direction:    drivers.MD_Stop,
		Processing:   false,
		NextOrder:    make(chan int, 1),
		Queue:		  queue,
	}
	go fsm.processOrders()
	return fsm
}

func (e *ElevatorFSM) processOrders() {
	for floor := range e.NextOrder {
		e.HandleEvent("button_press", floor)
	}
}

func (e *ElevatorFSM) HandleEvent(event string, param int) {
	e.Mutex.Lock()
	defer e.Mutex.Unlock()

	switch event {
	case "button_press":
		e.handleButtonPress(param)
	case "floor_arrival":
		e.handleFloorArrival(param)
	case "door_timeout":
		e.handleDoorTimeout()
	case "obstruction":
		e.handleObstruction(param)
	case "error":
		e.handleError()
	}
}

func (e *ElevatorFSM) handleButtonPress(floor int) {
	fmt.Println("Button pressed at floor:", floor)
	if e.Processing {
		return
	}
	e.RequestQueue[floor][drivers.BT_Cab] = true
	e.Processing = true
	if e.CurrentState == Idle {
		if floor == e.CurrentFloor {
			e.changeState(DoorOpen)
		} else {
			e.changeState(Moving)
		}
	}
}

func (e *ElevatorFSM) handleFloorArrival(floor int) {
	fmt.Println("Arrived at floor:", floor)
	e.CurrentFloor = floor
	drivers.SetFloorIndicator(floor)
	drivers.SetMotorDirection(drivers.MD_Stop)
	e.Queue.ElevatorDone <- true

	if e.RequestQueue[floor][drivers.BT_Cab] {
		e.RequestQueue[floor][drivers.BT_Cab] = false
		e.changeState(DoorOpen)
	}
	e.Processing = false
	select {
	case next := <-e.NextOrder:
		e.HandleEvent("button_press", next)
	default:
	}
}

func (e *ElevatorFSM) handleDoorTimeout() {
	fmt.Println("Door timeout, closing door")
	drivers.SetDoorOpenLamp(false)
	e.changeState(Idle)
}

func (e *ElevatorFSM) handleObstruction(isObstructed int) {
	if isObstructed == 1 {
		e.changeState(ErrorState)
	} else {
		if e.CurrentState == ErrorState {
			e.changeState(Idle)
		}
	}
}

func (e *ElevatorFSM) handleError() {
	fmt.Println("Error detected! Stopping elevator.")
	drivers.SetMotorDirection(drivers.MD_Stop)
	e.changeState(ErrorState)
}

func (e *ElevatorFSM) changeState(newState State) {
	fmt.Println("Changing state to:", newState)
	e.CurrentState = newState

	switch newState {
	case Idle:
		drivers.SetMotorDirection(drivers.MD_Stop)
		select {
		case next := <-e.NextOrder:
			e.HandleEvent("button_press", next)
		default:
		}
	case Moving:
		e.setDirection()
		drivers.SetMotorDirection(e.Direction)
	case DoorOpen:
		drivers.SetDoorOpenLamp(true)
		time.AfterFunc(3*time.Second, func() {
			e.HandleEvent("door_timeout", 0)
		})
	case ErrorState:
		drivers.SetMotorDirection(drivers.MD_Stop)
	}
}

func (e *ElevatorFSM) setDirection() {
	if e.CurrentFloor < len(e.RequestQueue)-1 {
		e.Direction = drivers.MD_Up
	} else if e.CurrentFloor > 0 {
		e.Direction = drivers.MD_Down
	} else {
		e.Direction = drivers.MD_Stop
	}
}