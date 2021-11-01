package main

import (
	"io"
	"net/http"
	"os"
	"os/exec"
)

type upgCommand struct {
	Url       string   `json:"url"`
	Sha256sum string   `json:"sha256sum"`
	Nodes     []string `json:"nodes"`
}

func upgrade(upgTool []string, cmd upgCommand) {
	res, err := http.Get(cmd.Url)
	if err != nil {
		return
	}

	defer res.Body.Close()

	if res.StatusCode != 200 {
		return
	}

	updateProcess := exec.Command(upgTool[0], upgTool[1:]...)

	// Preferably the output would be handled in more controlled
	// fashion, but in happy cases it is not expected that anyone
	// will be looking at the output of either this program or the
	// upgrade tool.
	updateProcess.Stdout = os.Stdout
	updateProcess.Stderr = os.Stderr
	updatePipe, err := updateProcess.StdinPipe()
	if err != nil {
		return
	}

	err = updateProcess.Start()
	if err != nil {
		return
	}
	defer updateProcess.Wait()

	defer updatePipe.Close()

	numCopied, err := io.Copy(updatePipe, res.Body)
	if err != nil {
		return
	}

	if numCopied != res.ContentLength {
		return
	}
	updatePipe.Close()
	updateProcess.Wait()
}
