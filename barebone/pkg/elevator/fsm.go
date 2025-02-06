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
	state        ElevatorState
	currentFloor int
	targetFloor  int

	// Kanaler for kommunikasjon
	orders         <-chan drivers.ButtonEvent // Mottar ordre fra QueueManager
	ready          chan<- bool                // Signaliserer til QueueManager at heisen er klar
	fsmEvents      chan FsmEvent              // Intern kanal for å trigge state-machine-hendelser
	doorTimer      *time.Timer
	doorObstructed bool
}

type FsmEvent int

const (
	EventArrivedAtFloor FsmEvent = iota
	EventDoorTimerElapsed
	EventDoorObstructed
	EventDoorReleased
	EventSetError
)

// NewElevator er en hjelpefunksjon som lager en Elevator med de tilhørende kanalene
func NewElevator(orders <-chan drivers.ButtonEvent, ready chan<- bool) *Elevator {

	drivers.SetMotorDirection(drivers.MD_Up)

	foundFloorChan := make(chan int)

	go func() {
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()

		for {
			<-ticker.C
			currentFloor := drivers.GetFloor()
			// Here, we assume drivers.GetFloor() returns -1 if no valid floor is detected
			if currentFloor != -1 {
				// Once we detect a valid floor, send it on the channel
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
		fsmEvents:      make(chan FsmEvent),
		doorTimer:      nil,
		doorObstructed: drivers.GetObstruction(),
	}
}

// Run starter heisens state-machine-løkke
func (e *Elevator) Run() {

	// doorTimer vil bli startet hver gang vi åpner døra (DoorOpen).
	// Vi stopper timere i en `select` når vi er ferdige med dør-åpning.
	var doorTimer *time.Timer
	fmt.Printf("Elevator has been initilized\n")

	for {
		select {
		// 1. Lytt på nye ordre (fra QueueManager)
		case order := <-e.orders:
			e.handleNewOrder(order)

		// 3. Lytt på interne FSM-events (f.eks. ankommet neste etasje, dør-timer utløpt)
		case ev := <-e.fsmEvents:
			e.handleFSMEvent(ev)

		// 4. Håndter dør-timeren
		case <-func() <-chan time.Time {
			// For å unngå nil-pointer, sjekker vi om doorTimer er satt
			if doorTimer == nil {
				return make(chan time.Time) // tom kanal som aldri sender
			}
			return doorTimer.C
		}():
			// Door-timer har utløpt -> vi sender eventDoorTimerElapsed
			e.fsmEvents <- EventDoorTimerElapsed
			doorTimer = nil
		}
	}
}

// handleNewOrder tar imot en ny ordre når heisen er i Idle,
// eller setter heisen i Error-tilstand hvis den ikke er klar
func (e *Elevator) handleNewOrder(order drivers.ButtonEvent) {
	fmt.Printf("[ElevatorFSM] Ny ordre mottatt: ID=%d, Floor=%d\n", order.Button, order.Floor)

	// Hvis vi allerede er opptatt (ikke i Idle) kan vi evt. avvise eller behandle senere.
	// Her setter vi bare en slags "feil"-tilstand for å demonstrere:
	if e.state != Idle {
		fmt.Printf("[ElevatorFSM] Kan ikke ta ny ordre: heisen er i tilstand %v\n", e.state)
		e.fsmEvents <- EventSetError
		return
	}

	// Ellers godtar vi ordren
	e.targetFloor = order.Floor
	switch {
	case e.targetFloor == e.currentFloor:
		// Samme etasje -> direkte til DoorOpen
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

// handleFsmEventhåndterer interne hendelser
func (e *Elevator) handleFSMEvent(ev FsmEvent) {
	switch ev {
	case EventArrivedAtFloor:
		drivers.SetFloorIndicator(e.currentFloor)
		e.currentFloor = drivers.GetFloor()
		drivers.SetFloorIndicator(e.currentFloor)
		if e.currentFloor == e.targetFloor {
			fmt.Printf("[ElevatorFSM] Ankommet måletasje (%d). Åpner dør.\n", e.currentFloor)
			e.doorTimer.Reset(3 * time.Second)
			drivers.SetMotorDirection(drivers.MD_Stop)
			drivers.SetDoorOpenLamp(true)
			drivers.SetButtonLamp(drivers.BT_Cab, e.currentFloor, false)
			drivers.SetButtonLamp(drivers.BT_HallUp, e.currentFloor, false)
			drivers.SetButtonLamp(drivers.BT_HallDown, e.currentFloor, false)
			e.transitionTo(DoorOpen)
			//drivers.SetButtonLamp(drivers.BT_Cab, e.currentFloor, false)
		}

	case EventDoorTimerElapsed:
		if e.state == DoorOpen {
			fmt.Printf("[ElevatorFSM] Dør er lukket. Går til Idle.\n")
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
			e.doorTimer.Reset(3 * time.Second)
		}

	case EventSetError:
		e.transitionTo(Error)
		fmt.Printf("[ElevatorFSM] Transition to FSM: Error")
		drivers.SetMotorDirection(drivers.MD_Stop)
	}
}

// transitionTo utfører overgang til en ny tilstand og utfører evt. logikk
func (e *Elevator) transitionTo(newState ElevatorState) {
	e.state = newState
	switch newState {
	case Idle:
		// Signaler at heisen er klar til å motta ny ordre
		fmt.Println("[ElevatorFSM] Tilstand = Idle -> Heisen er ledig.")
		e.ready <- true

	case DoorOpen:
		go func() {
			e.fsmEvents <- EventDoorTimerElapsed
			drivers.SetDoorOpenLamp(false)
		}()

	case DoorObstructed:
		e.doorTimer.Stop()
		fmt.Println("[ElevatorFSM] State = DoorObstructeed. Door is held open indefinitely.")
		e.ready <- false

	case MovingUp:
		fmt.Println("[ElevatorFSM] Tilstand = MovingUp -> Starter bevegelse oppover.")
		drivers.SetMotorDirection(drivers.MD_Up)

	case MovingDown:
		fmt.Println("[ElevatorFSM] Tilstand = MovingDown -> Starter bevegelse nedover.")
		drivers.SetMotorDirection(drivers.MD_Down)

	case Error:
		fmt.Println("[ElevatorFSM] Tilstand = Error -> Heisen er i feilmodus.")
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
