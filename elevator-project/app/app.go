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

// ----- Global Role Management Variables -----
var LocalElevatorID int = 0 // Set from main.go via flag.
var IsMaster bool = false   // Will be set true if this elevator is the master.
var CurrentMasterID int = 1 // Initially, elevator 1 is the master.

// masterStateStore holds the status of all elevators.
var masterStateStore = state.NewStore()

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

// StartHeartbeat sends heartbeat messages periodically.
// Now, every elevator sends heartbeat, regardless of role.
func StartHeartbeat(peerAddrs []string, elevatorID int) {
	ticker := time.NewTicker(100 * time.Millisecond)
	seq := 1
	for range ticker.C {
		hbMsg := message.Message{
			Type:       message.Heartbeat,
			ElevatorID: elevatorID,
			Seq:        seq,
		}
		for _, addr := range peerAddrs {
			if err := transport.SendMessage(hbMsg, addr); err != nil {
				fmt.Printf("Error sending heartbeat (seq %d) to %s: %v\n", seq, addr, err)
			}
		}
		seq++
	}
}

// StartStateSender broadcasts state updates periodically.
func StartStateSender(e *elevator.Elevator, peerAddrs []string) {
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
		for _, addr := range peerAddrs {
			if err := transport.SendMessage(stateMsg, addr); err != nil {
				fmt.Printf("Error sending state message (seq %d) to %s: %v\n", seq, addr, err)
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

// MonitorElevatorHeartbeats runs on the master and checks for stale heartbeats
// from any elevator. If an elevator's heartbeat is older than 5 seconds,
// it calls ReassignOrders to take over its hall calls.
func MonitorElevatorHeartbeats() {
	// This function should run only on the master.
	if !IsMaster {
		return
	}
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		statuses := masterStateStore.GetAll()
		for id, status := range statuses {
			// Skip self.
			if id == LocalElevatorID {
				continue
			}
			if time.Since(status.LastUpdated) > 5*time.Second {
				fmt.Printf("Elevator %d heartbeat stale. Reassigning its orders.\n", id)
				ReassignOrders(status)
				// Optionally, you might remove or mark the elevator as failed.
			}
		}
	}
}

// ReassignOrders takes the orders from a failed elevator and merges them
// into the global (master's) request matrix.
// (For hall calls only—per project requirements, cab calls can be handled separately.)
func ReassignOrders(failedStatus state.ElevatorStatus) {
	// For every floor, check if there is a hall request from the failed elevator.
	for floor, hallRequests := range failedStatus.RequestMatrix.HallRequests {
		for dir, active := range hallRequests {
			if active {
				fmt.Printf("Reassigning hall request at floor %d, direction %d from failed elevator %d.\n", floor, dir, failedStatus.ElevatorID)
				// Here you would merge this into your master's local RequestMatrix.
				// For example:
				// globalRequestMatrix.SetHallRequest(floor, dir, true)
			}
		}
	}
	// Optionally, you can handle cab requests if desired.
}

// MonitorMasterHeartbeat runs on every non-master elevator to detect a stale master heartbeat.
// It uses a tie-breaker: if this elevator has the smallest ID among all active slaves, it promotes itself.
func MonitorMasterHeartbeat(peerAddrs []string) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		statuses := masterStateStore.GetAll()
		masterStatus, exists := statuses[CurrentMasterID]
		if !exists || time.Since(masterStatus.LastUpdated) > 5*time.Second {
			// Determine the candidate for promotion among all non-master elevators.
			candidate := LocalElevatorID
			for id, status := range statuses {
				// Consider only non-master elevators that have an up-to-date heartbeat.
				if id != CurrentMasterID && time.Since(status.LastUpdated) <= 5*time.Second && id < candidate {
					candidate = id
				}
			}
			// Only the candidate (with the smallest ID) promotes itself.
			if LocalElevatorID == candidate {
				fmt.Println("Master heartbeat stale. Promoting self to master.")
				PromoteToMaster(LocalElevatorID, peerAddrs)
				// Once promoted, break out of the monitoring loop.
				break
			}
		}
	}
}

// PromoteToMaster promotes this elevator to master and broadcasts the new configuration.
func PromoteToMaster(localID int, peerAddrs []string) {
	IsMaster = true
	CurrentMasterID = localID
	fmt.Printf("Elevator %d is now promoted to master.\n", localID)
	configMsg := message.Message{
		Type:       message.MasterSlaveConfig,
		ElevatorID: localID,
		Seq:        0,
	}
	for _, addr := range peerAddrs {
		if err := transport.SendMessage(configMsg, addr); err != nil {
			fmt.Printf("Error broadcasting master config to %s: %v\n", addr, err)
		}
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
