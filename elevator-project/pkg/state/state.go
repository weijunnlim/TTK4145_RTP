package state

import (
	//"elevator-project/pkg/elevator"
	"elevator-project/pkg/orders"
	"sync"
	"time"
)

// ElevatorStatus holds the relevant status information for an elevator.
type ElevatorStatus struct {
	ElevatorID   int           
	State        int
	Direction	 int //Should be changed to driver.MD?
	CurrentFloor int           
	TargetFloor  int           
	LastUpdated  time.Time     
	RequestMatrix orders.RequestMatrix
}

// Store manages the state for multiple elevators in a thread-safe way.
type Store struct {
	mu        sync.RWMutex
	elevators map[int]ElevatorStatus
}

// NewStore creates and returns a new state store.
func NewStore() *Store {
	return &Store{
		elevators: make(map[int]ElevatorStatus),
	}
}

// Update inserts or updates the status of an elevator.
func (s *Store) Update(status ElevatorStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	status.LastUpdated = time.Now()
	s.elevators[status.ElevatorID] = status
}

// Get retrieves the state of a specific elevator.
func (s *Store) Get(elevatorID int) (ElevatorStatus, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status, exists := s.elevators[elevatorID]
	return status, exists
}

// GetAll returns a copy of all elevator states.
func (s *Store) GetAll() map[int]ElevatorStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	copy := make(map[int]ElevatorStatus)
	for id, status := range s.elevators {
		copy[id] = status
	}
	return copy
}
