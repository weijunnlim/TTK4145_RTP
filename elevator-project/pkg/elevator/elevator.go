// Translating `elevator.c` into Go as `pkg/elevator/elevator.go`

package elevator

import (
	"fmt"
)

type ElevatorBehaviour int

const (
	EBIdle ElevatorBehaviour = iota
	EBDoorOpen
	EBMoving
)

type Direction int

const (
	DirnUp Direction = iota
	DirnDown
	DirnStop
)

type Elevator struct {
	Floor     int
	Dirn      Direction
	Behaviour ElevatorBehaviour
}

func (eb ElevatorBehaviour) String() string {
	switch eb {
	case EBIdle:
		return "EB_Idle"
	case EBDoorOpen:
		return "EB_DoorOpen"
	case EBMoving:
		return "EB_Moving"
	default:
		return "EB_UNDEFINED"
	}
}

func (dir Direction) String() string {
	switch dir {
	case DirnUp:
		return "Dirn_Up"
	case DirnDown:
		return "Dirn_Down"
	case DirnStop:
		return "Dirn_Stop"
	default:
		return "Dirn_UNDEFINED"
	}
}

func PrintElevator(e Elevator) {
	fmt.Println("+--------------------+")
	fmt.Printf("|floor = %-2d          |\n|dirn  = %-12s|\n|behav = %-12s|\n", e.Floor, e.Dirn.String(), e.Behaviour.String())
	fmt.Println("+--------------------+")
}
