// app/app.go
package app

import (
	"elevator-project/pkg/config"
	"elevator-project/pkg/drivers"
	"elevator-project/pkg/elevator"
	"elevator-project/pkg/message"
	"elevator-project/pkg/orders"
	"elevator-project/pkg/state"
	"elevator-project/pkg/transport"
	"fmt"
	"net"
	"time"
)

// HandleMessage processes incoming messages from peers.
func HandleMessage(msg message.Message, addr *net.UDPAddr) {
	switch msg.Type {
	case message.State:
		status := state.ElevatorStatus{
			ElevatorID:    msg.ElevatorID,
			State:         msg.StateData.State,
			Direction:     msg.StateData.Direction,
			CurrentFloor:  msg.StateData.CurrentFloor,
			TargetFloor:   msg.StateData.TargetFloor,
			RequestMatrix: msg.StateData.RequestMatrix,
			LastUpdated:   msg.StateData.LastUpdated,
		}
		masterStateStore.UpdateStatus(status)

	case message.Heartbeat:
		// Every elevator sends heartbeat.
		masterStateStore.UpdateHeartbeat(msg.ElevatorID)

	case message.MasterSlaveConfig:
		// Update our view of the current master.
		fmt.Printf("Received master config update: new master is elevator %d\n", msg.ElevatorID)
		CurrentMasterID = msg.ElevatorID
		if LocalElevatorID != msg.ElevatorID {
			IsMaster = false
		} else {
			IsMaster = true
		}

	default:
		// Handle other message types as needed.
	}
}


func StartHeartbeat(elevatorID int) {
	ticker := time.NewTicker(100 * time.Millisecond)
	seq := 1
	for range ticker.C {
		hbMsg := message.Message{
			Type:       message.Heartbeat,
			ElevatorID: elevatorID,
			Seq:        seq,
		}


		for i := 1; i < 4; i++ {
			if i != elevatorID {
				if err := transport.SendMessage(hbMsg, elevatorID, i); err != nil {
					fmt.Printf("Error sending state message (seq %d) to %s: %v\n", seq, config.UDPAckAddresses[i], err)
				} else {
					//fmt.Printf("Sent state message (seq: %d) to %s\n", seq, addr)
				}
			}
		}
		seq++
	}
}

func StartStateSender(e *elevator.Elevator, elevatorID int) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	seq := 1
	for range ticker.C {
		status := e.GetStatus()
		masterStateStore.UpdateStatus(status)
		stateMsg := message.Message{
			Type:       message.State,
			ElevatorID: status.ElevatorID,
			Seq:        seq,
			StateData: &message.ElevatorState{
				ElevatorID:    status.ElevatorID,
				State:         status.State,
				CurrentFloor:  status.CurrentFloor,
				TargetFloor:   status.TargetFloor,
				LastUpdated:   time.Now(),
				RequestMatrix: status.RequestMatrix,
			},
		}

		for i := 1; i < 4; i++ {
			if i != elevatorID {
				if err := transport.SendMessage(stateMsg, elevatorID, i); err != nil {
					fmt.Printf("Error sending state message (seq %d) to %s: %v\n", seq, config.UDPAckAddresses[i], err)
				} else {
					//fmt.Printf("Sent state message (seq: %d) to %s\n", seq, addr)
				}
			}
		}
		seq++
	}
}

// PrintStateStore prints the current state of all elevators periodically.
func PrintStateStore() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		fmt.Println("----- Current Elevator States -----")
		statuses := masterStateStore.GetAll()
		for id, status := range statuses {
			fmt.Printf("Elevator %d:\n", id)
			fmt.Printf("  ElevatorID   : %d\n", status.ElevatorID)
			fmt.Printf("  State        : %d\n", status.State)
			fmt.Printf("  Direction    : %d\n", status.Direction)
			fmt.Printf("  CurrentFloor : %d\n", status.CurrentFloor)
			fmt.Printf("  TargetFloor  : %d\n", status.TargetFloor)
			fmt.Printf("  LastUpdated  : %v\n", status.LastUpdated.Format("15:04:05"))
			fmt.Printf("  HallRequests : %+v\n", status.RequestMatrix.HallRequests)
			fmt.Printf("  CabRequests  : %+v\n", status.RequestMatrix.CabRequests)
			fmt.Println()
		}
		fmt.Println("-----------------------------------")
	}
}

// RunEventLoop remains unchanged.
func RunEventLoop(elevatorFSM *elevator.Elevator, reqMatrix *orders.RequestMatrix) {
	drvButtons := make(chan drivers.ButtonEvent)
	drvFloors := make(chan int)
	drvObstr := make(chan bool)
	drvStop := make(chan bool)

	go drivers.PollButtons(drvButtons)
	go drivers.PollFloorSensor(drvFloors)
	go drivers.PollObstructionSwitch(drvObstr)
	go drivers.PollStopButton(drvStop)

	for {
		select {
		case be := <-drvButtons:
			switch be.Button {
			case drivers.BT_Cab:
				_ = reqMatrix.SetCabRequest(be.Floor, true)
			case drivers.BT_HallUp:
				_ = reqMatrix.SetHallRequest(be.Floor, 0, true)
			case drivers.BT_HallDown:
				_ = reqMatrix.SetHallRequest(be.Floor, 1, true)
			}
			drivers.SetButtonLamp(be.Button, be.Floor, true)
		case <-drvFloors:
			elevatorFSM.UpdateElevatorState(elevator.EventArrivedAtFloor)
		case obstr := <-drvObstr:
			if obstr {
				elevatorFSM.UpdateElevatorState(elevator.EventDoorObstructed)
			} else {
				elevatorFSM.UpdateElevatorState(elevator.EventDoorReleased)
			}
		case <-drvStop:
			for f := 0; f < config.NumFloors; f++ {
				for b := drivers.ButtonType(0); b < 3; b++ {
					drivers.SetButtonLamp(b, f, false)
				}
			}
		}
	}
}

func StartMasterProcess(peerAddrs []string, elevatorFSM *elevator.Elevator) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if elevatorFSM.ElevatorID == 1 { // Master elevator check
			fmt.Println("[Master] Checking for unassigned orders...")

			unassignedOrders := orders.GetUnassignedOrders(elevatorFSM.GetRequestMatrix())
			fmt.Println("[Master] Unassigned Orders:", unassignedOrders)

			for _, order := range unassignedOrders {
				assignedElevator := orders.FindBestElevator(order, peerAddrs)
				fmt.Printf("[Master] Assigning order: Floor %d to Elevator %s\n", order.Floor, assignedElevator)

				if assignedElevator != "" {
					transport.SendOrderToElevator(order, assignedElevator)
				}
			}
		}
	}
}