package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"net/url"

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

type mqttservice struct {
	didconnect <-chan struct{}
	params     chan<- mqttparams
	messages   <-chan string
	csrs       chan<- *x509.CertificateRequest
	certs      <-chan []*x509.Certificate
	stop       chan<- struct{}
}

func mqttRunOnce(
	params mqttparams,
	didconnect chan<- struct{},
	messages chan<- string,
	stop <-chan struct{},
	csrsIn <-chan *x509.CertificateRequest,
	certsOut chan<- []*x509.Certificate) {

	if len(params.server) == 0 {
		return
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(params.server)

	mqttName := params.nodename

	if params.tlsconf != nil {
		mqttName = params.tlsconf.Certificates[0].Leaf.Subject.CommonName
		opts.SetTLSConfig(params.tlsconf)
	}

	topicCert := fmt.Sprintf("joonos/%s/cert", mqttName)
	topicCsr := fmt.Sprintf("joonos/%s/csr", mqttName)

	opts.SetAutoReconnect(true)
	opts.SetUsername(mqttName)
	opts.SetOnConnectHandler(func(c mqtt.Client) {
		c.Subscribe(topicCert, 1, func(c mqtt.Client, m mqtt.Message) {
			certs, err := x509.ParseCertificates(m.Payload())
			if err != nil {
				messages <- fmt.Sprintf("Failed to read incoming certificate: %v", err)
				return
			}

			certsOut <- certs
		}).Wait()
		didconnect <- struct{}{}
		messages <- fmt.Sprintf("Connected and subscribed.")
	})

	opts.SetConnectionAttemptHandler(func(broker *url.URL, tlsCfg *tls.Config) *tls.Config {
		messages <- fmt.Sprintf("Attempting connection to %v", broker)
		return tlsCfg
	})

	opts.SetReconnectingHandler(func(c mqtt.Client, co *mqtt.ClientOptions) {
		messages <- "Reconnecting"
	})

	client := mqtt.NewClient(opts)

	messages <- "Initiating connection"
	connectToken := client.Connect()
	connectToken.Wait()

	messages <- fmt.Sprintf("Connected to %s as %s", params.server, mqttName)

	keepgoing := true
	for keepgoing {
		messages <- fmt.Sprintf("Waiting for stuff")
		select {
		case csr := <-csrsIn:
			payload := []byte{}
			if csr != nil {
				payload = csr.Raw
				messages <- fmt.Sprintf("Publishing CSR at %s", topicCsr)
			} else {
				messages <- fmt.Sprintf("Clearing csr at %s", topicCsr)
			}
			client.Publish(topicCsr, 1, true, payload).Wait()
			messages <- fmt.Sprintf("Published to %s", topicCsr)
		case <-stop:
			keepgoing = false
		}
	}
}

func mqttStartNode() mqttservice {
	didconnect := make(chan struct{})
	params := make(chan mqttparams)
	messages := make(chan string, 50)
	csrs := make(chan *x509.CertificateRequest)
	certs := make(chan []*x509.Certificate)
	stop := make(chan struct{})

	go func() {
		parameters := <-params
		for {
			stopCurrent := make(chan struct{})
			go mqttRunOnce(parameters, didconnect, messages, stopCurrent, csrs, certs)
			parameters = <-params
			stopCurrent <- struct{}{}
		}
	}()

	return mqttservice{
		didconnect: didconnect,
		messages:   messages,
		params:     params,
		csrs:       csrs,
		certs:      certs,
		stop:       stop,
	}
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

	mqttConnectCommand := subcommand{
		flagset: flagset,
		run:     run,
	}
	return &mqttConnectCommand
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
