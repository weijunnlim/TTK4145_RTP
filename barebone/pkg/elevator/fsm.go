package elevator

import (
	"barebone/pkg/drivers"
	"fmt"
	"time"
)

type ElevatorState int

const (
	Idle ElevatorState = iota
	MovingUp
	MovingDown
	DoorOpen
	DoorObstructed
	Error
)

type Elevator struct {
	state          ElevatorState
	currentFloor   int
	targetFloor    int
	doorObstructed bool
	orders    <-chan drivers.ButtonEvent 
	ready     chan<- bool               
	fsmEvents chan FsmEvent              
	doorTimer *time.Timer
}

type FsmEvent int

const (
	EventArrivedAtFloor FsmEvent = iota
	EventDoorTimerElapsed
	EventDoorObstructed
	EventDoorReleased
	EventSetError
)

func NewElevator(orders <-chan drivers.ButtonEvent, ready chan<- bool) *Elevator {
	drivers.SetMotorDirection(drivers.MD_Up)
	foundFloorChan := make(chan int)

	go func() {
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()

		for {
			<-ticker.C
			currentFloor := drivers.GetFloor()
			if currentFloor != -1 {
				foundFloorChan <- currentFloor
				drivers.SetMotorDirection(drivers.MD_Stop)
				return
			}
		}
	}()

	validFloor := <-foundFloorChan

	return &Elevator{
		state:          Idle,
		currentFloor:   validFloor,
		orders:         orders,
		ready:          ready,
		fsmEvents:      make(chan FsmEvent, 10),
		doorTimer:      nil,
		doorObstructed: drivers.GetObstruction(),
	}
}

func (e *Elevator) Run() {
	for {
		select {
		case order := <-e.orders:
			e.handleNewOrder(order)

		case ev := <-e.fsmEvents:
			e.handleFSMEvent(ev)

		case <-func() <-chan time.Time {
			if e.doorTimer == nil {
				return make(chan time.Time)
			}
			return e.doorTimer.C
		}():
			e.fsmEvents <- EventDoorTimerElapsed
		}
	}
}

func (e *Elevator) handleNewOrder(order drivers.ButtonEvent) {
	fmt.Printf("[ElevatorFSM] Ny ordre mottatt: ID=%d, Floor=%d\n", order.Button, order.Floor)

	if e.state != Idle {
		fmt.Printf("[ElevatorFSM] Kan ikke ta ny ordre: heisen er i tilstand %v\n", e.state)
		e.fsmEvents <- EventSetError
		return
	}

	e.targetFloor = order.Floor
	switch {
	case e.targetFloor == e.currentFloor:
		fmt.Println("[ElevatorFSM] Ordre er i samme etasje -> åpner dør")
		e.transitionTo(DoorOpen)

	case e.targetFloor > e.currentFloor:
		fmt.Println("[ElevatorFSM] Flytter til MovingUp")
		e.transitionTo(MovingUp)

	case e.targetFloor < e.currentFloor:
		fmt.Println("[ElevatorFSM] Flytter til MovingDown")
		e.transitionTo(MovingDown)
	}
}

func (e *Elevator) handleFSMEvent(ev FsmEvent) {
	switch ev {
	case EventArrivedAtFloor:
		if e.state == MovingUp || e.state == MovingDown {
			drivers.SetFloorIndicator(e.currentFloor)
			e.currentFloor = drivers.GetFloor()
			drivers.SetFloorIndicator(e.currentFloor)
			if e.currentFloor == e.targetFloor {
				fmt.Printf("[ElevatorFSM] Ankommet måletasje (%d). Åpner dør.\n", e.currentFloor)
				drivers.SetMotorDirection(drivers.MD_Stop)
				drivers.SetDoorOpenLamp(true)
				drivers.SetButtonLamp(drivers.BT_Cab, e.currentFloor, false)
				drivers.SetButtonLamp(drivers.BT_HallUp, e.currentFloor, false)
				drivers.SetButtonLamp(drivers.BT_HallDown, e.currentFloor, false)
				e.transitionTo(DoorOpen)
			}
		}

	case EventDoorTimerElapsed:
		if e.state == DoorOpen {
			fmt.Printf("[ElevatorFSM] Dør er lukket. Går til Idle.\n")
			drivers.SetDoorOpenLamp(false)
			e.transitionTo(Idle)
		}

	case EventDoorObstructed:
		if e.state == DoorOpen {
			fmt.Println("[ElevatorFSM] Door hold pressed.")
			e.transitionTo(DoorObstructed)
		}

	case EventDoorReleased:
		if e.state == DoorObstructed {
			fmt.Println("[ElevatorFSM] Door hold released.")
			e.transitionTo(DoorOpen)
		}

	case EventSetError:
		e.transitionTo(Error)
		fmt.Printf("[ElevatorFSM] Transition to FSM: Error")
		drivers.SetMotorDirection(drivers.MD_Stop)
	}
}

func (e *Elevator) transitionTo(newState ElevatorState) {
	/*
		if e.state == DoorOpen && e.doorTimer != nil {
			if !e.doorTimer.Stop() {
				<-e.doorTimer.C // drain
			}
			e.doorTimer = nil
		}
	*/
	
	e.state = newState
	switch newState {
	case Idle:
		fmt.Println("[ElevatorFSM] Tilstand = Idle -> Heisen er ledig.")
		e.ready <- true

	case DoorOpen:
		e.doorTimer = time.NewTimer(3 * time.Second)
		fmt.Println("[ElevatorFSM] Tilstand = Door open")

	case DoorObstructed:
		if e.doorTimer != nil {
			if !e.doorTimer.Stop() {
				<-e.doorTimer.C
			}
			e.doorTimer = nil
		}
		fmt.Println("[ElevatorFSM] State = DoorObstructeed.")

	case MovingUp:
		fmt.Println("[ElevatorFSM] Tilstand = MovingUp.")
		drivers.SetMotorDirection(drivers.MD_Up)

	case MovingDown:
		fmt.Println("[ElevatorFSM] Tilstand = MovingDown.")
		drivers.SetMotorDirection(drivers.MD_Down)

	case Error:
		fmt.Println("[ElevatorFSM] Tilstand = Error.")
		drivers.SetMotorDirection(drivers.MD_Stop)
	}
}

func (e *Elevator) UpdateElevatorState(state FsmEvent) {
	switch state {
	case EventArrivedAtFloor:
		e.fsmEvents <- EventArrivedAtFloor

	case EventDoorObstructed:
		e.fsmEvents <- EventDoorObstructed

	case EventDoorReleased:
		e.fsmEvents <- EventDoorReleased
	}
}
