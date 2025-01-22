// Translating `con_load.h` into Go as `pkg/config/config.go`

package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	DoorOpenDurationS float64
	ClearRequestVariant string
}

func LoadConfig(filename string) (Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return Config{}, err
	}
	defer file.Close()

	config := Config{}
	var lines []string
	buffer := make([]byte, 1024)
	for {
		n, err := file.Read(buffer)
		if err != nil {
			break
		}
		lines = append(lines, strings.Split(string(buffer[:n]), "\n")...)
	}

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		switch fields[0] {
		case "doorOpenDuration_s":
			config.DoorOpenDurationS, _ = strconv.ParseFloat(fields[1], 64)
		case "clearRequestVariant":
			config.ClearRequestVariant = fields[1]
		}
	}

	return config, nil
}
