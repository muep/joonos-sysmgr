package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
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
	provisioning bool
	nodename     string
	server       string
	tlsconf      *tls.Config
}

type mqttdidconnect struct {
	provisioning bool
}

type mqttservice struct {
	didconnect <-chan mqttdidconnect
	params     chan<- mqttparams
	messages   <-chan string
	csrs       chan<- *x509.CertificateRequest
	certs      <-chan []*x509.Certificate
	stop       chan<- struct{}
	sysdesc    chan<- sysdesc
	sysstat    chan<- sysstat
	upgcmds    <-chan upgCommand
}

func mqttRunOnce(
	params mqttparams,
	didconnect chan<- mqttdidconnect,
	messages chan<- string,
	stop <-chan struct{},
	sysdesc <-chan sysdesc,
	sysstat <-chan sysstat,
	upgcmds chan<- upgCommand,
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
	topicSysdesc := fmt.Sprintf("joonos/%s/status/description", mqttName)
	topicSysstat := fmt.Sprintf("joonos/%s/status/stat", mqttName)
	topicSwupdate := fmt.Sprintf("joonos/%s/upgrade", mqttName)

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

		c.Subscribe(topicSwupdate, 1, func(c mqtt.Client, m mqtt.Message) {
			var cmd upgCommand

			err := json.Unmarshal(m.Payload(), &cmd)
			if err != nil {
				messages <- fmt.Sprintf("failed to read upgrade cmd: %v", err)
				return
			}

			if len(cmd.Url) == 0 {
				messages <- "expected a non-empty URL"
				return
			}

			if len(cmd.Nodes) > 0 {
				if !sliceContains(cmd.Nodes, mqttName) {
					return
				}
			}

			upgcmds <- cmd
		}).Wait()

		didconnect <- mqttdidconnect{
			provisioning: params.provisioning,
		}
		messages <- "Connected and subscribed."
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

	keepgoing := true
	messages <- "Entering the MQTT main loop"
	for keepgoing {
		select {
		case desc := <-sysdesc:
			payload, err := json.Marshal(&desc)
			if err == nil {
				client.Publish(topicSysdesc, 1, true, payload)
			}
		case stat := <-sysstat:
			if !params.provisioning {
				payload, err := json.Marshal(&stat)
				if err == nil {
					client.Publish(topicSysstat, 1, true, payload)
				}
			}
		case csr := <-csrsIn:
			payload := []byte{}

			var msg string
			if csr != nil {
				payload = csr.Raw
				msg = fmt.Sprintf("Published CSR at %s", topicCsr)
			} else {
				msg = fmt.Sprintf("Cleared csr at %s", topicCsr)
			}
			client.Publish(topicCsr, 1, true, payload).Wait()
			messages <- msg
		case <-stop:
			messages <- "Closing down"
			keepgoing = false
		}
	}

	messages <- "Asking MQTT client to disconnect"
	client.Disconnect(0)
}

func mqttStartNode() mqttservice {
	didconnect := make(chan mqttdidconnect)
	params := make(chan mqttparams)
	messages := make(chan string, 50)
	csrs := make(chan *x509.CertificateRequest)
	certs := make(chan []*x509.Certificate)
	sysdescs := make(chan sysdesc)
	sysstats := make(chan sysstat)
	swupdates := make(chan upgCommand)
	stop := make(chan struct{})

	go func() {
		parameters := <-params
		for {
			stopCurrent := make(chan struct{})
			go mqttRunOnce(
				parameters,
				didconnect,
				messages,
				stopCurrent,
				sysdescs,
				sysstats,
				swupdates,
				csrs,
				certs,
			)

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
		sysdesc:    sysdescs,
		sysstat:    sysstats,
		upgcmds:    swupdates,
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
		return fmt.Errorf("failed to load config: %w", err)
	}

	state, err := stateLoad(config)
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
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
