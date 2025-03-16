package app

import (
	"elevator-project/pkg/config"
	"elevator-project/pkg/drivers"
	"strconv"
)

func convertOrderDataToButtonEvents(orderData map[string][][2]bool) []drivers.ButtonEvent {
	var events []drivers.ButtonEvent
	orders := orderData[strconv.Itoa(config.ElevatorID)]

	for floor, calls := range orders {
		if calls[0] { // Hall up call
			events = append(events, drivers.ButtonEvent{Floor: floor, Button: drivers.BT_HallUp})
		}
		if calls[1] { // Hall down call
			events = append(events, drivers.ButtonEvent{Floor: floor, Button: drivers.BT_HallDown})
		}
	}

	return events
}
