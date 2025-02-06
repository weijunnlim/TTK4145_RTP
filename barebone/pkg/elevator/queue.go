package elevator

import (
	"barebone/pkg/drivers"
	"barebone/pkg/utils"
	"fmt"
)

type Order struct {
	ID    int
	Floor int
}

func QueueManager(newOrders <-chan drivers.ButtonEvent, elevatorReady <-chan bool, sendOrder chan<- drivers.ButtonEvent) {
	var queue []drivers.ButtonEvent
	isElevatorReady := false

	for {
		select {
		// Mottar ny bestilling
		case o := <-newOrders:
			fmt.Printf("[QueueManager] Ny bestilling lagt i køen. Floor: %d, Button: %s\n", o.Floor, utils.ButtonTypeToString(o.Button))
			queue = append(queue, o)
			//fmt.Printf("[QueueManager] Queuelenght is now: %d\n", len(queue))
			drivers.SetButtonLamp(o.Button, o.Floor, true)
			if isElevatorReady {
				next := queue[0]
				queue = queue[1:]
				fmt.Printf("[QueueManager] Sender bestilling til heisen: %#v\n", next)
				sendOrder <- next
				isElevatorReady = false
			}

		// Heisen signaliserer at den er klar
		case <-elevatorReady:
			isElevatorReady = true
			if len(queue) > 0 {
				next := queue[0]
				queue = queue[1:]
				fmt.Printf("[QueueManager] Sender bestilling til heisen: %#v\n", next)
				sendOrder <- next
				isElevatorReady = false
			} else {
				fmt.Println("[QueueManager] Ingen ordre i kø.")
			}
		}
	}
}
