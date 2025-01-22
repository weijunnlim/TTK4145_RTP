// Translating `requests.c` into Go as `pkg/elevator/requests.go`

package elevator

type Requests struct {
	Elevator *Elevator
}

func (r *Requests) RequestsAbove() bool {
	for f := r.Elevator.Floor + 1; f < 4; f++ { // Assume 4 floors for simplicity
		for b := 0; b < 3; b++ {
			if r.Elevator.Requests[f][b] {
				return true
			}
		}
	}
	return false
}

func (r *Requests) RequestsBelow() bool {
	for f := 0; f < r.Elevator.Floor; f++ {
		for b := 0; b < 3; b++ {
			if r.Elevator.Requests[f][b] {
				return true
			}
		}
	}
	return false
}

func (r *Requests) RequestsHere() bool {
	for b := 0; b < 3; b++ {
		if r.Elevator.Requests[r.Elevator.Floor][b] {
			return true
		}
	}
	return false
}

type DirnBehaviourPair struct {
	Dirn      Direction
	Behaviour ElevatorBehaviour
}

func (r *Requests) ChooseDirection() DirnBehaviourPair {
	switch r.Elevator.Dirn {
	case DirnUp:
		if r.RequestsAbove() {
			return DirnBehaviourPair{DirnUp, EBMoving}
		} else if r.RequestsHere() {
			return DirnBehaviourPair{DirnStop, EBDoorOpen}
		} else if r.RequestsBelow() {
			return DirnBehaviourPair{DirnDown, EBMoving}
		}
	case DirnDown:
		if r.RequestsBelow() {
			return DirnBehaviourPair{DirnDown, EBMoving}
		} else if r.RequestsHere() {
			return DirnBehaviourPair{DirnStop, EBDoorOpen}
		} else if r.RequestsAbove() {
			return DirnBehaviourPair{DirnUp, EBMoving}
		}
	case DirnStop:
		if r.RequestsHere() {
			return DirnBehaviourPair{DirnStop, EBDoorOpen}
		} else if r.RequestsAbove() {
			return DirnBehaviourPair{DirnUp, EBMoving}
		} else if r.RequestsBelow() {
			return DirnBehaviourPair{DirnDown, EBMoving}
		}
	}
	return DirnBehaviourPair{DirnStop, EBIdle}
}

func (r *Requests) ClearAtCurrentFloor() {
	for b := 0; b < 3; b++ {
		r.Elevator.Requests[r.Elevator.Floor][b] = false
	}
}
