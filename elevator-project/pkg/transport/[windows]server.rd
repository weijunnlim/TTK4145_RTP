package transport

import (
	"context"
	"elevator-project/pkg/message"
	"fmt"
	"net"
	"syscall"
)

// StartServer starts a UDP server on listenAddr with socket reuse options enabled.
// The handleMsg callback is invoked whenever a valid Message is received.
func StartServer(listenAddr string, handleMsg func(msg message.Message, addr *net.UDPAddr)) error {
	conn, err := listenUDPWithReuse(listenAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	fmt.Printf("Server listening on %s\n", listenAddr)

	buf := make([]byte, 1024)
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

		// For order messages, send an ACK back.
		if msg.Type == message.Order {
			sendAck(conn, remoteAddr, msg.Seq, 0)
		}

		// For heartbeat messages, you might update a "last seen" variable here.
		if msg.Type == message.Heartbeat {
			// Update last seen logic here.
		}

		// Handle the message using the provided callback.
		handleMsg(msg, remoteAddr)
	}
}

// listenUDPWithReuse creates a UDP connection with SO_REUSEADDR and SO_REUSEPORT options set.
func listenUDPWithReuse(address string) (*net.UDPConn, error) {
	lc := net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			var controlErr error
			err := c.Control(func(fd uintptr) {
				// Enable address reuse.
				if err := syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
					controlErr = err
					return
				}
				// Enable port reuse (note: may not be supported on all platforms).
				if err := syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEPORT, 1); err != nil {
					controlErr = err
					return
				}
			})
			if err != nil {
				return err
			}
			return controlErr
		},
	}

	// Use ListenPacket to create a packet connection.
	pc, err := lc.ListenPacket(context.Background(), "udp", address)
	if err != nil {
		return nil, err
	}

	// Convert the PacketConn to a UDPConn.
	return pc.(*net.UDPConn), nil
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
