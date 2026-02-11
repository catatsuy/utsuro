package server

import (
	"fmt"
	"strconv"
	"strings"
)

type request struct {
	cmd    string
	args   []string
	isQuit bool
}

func parseLine(line string) (request, error) {
	line = strings.TrimSuffix(line, "\r\n")
	line = strings.TrimSuffix(line, "\n")
	if line == "" {
		return request{}, fmt.Errorf("empty command")
	}
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return request{}, fmt.Errorf("empty command")
	}

	cmd := strings.ToLower(fields[0])
	if cmd == "quit" {
		return request{cmd: cmd, isQuit: true}, nil
	}

	return request{cmd: cmd, args: fields[1:]}, nil
}

func parseSetArgs(args []string) (key string, flags uint32, bytesN int, err error) {
	if len(args) != 4 {
		return "", 0, 0, fmt.Errorf("set requires 4 arguments")
	}
	key = args[0]

	parsedFlags, err := strconv.ParseUint(args[1], 10, 32)
	if err != nil {
		return "", 0, 0, fmt.Errorf("invalid flags")
	}
	flags = uint32(parsedFlags)

	if _, err := strconv.ParseInt(args[2], 10, 64); err != nil {
		return "", 0, 0, fmt.Errorf("invalid exptime")
	}

	parsedBytes, err := strconv.ParseInt(args[3], 10, 32)
	if err != nil || parsedBytes < 0 {
		return "", 0, 0, fmt.Errorf("invalid bytes")
	}
	return key, flags, int(parsedBytes), nil
}

func parseDeltaArgs(args []string) (key string, delta uint64, err error) {
	if len(args) != 2 {
		return "", 0, fmt.Errorf("requires key and delta")
	}
	delta, err = strconv.ParseUint(args[1], 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("invalid delta")
	}
	return args[0], delta, nil
}
