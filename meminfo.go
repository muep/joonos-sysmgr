package main

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

type meminfo struct {
	MemTotal     uint32
	MemFree      uint32
	Cached       uint32
	MemAvailable uint32
}

const (
	meminfoTotal     string = "MemTotal:"
	meminfoFree      string = "MemFree:"
	meminfoCached    string = "Cached:"
	meminfoAvailable string = "MemAvailable:"
	meminfoKb        string = " kB"
)

func strWithoutPrefixAndSuffix(snl string, prefix string, suffix string) string {
	if !strings.HasPrefix(snl, prefix) {
		return ""
	}

	s := strings.TrimSpace(snl)

	if !strings.HasSuffix(s, suffix) {
		return ""
	}

	return strings.TrimSpace(
		strings.TrimSuffix(
			strings.TrimPrefix(s, prefix),
			suffix,
		),
	)
}

func meminfoLoad() meminfo {
	res := meminfo{}

	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return res
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		memtotalText := strWithoutPrefixAndSuffix(line, meminfoTotal, meminfoKb)
		if len(memtotalText) > 0 {
			memtotal, err := strconv.ParseInt(memtotalText, 10, 32)
			if err != nil {
				continue
			}
			res.MemTotal = uint32(memtotal)
			continue
		}

		memfreeText := strWithoutPrefixAndSuffix(line, meminfoFree, meminfoKb)
		if len(memfreeText) > 0 {
			memfree, err := strconv.ParseInt(memfreeText, 10, 32)
			if err != nil {
				continue
			}
			res.MemFree = uint32(memfree)
			continue
		}

		memcachedText := strWithoutPrefixAndSuffix(line, meminfoCached, meminfoKb)
		if len(memcachedText) > 0 {
			memcached, err := strconv.ParseInt(memcachedText, 10, 32)
			if err != nil {
				continue
			}
			res.Cached = uint32(memcached)
			continue
		}

		memavailText := strWithoutPrefixAndSuffix(line, meminfoAvailable, meminfoKb)
		if len(memavailText) > 0 {
			memavail, err := strconv.ParseInt(memavailText, 10, 32)
			if err != nil {
				continue
			}
			res.MemAvailable = uint32(memavail)
			continue
		}
	}

	return res
}
