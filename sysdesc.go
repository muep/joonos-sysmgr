package main

import (
	"os"
)

var sysdescSysmgrversion string = "0.0"

type sysdesc struct {
	Hostname      string `json:"hostname"`
	SysmgrVersion string `json:"sysmgr-version"`
	OsName        string `json:"os-name"`
	OsVersion     string `json:"os-version"`
	OsKernelVer   string `json:"os-uname"`
	OsArch        string `json:"os-architecture"`
	RamSize       uint32 `json:"ram-size"`
}

const notAvailable = "(not available)"

func sysdescLoad() sysdesc {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = notAvailable
	}

	osr, err := osreleaseLoad()
	if err != nil {
		osr.Name = notAvailable
		osr.Version = notAvailable
	}

	kernelVer := notAvailable
	arch := notAvailable

	return sysdesc{
		Hostname:      hostname,
		SysmgrVersion: sysdescSysmgrversion,
		OsName:        osr.Name,
		OsVersion:     osr.Version,
		OsKernelVer:   kernelVer,
		OsArch:        arch,
		RamSize:       meminfoLoad().MemTotal,
	}
}
