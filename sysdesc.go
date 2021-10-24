package main

import (
	"os"

	"golang.org/x/sys/unix"
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

func sysdescStringFromBytesSlice(slice []byte) string {
	strLen := len(slice)
	for n, b := range slice {
		if b == 0 {
			strLen = n
			break
		}
	}

	return string(slice[:strLen])
}

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

	utsname := unix.Utsname{}

	kernelVer := notAvailable
	arch := notAvailable

	err = unix.Uname(&utsname)
	if err == nil {
		kernelVer = sysdescStringFromBytesSlice(utsname.Release[:])
		arch = sysdescStringFromBytesSlice(utsname.Machine[:])
	}

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
