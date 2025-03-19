package HRA

import (
	"elevator-project/pkg/state"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

type HRAElevState struct {
	Behavior    string `json:"behaviour"`
	Floor       int    `json:"floor"`
	Direction   string `json:"direction"`
	CabRequests []bool `json:"cabRequests"`
}

type HRAInput struct {
	HallRequests [][2]bool               `json:"hallRequests"`
	States       map[string]HRAElevState `json:"states"`
}

func HRARun(st *state.Store) (map[string][][2]bool, error) {
	allElevators := st.GetAll()
	statesMap := make(map[string]HRAElevState)
	for id, elev := range allElevators {
		dirString := directionIntToString(elev.Direction)
		stateString := stateIntToString(elev.State)
		statesMap[strconv.Itoa(id)] = HRAElevState{
			Behavior:    stateString,
			Floor:       elev.CurrentFloor,
			Direction:   dirString,
			CabRequests: elev.RequestMatrix.CabRequests,
		}
	}

	input := HRAInput{
		HallRequests: st.HallRequests,
		States:       statesMap,
	}

	PrintHRAInput(input)
	jsonBytes, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("json.Marshal error: %v", err)
	}

	// Select the executable and command based on the OS.
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("../hall_request_assigner.exe", "-i", string(jsonBytes))
	case "darwin":
		cmd = exec.Command("wine", "../hall_request_assigner.exe", "-i", string(jsonBytes))
	case "linux":
		cmd = exec.Command("../hall_request_assigner", "-i", string(jsonBytes))
	default:
		return nil, fmt.Errorf("OS not supported")
	}

	ret, err := cmd.CombinedOutput()
	rawOutput := string(ret)
	fmt.Println("Raw output from hall_request_assigner:", rawOutput)
	if err != nil {
		return nil, fmt.Errorf("exec.Command error: %v, output: %s", err, rawOutput)
	}

	// Extract the JSON part by finding the first '{'
	jsonStart := strings.Index(rawOutput, "{")
	if jsonStart == -1 {
		return nil, fmt.Errorf("could not find JSON in output: %s", rawOutput)
	}
	cleanOutput := rawOutput[jsonStart:]

	output := make(map[string][][2]bool)
	err = json.Unmarshal([]byte(cleanOutput), &output)
	if err != nil {
		return nil, fmt.Errorf("json.Unmarshal error: %v", err)
	}

	fmt.Println("Master sending the output:")
	for k, v := range output {
		fmt.Printf("%6v : %+v\n", k, v)
	}

	return output, nil
}

func directionIntToString(dir int) string {
	switch dir {
	case 0:
		return "stop"
	case -1:
		return "down"
	case 1:
		return "up"
	default:
		return "Unknown button"
	}
}

func stateIntToString(state int) string {
	switch state {
	case 0:
		return "idle"
	case 1:
		return "moving"
	case 2:
		return "moving"
	case 3:
		return "doorOpen"
	case 4:
		return "doorOpen"
	default:
		return "Unknown button"
	}
}

func PrintHRAInput(input HRAInput) {
	fmt.Println("HRA Input:")

	fmt.Println("Hall Requests:")
	for floor, req := range input.HallRequests {
		// Each req is an array of two booleans: [Up, Down]
		fmt.Printf("  Floor %d: Up: %t, Down: %t\n", floor, req[0], req[1])
	}

	fmt.Println("States:")
	for id, state := range input.States {
		fmt.Printf("  Elevator %s:\n", id)
		fmt.Printf("    Behavior   : %s\n", state.Behavior)
		fmt.Printf("    Floor      : %d\n", state.Floor)
		fmt.Printf("    Direction  : %s\n", state.Direction)
		fmt.Printf("    CabRequests: %v\n", state.CabRequests)
	}
}
