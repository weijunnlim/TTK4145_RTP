package orders

import (
	"elevator-project/pkg/drivers"
	"fmt"
	"math"
)

const ( //given in the example code
	travelTime   = 2500 //Example travel time in milliseconds
	doorOpenTime = 3000 //Example door open time in milliseconds
)

type Dirn int

type ElevatorState struct {
	RequestMatrix *RequestMatrix
	Floor         int
	Direction     Dirn
}

type Req struct {
	Active     bool
	AssignedTo string
}

type State struct {
	ID    string
	State ElevatorState
	Time  int64 //Assuming time as an int64 representation, last time the state was updated
}

// Shall be used by the master to ensure we dont assign the same req multiple times
func isUnassigned(r Req) bool {
	return r.Active && r.AssignedTo == "" //checks if request is active and not assign to any elevator
}

// check periodically?
func anyUnassigned(reqs [][]Req) bool {
	for _, floor := range reqs {
		for _, r := range floor { //iterates thorugh the whole request matrix
			if isUnassigned(r) { //unassigned?
				return true
			}
		}
	}
	return false
}

func clearRequestsAtFloor(e SimulatedElevator) SimulatedElevator {
	e.RequestMat.CabRequests[e.Floor] = false //always clear cab requests

	switch e.Direction {
	case drivers.MD_Up:
		e.RequestMat.HallRequests[e.Floor][0] = false //Clear only up button
	case drivers.MD_Down:
		e.RequestMat.HallRequests[e.Floor][1] = false //Clear only down button
	case drivers.MD_Stop: //If idle, clear both
		e.RequestMat.HallRequests[e.Floor][0] = false
		e.RequestMat.HallRequests[e.Floor][1] = false
	}

	return e
}

func requestsAbove(e ElevatorState) bool {
	for i := e.Floor + 1; i < len(e.RequestMatrix.HallRequests); i++ { //Loop through all floors ABOVE the current one
		for _, req := range e.RequestMatrix.HallRequests[i] { //Check hall requests (up/down buttons)
			if req {
				return true //Found a request above, return true
			}
		}
		if e.RequestMatrix.CabRequests[i] { //Check if an inside cab request exists above
			return true
		}
	}
	return false //No requests found above
}

// The same just for below
func requestsBelow(e ElevatorState) bool {
	for i := 0; i < e.Floor; i++ { //Loop through all floors BELOW the current one
		for _, req := range e.RequestMatrix.HallRequests[i] {
			if req {
				return true
			}
		}
		if e.RequestMatrix.CabRequests[i] {
			return true
		}
	}
	return false
}

func anyRequestsAtFloor(e ElevatorState) bool {
	if e.Floor < 0 || e.Floor >= len(e.RequestMatrix.HallRequests) {
		fmt.Printf("[ERROR] Floor index out of range: %d\n", e.Floor)
		return false
	}

	for _, req := range e.RequestMatrix.HallRequests[e.Floor] {
		if req {
			return true
		}
	}
	return e.RequestMatrix.CabRequests[e.Floor]
}

func shouldStop(e SimulatedElevator) bool {
	switch e.Direction {
	case drivers.MD_Up:
		return e.RequestMat.HallRequests[e.Floor][0] || e.RequestMat.CabRequests[e.Floor] //if the elevator moves up, and there are a up request at this floor or cab
	case drivers.MD_Down:
		return e.RequestMat.HallRequests[e.Floor][1] || e.RequestMat.CabRequests[e.Floor] // Same for down
	default:
		return true
	}
}

func chooseDirection(e ElevatorState) Dirn {
	if requestsAbove(e) {
		return 1 //Keep moving up
	} else if anyRequestsAtFloor(e) {
		return 0 //Stay at this floor (open doors)
	} else if requestsBelow(e) {
		return -1 //Move down
	}
	return 0 //No requests, stay idle
}

// Simulate an elevator for taking the time, to not fuck with our actual elevators
type SimulatedElevator struct {
	Floor      int
	Direction  drivers.MotorDirection
	RequestMat *RequestMatrix
}

func CalculateTimeToIdle(e SimulatedElevator) int {
	fmt.Printf("[DEBUG] Simulating elevator at Floor %d\n", e.Floor)

	if e.Floor < 0 || e.Floor >= len(e.RequestMat.HallRequests) {
		fmt.Printf("[ERROR] Invalid floor index: %d\n", e.Floor)
		return math.MaxInt32
	}

	duration := 0
	switch e.Direction {
	case drivers.MD_Stop:
		e.Direction = drivers.MotorDirection(chooseDirection(ElevatorState{
			RequestMatrix: e.RequestMat,
			Floor:         e.Floor,
			Direction:     Dirn(e.Direction),
		}))
		if e.Direction == drivers.MD_Stop {
			return duration
		}
	case drivers.MD_Up, drivers.MD_Down:
		duration += travelTime / 2
		e.Floor += int(e.Direction)
	}

	for {
		if shouldStop(e) {
			e = clearRequestsAtFloor(e)
			duration += doorOpenTime

			e.Direction = drivers.MotorDirection(chooseDirection(ElevatorState{
				RequestMatrix: e.RequestMat,
				Floor:         e.Floor,
				Direction:     Dirn(e.Direction),
			}))

			if e.Direction == drivers.MD_Stop {
				return duration
			}
		}
		e.Floor += int(e.Direction)
		duration += travelTime
	}
}

func AssignElevator(requestFloor int, requestButton drivers.ButtonType, elevators map[string]SimulatedElevator) string {
	bestElevator := ""
	bestTime := math.MaxInt32 //Start with an extremely high cost

	for id, e := range elevators { //loop through all elevators
		copyE := e                                                              //Simulate this elevator, copying
		copyE.RequestMat.SetHallRequest(requestFloor, int(requestButton), true) //Add request temporarily
		timeToIdle := CalculateTimeToIdle(copyE)                                //Estimate time to finish

		if timeToIdle < bestTime { //picks the fastest elevator
			bestTime = timeToIdle
			bestElevator = id //alvays hold the id to the current best time
		}
	}
	return bestElevator //Return the ID of the best elevator
}

func FindBestElevator(order drivers.ButtonEvent, elevators []string) string {
	if len(elevators) == 0 {
		fmt.Println("[ERROR] No elevators available for assignment!")
		return ""
	}

	bestElevator := ""
	bestTime := math.MaxInt32

	for _, addr := range elevators {
		fmt.Printf("[DEBUG] Evaluating elevator at %s\n", addr)

		// Initialize a request matrix with the correct number of floors
		requestMatrix := NewRequestMatrix(4) // Use config.NumFloors if applicable

		estimatedTime := CalculateTimeToIdle(SimulatedElevator{
			Floor:      order.Floor,
			Direction:  drivers.MD_Stop,
			RequestMat: requestMatrix, // Correctly initialized
		})

		fmt.Printf("[DEBUG] Elevator %s estimated idle time: %d\n", addr, estimatedTime)

		if estimatedTime < bestTime {
			bestTime = estimatedTime
			bestElevator = addr
		}
	}

	if bestElevator == "" {
		fmt.Println("[ERROR] No valid elevator found!")
	}
	return bestElevator
}
