// transport/client.go
package transport

import (
	"fmt"
	"net"
	"time"
	"elevator-project/pkg/message"
)

// SendMessage sends a Message to the specified peer address.
// For order messages, it performs a simple retransmission loop.
// Heartbeat messages are sent once, as they are lightweight.
func SendMessage(msg message.Message, peerAddr string) error {
	addr, err := net.ResolveUDPAddr("udp", peerAddr)
	if err != nil {
		return err
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	data, err := message.Marshal(msg)
	if err != nil {
		return err
	}

	switch msg.Type {
	case message.Order:
		// For order messages, attempt multiple sends.
		maxRetries := 3
		for i := 0; i < maxRetries; i++ {
			_, err = conn.Write(data)
			if err != nil {
				fmt.Println("Error sending order message:", err)
			} else {
				fmt.Printf("Order message sent to %s, attempt %d\n", peerAddr, i+1)
			}
			// In a complete system, wait for an ACK before breaking out.
			time.Sleep(100 * time.Millisecond)
		}
	default:
		// For state updates and heartbeat messages, one send is sufficient.
		_, err = conn.Write(data)
		if err != nil {
			return err
		}
	}

	return nil
}