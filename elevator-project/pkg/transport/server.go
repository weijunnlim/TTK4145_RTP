package transport

import (
	"elevator-project/pkg/config"
	"elevator-project/pkg/message"
	"fmt"
	"net"
)

func StartServer(elevatorID int, handleMsg func(msg message.Message, addr *net.UDPAddr)) error {
	addr, err := net.ResolveUDPAddr("udp", config.UDPAddresses[elevatorID])
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	buf := make([]byte, 1024)
	fmt.Printf("Server listening on %s\n", config.UDPAddresses[elevatorID])
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

		// If it's an ACK message, ignore it.
		if msg.Type == message.Ack {
			continue
		}

		// For order messages, send an ACK back.
		if msg.Type == message.Order {
			//sendAck(conn, remoteAddr, msg.Seq, 0)
			ackMsg := message.Message{
				Type:       message.Ack,
				ElevatorID: elevatorID,
				AckSeq:     msg.Seq,
			}
			SendMessage(ackMsg, elevatorID, msg.ElevatorID)
		}

		// Only non-ACK messages are passed to the general handler.
		handleMsg(msg, remoteAddr)
	}
}