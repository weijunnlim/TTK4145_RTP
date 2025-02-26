package transport

import (
	"elevator-project/pkg/message"
	"fmt"
	"net"
)

// StartServer starts a UDP server listening on listenAddr.
// The handleMsg callback is invoked whenever a valid Message is received.
func StartServer(listenAddr string, handleMsg func(msg message.Message, addr *net.UDPAddr)) error {
	addr, err := net.ResolveUDPAddr("udp", listenAddr)
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	buf := make([]byte, 1024)
	fmt.Printf("Server listening on %s\n", listenAddr)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("Error reading UDP message:", err)
			continue
		}
		msg, err := message.Unmarshal(buf[:n])
		if err != nil {
			fmt.Println("Error unmarshaling message:", err)
			continue
		}

		//fmt.Printf("Received message from %s: %+v\n", remoteAddr, msg)

		// For order messages, send an ACK back.
		if msg.Type == message.Order {
			sendAck(conn, remoteAddr, msg.Seq, 0)
		}

		if msg.Type == message.Heartbeat {
			//Should update a "last seen" variable
		}

		// Invoke the provided callback to handle the message.
		handleMsg(msg, remoteAddr)
	}
}

// sendAck sends an acknowledgment for a received order message.
func sendAck(conn *net.UDPConn, addr *net.UDPAddr, seq int, elevatorID int) {
	ackMsg := message.Message{
		Type:       message.Ack,
		ElevatorID: elevatorID,
		AckSeq:     seq,
	}
	data, err := message.Marshal(ackMsg)
	if err != nil {
		fmt.Println("Error marshaling ACK message:", err)
		return
	}
	_, err = conn.WriteToUDP(data, addr)
	if err != nil {
		fmt.Println("Error sending ACK message:", err)
	} else {
		fmt.Printf("Sent ACK for seq %d to %s\n", seq, addr.String())
	}
}