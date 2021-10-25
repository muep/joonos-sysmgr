package main

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
)

type loadavg struct {
	Loads         [3]float32
	TasksRunnable uint32
	TasksTotal    uint32
	LastPid       uint32
}

func loadavgLoad() (loadavg, error) {
	var la loadavg

	avgbytes, err := ioutil.ReadFile("/proc/loadavg")
	if err != nil {
		return la, err
	}

	avgtxt := strings.TrimSpace(string(avgbytes))
	pieces := strings.Split(avgtxt, " ")
	if len(pieces) != 5 {
		return la, fmt.Errorf("expected 5 pieces, got %v", pieces)
	}

	load1, err := strconv.ParseFloat(pieces[0], 32)
	if err == nil {
		la.Loads[0] = float32(load1)
	}

	load5, err := strconv.ParseFloat(pieces[1], 32)
	if err == nil {
		la.Loads[1] = float32(load5)
	}

	load15, err := strconv.ParseFloat(pieces[2], 32)
	if err == nil {
		la.Loads[2] = float32(load15)
	}

	procPieces := strings.Split(pieces[3], "/")
	if len(procPieces) == 2 {
		tasksRunnable, err := strconv.ParseUint(procPieces[0], 10, 32)
		if err == nil {
			la.TasksRunnable = uint32(tasksRunnable)
		}

		tasksTotal, err := strconv.ParseUint(procPieces[1], 10, 32)
		if err == nil {
			la.TasksTotal = uint32(tasksTotal)
		}
	}

	lastPid, err := strconv.ParseUint(pieces[4], 10, 32)
	if err == nil {
		la.LastPid = uint32(lastPid)
	}

	return la, nil
}
