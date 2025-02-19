package app

import (
	"elevator-project/pkg/elevator"
	"elevator-project/pkg/message"
	"elevator-project/pkg/state"
	"elevator-project/pkg/transport"
	"fmt"
	"net"
	"time"
)

var masterStateStore = state.NewStore()

func HandleMessage(msg message.Message, addr *net.UDPAddr) {
	switch msg.Type {
	case message.State:
		// Convert the received message to ElevatorStatus.
		// This assumes that msg.StateData contains fields that match ElevatorStatus.
		// You might need to do some mapping if your message struct differs.
		var status state.ElevatorStatus
		// Option 1: If your message carries JSON that's directly in ElevatorStatus format:
		// err := json.Unmarshal(receivedData, &status)
		// Option 2: If you manually construct it from the message fields:
		
		status = state.ElevatorStatus{
			ElevatorID:  	 msg.ElevatorID,
			State:        	 msg.StateData.State,
			Direction: 		 msg.StateData.Direction,
			CurrentFloor:	 msg.StateData.CurrentFloor,
			TargetFloor: 	 msg.StateData.TargetFloor,
			RequestMatrix:   msg.StateData.RequestMatrix,
			LastUpdated:     msg.StateData.LastUpdated,
		}
		
		// Update the master's store.
		masterStateStore.Update(status)


	case message.Heartbeat:
		currentTime := time.Now().Format("15:04:05")
		fmt.Printf("Handler: Heartbeat from elevator %d at %s\n", msg.ElevatorID, currentTime)
	default:
		//fmt.Printf("Handler received message from %s: %+v\n", addr.String(), msg)
	}
}


func StartHeartbeat(peerAddr string, elevatorID int) {
	ticker := time.NewTicker(5 * time.Second)
	seq := 1
	for range ticker.C {
		hbMsg := message.Message{
			Type:       message.Heartbeat,
			ElevatorID: elevatorID,
			Seq:        seq,
		}
		err := transport.SendMessage(hbMsg, peerAddr)
		if err != nil {
			fmt.Println("Error sending heartbeat message:", err)
		} else {
			fmt.Printf("Heartbeat sent (seq: %d) to %s\n", seq, peerAddr)
		}
		seq++
	}
}

// StartStateSender packages and sends the elevator's state every 5 seconds to the given peer address.
// e: pointer to your elevator instance.
// peerAddr: the UDP address of the master (or target peer).
func StartStateSender(e *elevator.Elevator, peerAddr string) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	seq := 1
	for range ticker.C {
		// Retrieve the current status of the elevator.
		status := e.GetStatus()

		// Create a state message using the shared message format.
		stateMsg := message.Message{
			Type:       message.State,
			ElevatorID: status.ElevatorID,
			Seq:        seq,
			StateData: &message.ElevatorState{
				ElevatorID:    status.ElevatorID,
				State:         status.State,         // Assuming status.State is of type model.ElevatorState.
				CurrentFloor:  status.CurrentFloor,
				TargetFloor:   status.TargetFloor,
				LastUpdated:   time.Now(),           // You can also use status.LastUpdated if maintained.
				RequestMatrix: status.RequestMatrix, // Assuming this field is serializable.
			},
		}

		// Send the state message via UDP.
		if err := transport.SendMessage(stateMsg, peerAddr); err != nil {
			fmt.Printf("Error sending state message (seq %d): %v\n", seq, err)
		} else {
			fmt.Printf("Sent state message (seq: %d) to %s\n", seq, peerAddr)
		}
		seq++
	}
}
