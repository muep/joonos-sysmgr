package main

import (
	"time"

	"golang.org/x/sys/unix"
)

type sysstat struct {
	Time          int64      `json:"time"`
	Uptime        int64      `json:"uptime"`
	LoadAvg       [3]float32 `json:"load-avg"`
	LastPid       uint32     `json:"last-pid"`
	TasksRunnable uint32     `json:"tasks-runnable"`
	TasksTotal    uint32     `json:"tasks-total"`
	RamTotal      uint32     `json:"ram-total"`
	RamFree       uint32     `json:"ram-free"`
	RamCached     uint32     `json:"ram-cached"`
	UsedCpu       uint64     `json:"used-cpu"`
	UsedMaxRss    uint64     `json:"used-max-rss"`
}

func sysstatGet() (sysstat, error) {
	res := sysstat{
		Time: time.Now().Unix(),
	}

	var sysinfo unix.Sysinfo_t
	err := unix.Sysinfo(&sysinfo)
	if err == nil {
		res.Uptime = sysinfo.Uptime
	}

	var usage unix.Rusage
	err = unix.Getrusage(unix.RUSAGE_SELF, &usage)
	if err == nil {
		res.UsedCpu = sysstatUsFromTimeval(usage.Utime) + sysstatUsFromTimeval(usage.Stime)
		res.UsedMaxRss = uint64(usage.Maxrss)
	}

	loadavg, err := loadavgLoad()
	if err == nil {
		res.LoadAvg = loadavg.Loads
		res.LastPid = loadavg.LastPid
		res.TasksRunnable = loadavg.TasksRunnable
		res.TasksTotal = loadavg.TasksTotal
	}

	meminfo, err := meminfoLoad()
	if err == nil {
		res.RamTotal = meminfo.MemTotal
		res.RamFree = meminfo.MemFree
		res.RamCached = meminfo.Cached
	}

	return res, nil
}

func sysstatUsFromTimeval(tv unix.Timeval) uint64 {
	return uint64(tv.Sec)*1000000 + uint64(tv.Usec)
}
