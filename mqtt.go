package main

import (
	"crypto/tls"
	"flag"
	"fmt"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type mqttconres struct {
	client mqtt.Client
	conid  uint
}

type mqttconlost struct {
	client mqtt.Client
	err    error
}

type mqttparams struct {
	nodename string
	server   string
	tlsconf  *tls.Config
}

func mqttConnect(
	conid uint,
	params mqttparams,
	chanConLost chan<- mqttconlost,
) (mqttconres, error) {
	server := params.server
	if len(server) == 0 {
		return mqttconres{}, fmt.Errorf("MQTT server address is empty")
	}

	opts := mqtt.NewClientOptions()

	opts.AddBroker(params.server)
	opts.SetAutoReconnect(false)

	if chanConLost != nil {
		opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
			chanConLost <- mqttconlost{
				client: client,
				err:    err,
			}
		})
	}

	mqttName := params.nodename

	if params.tlsconf != nil {
		mqttName = params.tlsconf.Certificates[0].Leaf.Subject.CommonName
		opts.SetTLSConfig(params.tlsconf)
	}

	opts.SetUsername(mqttName)

	client := mqtt.NewClient(opts)

	connectToken := client.Connect()
	connectToken.Wait()

	res := mqttconres{
		client: client,
		conid:  conid,
	}

	return res, connectToken.Error()
}

func mqttConnectSubcmd() *subcommand {
	flagset := flag.NewFlagSet("mqtt-connect", flag.ExitOnError)
	args := commonArgs{}
	commonFlags(flagset, &args)

	run := func() error {
		return mqttConnectSubcmdWithConfig(args.config)
	}

	certShowCommand := subcommand{
		flagset: flagset,
		run:     run,
	}
	return &certShowCommand
}

func mqttConnectSubcmdWithConfig(configPath string) error {
	config, err := configLoad(configPath)
	if err != nil {
		return fmt.Errorf("Failed to load config: %w", err)
	}

	state, err := stateLoad(config)
	if err != nil {
		return fmt.Errorf("Failed to load state: %w", err)
	}

	connectionLostChannel := make(chan mqttconlost)

	mqttparams := mqttparams{
		nodename: state.nodename,
		server:   config.Mqttsrv,
		tlsconf:  state.tlsconfig(),
	}

	conRes, err := mqttConnect(
		0,
		mqttparams,
		connectionLostChannel,
	)
	if err != nil {
		return err
	}

	fmt.Printf("Connected %v\n", conRes.client)
	fmt.Println("Disconnecting...")

	conRes.client.Disconnect(1)
	fmt.Println("Disconnected.")

	return nil
}
