package elevator

import (
	"elevator-project/pkg/drivers"
	"elevator-project/pkg/orders"
)

// OptimalAssignment evaluates all pending orders in the request matrix
// and returns the order with the lowest cost. The cost function takes into
// account the elevator's current floor and direction.
func OptimalAssignment(rm *orders.RequestMatrix, currentFloor int, currentDirection drivers.MotorDirection) (drivers.ButtonEvent, bool) {
	var bestOrder drivers.ButtonEvent
	bestCost := int(^uint(0) >> 1)
	found := false

	for floor, active := range rm.CabRequests {
		if active {
			cost := computeCost(floor, drivers.BT_Cab, currentFloor, currentDirection)
			if cost < bestCost {
				bestCost = cost
				bestOrder = drivers.ButtonEvent{Floor: floor, Button: drivers.BT_Cab}
				found = true
			}
		}
	}

	for floor, reqs := range rm.HallRequests {
		for dir, active := range reqs {
			if active {
				var button drivers.ButtonType
				if dir == 0 {
					button = drivers.BT_HallUp
				} else {
					button = drivers.BT_HallDown
				}
				cost := computeCost(floor, button, currentFloor, currentDirection)
				if cost < bestCost {
					bestCost = cost
					bestOrder = drivers.ButtonEvent{Floor: floor, Button: button}
					found = true
				}
			}
		}
	}

	return bestOrder, found
}

// computeCost calculates the "cost" to serve a request at requestFloor with the given button.
// The base cost is the absolute distance. If the request is in the direction the elevator is moving,
// no penalty is applied; otherwise, a high penalty is added.
func computeCost(requestFloor int, requestButton drivers.ButtonType, currentFloor int, currentDirection drivers.MotorDirection) int {
	baseCost := abs(currentFloor - requestFloor)
	penalty := 1000
	switch currentDirection {
	case drivers.MD_Up:
		if requestFloor >= currentFloor && (requestButton == drivers.BT_HallUp || requestButton == drivers.BT_Cab) {
			return baseCost
		}
		return baseCost + penalty
	case drivers.MD_Down:
		if requestFloor <= currentFloor && (requestButton == drivers.BT_HallDown || requestButton == drivers.BT_Cab) {
			return baseCost
		}
		return baseCost + penalty
	default: // MD_Stop or unknown: no directional bias.
		return baseCost
	}
}
