package main

import (
	"os"
)

func fsCheckDirPresent(datadir string) error {
	ddirStat, err := os.Stat(datadir)

	if err == nil && ddirStat.IsDir() {
		return nil
	}

	if !os.IsNotExist(err) {
		// Some error that can not be addressed by attempting the creation
		return err
	}

	return os.Mkdir(datadir, 0700)
}
