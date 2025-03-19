package message

import (
	"elevator-project/pkg/drivers"
	"elevator-project/pkg/orders"
	"sync"
	"time"
)

type MessageType int

const (
	State           MessageType = iota // Full worldview
	ButtonEvent                        // All types of buttonpresses
	OrderDelegation                    // Master delegates an order to a specific elevator
	CompletedOrder
	Ack
	Heartbeat
	MasterSlaveConfig // ???
	Promotion         // Promotion msg letting other elevators know that a new elevator is master?
)

type ElevatorState struct {
	ElevatorID      int
	State           int
	Direction       int
	CurrentFloor    int
	TravelDirection int
	LastUpdated     time.Time
	RequestMatrix   orders.RequestMatrix
}

type Message struct {
	Type        MessageType          `json:"type"`
	ElevatorID  int                  `json:"elevatorID"`
	MsgID       int                  `json:"msgID"`
	StateData   *ElevatorState       `json:"stateData,omitempty"`
	ButtonEvent drivers.ButtonEvent  `json:"buttonEvent,omitempty"`
	OrderData   map[string][][2]bool `json:"orderData,omitempty"`
	AckID       int                  `json:"ackID,omitempty"`
}

type MsgID struct {
	mu sync.Mutex
	id int
}

func (mc *MsgID) Next() int {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	currentID := mc.id
	mc.id++
	return currentID
}

func (m *MsgID) Get() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.id
}
