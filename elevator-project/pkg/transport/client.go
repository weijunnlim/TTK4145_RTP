package transport

import (
	"fmt"
	"net"
	"sync"
	"time"

	"elevator-project/pkg/config"
	"elevator-project/pkg/message"
)

// ackChannels maps message sequence numbers to channels that signal when an ACK is received.
var (
	ackChannels      = make(map[int]chan bool)
	ackChannelsMutex = sync.Mutex{}
)

// StartAckListener sets up a UDP listener on the provided address (e.g., "0.0.0.0:9001")
// that exclusively handles ACK messages.
func StartAckListener(elevatorID int) error {
	addr, err := net.ResolveUDPAddr("udp", config.UDPAckAddresses[elevatorID])
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	fmt.Printf("ACK Listener running on %s\n", config.UDPAckAddresses[elevatorID])

	// Run the ACK listener in a separate goroutine.
	go func() {
		buf := make([]byte, 1024)
		for {
			n, remoteAddr, err := conn.ReadFromUDP(buf)
			if err != nil {
				fmt.Println("Error reading ACK message:", err)
				continue
			}

			ackMsg, err := message.Unmarshal(buf[:n])
			if err != nil {
				fmt.Println("Error unmarshaling ACK message:", err)
				continue
			}

			// Check if the message is an ACK.
			if ackMsg.Type == message.Ack {
				// Look up the channel for this sequence number.
				ackChannelsMutex.Lock()
				ch, exists := ackChannels[ackMsg.AckSeq]
				ackChannelsMutex.Unlock()

				if exists {
					fmt.Printf("Received ACK for seq %d from %s\n", ackMsg.AckSeq, remoteAddr.String())
					// Signal the waiting sender.
					ch <- true
				} else {
					fmt.Printf("Received ACK for unknown seq %d\n", ackMsg.AckSeq)
				}
			}
		}
	}()
	return nil
}

// SendMessage sends a message to peerAddr. For order messages, it repeatedly sends
// the message until the dedicated ACK listener (running on ackListenAddr) signals that
// the ACK with a matching sequence has been received.
func SendMessage(msg message.Message, sendingElevatorID int, receivingElevatorID int) error {
	remoteAddr, err := net.ResolveUDPAddr("udp", config.UDPAddresses[receivingElevatorID])
	if err != nil {
		return err
	}

	// Bind the local address to the ACK listener's address.
	localAddr, err := net.ResolveUDPAddr("udp", config.UDPAckAddresses[sendingElevatorID])
	if err != nil {
		return err
	}

	conn, err := net.DialUDP("udp", localAddr, remoteAddr)
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
		// Create a channel for this message's ACK.
		ackCh := make(chan bool, 1)
		ackChannelsMutex.Lock()
		ackChannels[msg.Seq] = ackCh
		ackChannelsMutex.Unlock()

		// Ensure the channel is cleaned up once done.
		defer func() {
			ackChannelsMutex.Lock()
			delete(ackChannels, msg.Seq)
			ackChannelsMutex.Unlock()
		}()

		// Keep sending the order until the correct ACK is received.
		for {
			_, err = conn.Write(data)
			if err != nil {
				fmt.Println("Error sending order message:", err)
			} else {
				fmt.Printf("Order message sent to %s, waiting for ACK on %s...\n", config.UDPAddresses[receivingElevatorID], config.UDPAckAddresses[sendingElevatorID])
			}

			// Wait for the ACK signal with a timeout before retrying.
			select {
			case <-ackCh:
				fmt.Printf("Received ACK for seq %d, stopping retries\n", msg.Seq)
				return nil
			case <-time.After(100 * time.Millisecond):
				// Timeout elapsed; retry sending the message.
			}
		}
	default:
		// For non-order messages, just send once.
		_, err = conn.Write(data)
		if err != nil {
			return err
		}
	}

	return nil
}
