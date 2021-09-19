package main

import (
	"testing"
)

func mqttParamsUsingConfig(config config) (mqttparams, error) {
	state, err := stateLoad(config)
	if err != nil {
		return mqttparams{}, err
	}

	params := mqttparams{
		nodename: state.nodename,
		server:   config.Mqttsrv,
		tlsconf:  state.tlsconfig(),
	}
	return params, nil
}

func mqttParamsUsingConfigFrom(configPath string) (mqttparams, error) {
	config, err := configLoad(configPath)
	if err != nil {
		return mqttparams{}, err
	}

	return mqttParamsUsingConfig(config)
}

func TestMqttConnect(t *testing.T) {
	const configPath = "test-files/joonos.json"
	mqttconf, err := mqttParamsUsingConfigFrom(configPath)
	if err != nil {
		t.Fatalf("Failed to get mqtt params %v", err)
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

func TestMqttConnectBadSrvname(t *testing.T) {
	const configPath = "test-files/joonos.json"
	config, err := configLoad(configPath)
	if err != nil {
		t.Fatalf("Failed to read %s: %v", configPath, err)
	}

	// This is expected to produce an error in the verification
	// of the certificate
	config.Mqttsrv = "tls://127.0.0.2:8884"

	mqttconf, err := mqttParamsUsingConfig(config)
	if err != nil {
		t.Fatalf("Failed to get mqtt params %v", err)
	}

	conres, err := mqttConnect(0, mqttconf, nil)
	if err == nil {
		t.Fatal("Expected to fail connecting")
	}

	// Here would be great if we could somehow check that the failure
	// was the one expected one, about mismatch between server address
	// and the IP field in the certificate

	if conres.client.IsConnected() {
		t.Error("Expected to not have a connected client")
	}
}
