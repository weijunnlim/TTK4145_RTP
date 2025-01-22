// Translating `fsm.c` into Go as `pkg/elevator/fsm.go`

package elevator

import (
	"fmt"
)

type FSM struct {
	Elevator Elevator
}

func (fsm *FSM) OnInitBetweenFloors() {
	fsm.Elevator.Dirn = DirnDown
	fmt.Println("Initializing elevator: Moving down to find a floor.")
}

func (fsm *FSM) OnFloorArrival(newFloor int) {
	fsm.Elevator.Floor = newFloor
	fmt.Printf("Elevator arrived at floor %d\n", newFloor)

	if fsm.ShouldStopAtFloor() {
		fsm.OnStopAtFloor()
	}
}

func (fsm *FSM) OnStopAtFloor() {
	fsm.Elevator.Dirn = DirnStop
	fmt.Println("Elevator stopped at floor", fsm.Elevator.Floor)
	fsm.ClearRequestsAtCurrentFloor()
}

func (fsm *FSM) ShouldStopAtFloor() bool {
	// Placeholder for logic that checks if the elevator should stop
	return true // Simplified logic for now
}

func (fsm *FSM) ClearRequestsAtCurrentFloor() {
	fmt.Println("Clearing requests at floor", fsm.Elevator.Floor)
}

func (fsm *FSM) OnRequestButtonPress(floor int, buttonType ButtonType) {
	fmt.Printf("Request button pressed: floor %d, button %d\n", floor, buttonType)
}
