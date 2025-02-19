package message

import(
	"encoding/json"
	"elevator-project/pkg/orders"
	//"elevator-project/pkg/state"
	"time"
)

type MessageType int

const (
	State MessageType = iota //worldView
	Order
	Ack
	Heartbeat
	MasterSlaveConfig //to configure whos backup etc
)

type ElevatorState struct {
	ElevatorID    int                `json:"elevatorID"`
	State         int				 `json:"state"` // Shared type from model package.
	Direction     int				 `json:"direction"`
	CurrentFloor  int                `json:"currentFloor"`
	TargetFloor   int                `json:"targetFloor"`
	LastUpdated   time.Time          `json:"lastUpdated"`
	RequestMatrix orders.RequestMatrix `json:"requestMatrix"`
}

type OrderData struct {
	Floor      int    `json:"floor"`
	ButtonType string `json:"button_type"`
}


type Message struct {
	Type       MessageType    `json:"type"`
	ElevatorID int            `json:"elevator_id"`
	Seq        int            `json:"seq"`
	StateData  *ElevatorState `json:"state_data,omitempty"`
	OrderData  *OrderData     `json:"order_data,omitempty"`
	AckSeq     int            `json:"ack_seq,omitempty"`
}

// Marshal converts a Message into JSON.
func Marshal(msg Message) ([]byte, error) {
	return json.Marshal(msg)
}

// Unmarshal converts JSON data into a Message.
func Unmarshal(data []byte) (Message, error) {
	var msg Message
	err := json.Unmarshal(data, &msg)
	return msg, err
}
