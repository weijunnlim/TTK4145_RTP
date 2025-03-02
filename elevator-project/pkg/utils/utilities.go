package utils

import (
	"elevator-project/pkg/config"
	"elevator-project/pkg/drivers"
)

func ButtonTypeToString(b drivers.ButtonType) string {
	switch b {
	case drivers.BT_HallUp:
		return "Hallcall up"
	case drivers.BT_HallDown:
		return "Hallcall down"
	case drivers.BT_Cab:
		return "Cab call"
	default:
		return "Unknown button"
	}
}

func GetOtherElevatorAddresses(elevatorID int) []string {
	others := []string{}
	for id, address := range config.UDPAddresses {
		if id != elevatorID {
			others = append(others, address)
		}
	}
	return others
}
