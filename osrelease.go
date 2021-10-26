package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type osrelease struct {
	Name    string
	Version string
}

const osreleaseFilename string = "/etc/os-release"
const osreleasePrefixName string = "ID="
const osreleasePrefixVersion string = "VERSION_ID="

func osreleaseLoad() (osrelease, error) {
	var res osrelease
	file, err := os.Open(osreleaseFilename)
	if err != nil {
		return res, fmt.Errorf("failed to open %s: %w", osreleaseFilename, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, osreleasePrefixName) {
			res.Name = strings.TrimSpace(strings.TrimPrefix(line, osreleasePrefixName))
		}

		if strings.HasPrefix(line, osreleasePrefixVersion) {
			res.Version = strings.TrimSpace(strings.TrimPrefix(line, osreleasePrefixVersion))
		}
	}

	return res, nil
}
