package main

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"
)

func waitMosquittoOk(config config) error {
	state, err := stateLoad(config)
	if err != nil {
		return fmt.Errorf("Failed to set up tmp state for MQTT check: %w", err)
	}

	mqttparams := mqttparams{
		nodename: "unprovisioned",
		server:   config.Mqttsrv,
		tlsconf:  state.tlsconfig(),
	}
	mqttparams.tlsconf.InsecureSkipVerify = true

	for n := 0; n < 20; n++ {
		conres, err := mqttConnect(0, mqttparams, nil)
		if err == nil {
			conres.client.Disconnect(0)
			fmt.Println("Got OK connection")
			return nil
		}

		fmt.Printf("Connection failed with %v. Waiting for a bit.\n", err)
		time.Sleep(10 * time.Millisecond)
	}

	return err
}

func runMosquitto(config config) (chan struct{}, error) {
	fmt.Println("Setting up mqtt")

	doneChan := make(chan struct{})
	cmd := exec.Command("mosquitto", "-v", "-c", "test-files/mosquitto.conf")
	err := cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("Failed to execute mosquitto: %w", err)
	}

	err = waitMosquittoOk(config)
	if err != nil {
		return nil, fmt.Errorf("Failed to wait for MQTT service: %w", err)
	}

	go func() {
		<-doneChan
		fmt.Println("Cleaning up mqtt")

		cmd.Process.Signal(os.Interrupt)
		time.Sleep(100 * time.Millisecond)
		cmd.Process.Signal(os.Kill)
		_, cmdErr := cmd.Process.Wait()
		if cmdErr != nil {
			fmt.Printf("Failed to terminate process: %v", cmdErr)
		}
		doneChan <- struct{}{}
	}()

	return doneChan, nil
}

func TestMain(m *testing.M) {
	config, err := configLoad("test-files/joonos.json")
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		return
	}

	doneChan, err := runMosquitto(config)
	if err != nil {
		fmt.Printf("Failed to prepare mosquitto: %v\n", err)
		return
	}

	res := m.Run()

	doneChan <- struct{}{}
	<-doneChan

	os.Exit(res)
}
