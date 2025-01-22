// Translating `elevator_io_device.c` into Go as `pkg/drivers/elevator_io_device.go`

package drivers

import (
	"elevator-control/pkg/elevio"
)

type ElevatorInputDevice struct {
	FloorSensor   func() int
	RequestButton func(floor int, button ButtonType) bool
	StopButton    func() bool
	Obstruction   func() bool
}

type ElevatorOutputDevice struct {
	SetButtonLamp      func(floor int, button ButtonType, value bool)
	SetMotorDirection  func(direction MotorDirection)
	SetDoorOpenLamp    func(value bool)
	SetFloorIndicator  func(floor int)
}

func GetInputDevice() ElevatorInputDevice {
	return ElevatorInputDevice{
		FloorSensor:   hardware.GetFloorSensorSignal,
		RequestButton: func(floor int, button ButtonType) bool { return hardware.GetButtonSignal(button, floor) },
		StopButton:    hardware.GetStopSignal,
		Obstruction:   hardware.GetObstructionSignal,
	}
}

func GetOutputDevice() ElevatorOutputDevice {
	return ElevatorOutputDevice{
		SetButtonLamp: func(floor int, button ButtonType, value bool) {
			hardware.SetButtonLamp(button, floor, value)
		},
		SetMotorDirection: func(direction MotorDirection) {
			hardware.SetMotorDirection(direction)
		},
		SetDoorOpenLamp: func(value bool) {
			hardware.SetDoorOpenLamp(value)
		},
		SetFloorIndicator: func(floor int) {
			hardware.SetFloorIndicator(floor)
		},
	}
}
