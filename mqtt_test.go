package main

import (
	"testing"
)

func TestMqttConnect(t *testing.T) {
	const configPath = "test-files/joonos.json"
	config, err := configLoad(configPath)
	if err != nil {
		t.Fatalf("Failed to read %s: %v", configPath, err)
	}

	state, err := stateLoad(config)
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	mqttconf := mqttparams{
		nodename: state.nodename,
		server:   state.config.Mqttsrv,
		tlsconf:  state.tlsconfig(),
	}

	conres, err := mqttConnect(0, mqttconf, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	if !conres.client.IsConnected() {
		t.Error("Expected to have a connected client")
	}

	conres.client.Disconnect(0)
}
