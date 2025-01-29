package elevator

import (
	"elevator-project/pkg/drivers"
	"fmt"
	"sync"
	"time"
)

// OrderQueue represents a queue for handling button events
type OrderQueue struct {
	Queue         []drivers.ButtonEvent
	Mutex         sync.Mutex
	AddChan       chan drivers.ButtonEvent
	NextOrder     chan int
	ElevatorDone  chan bool // Signal when elevator is done processing an order
}

// NewOrderQueue initializes an empty OrderQueue
func NewOrderQueue() *OrderQueue {
	q := &OrderQueue{
		Queue:        []drivers.ButtonEvent{},
		AddChan:      make(chan drivers.ButtonEvent, 10),
		NextOrder:    make(chan int, 1),
		ElevatorDone: make(chan bool, 1), // Signal FSM when elevator is done
	}
	go q.queueWorker()
	go q.queuePrinter()
	go q.processNextOrder() // This now waits for the elevator to finish processing
	return q
}

// queueWorker runs in a separate goroutine to handle incoming orders
func (q *OrderQueue) queueWorker() {
	for event := range q.AddChan {
		q.Mutex.Lock()
		q.Queue = append(q.Queue, event)
		q.Mutex.Unlock()
	}
}

// processNextOrder waits for the elevator to finish before dequeuing the next order
func (q *OrderQueue) processNextOrder() {
	for {
		<-q.ElevatorDone // Wait for FSM to signal that the elevator is done
		q.Mutex.Lock()
		if len(q.Queue) > 0 {
			order := q.Queue[0]
			q.Queue = q.Queue[1:]
			q.Mutex.Unlock()
			q.NextOrder <- order.Floor // Send the next floor request to FSM
		} else {
			q.Mutex.Unlock()
		}
	}
}

// Enqueue sends a new button event to the queue worker
func (q *OrderQueue) Enqueue(event drivers.ButtonEvent) {
	q.AddChan <- event
}

// PrintQueue prints the current state of the queue for debugging
func (q *OrderQueue) PrintQueue() {
	q.Mutex.Lock()
	defer q.Mutex.Unlock()
	fmt.Println("Current Queue:")
	for i, event := range q.Queue {
		fmt.Printf("%d: Floor %d, Button %d\n", i, event.Floor, event.Button)
	}
}

// queuePrinter runs a goroutine that prints the queue every 5 seconds
func (q *OrderQueue) queuePrinter() {
	for {
		time.Sleep(5 * time.Second)
		q.PrintQueue()
	}
}
