package main

import (
	"flag"
	"fmt"
	"time"
)

func runWithConfig(configPath string) error {
	fmt.Printf("Loading configuration from %s...\n", configPath)

	config, err := configLoad(configPath)
	if err != nil {
		return fmt.Errorf("Failed to load configuration: %w", err)
	}
	fmt.Println("Configuration loaded. Initializing state...")

	state, err := stateLoad(config)
	if err != nil {
		return fmt.Errorf("Failed to initialize state: %w", err)
	}

	fmt.Println("State initialized")

	if state.nodecert == nil {
		fmt.Println("Node certificate is not present.")
	}
	if state.nodecerterr != nil {
		fmt.Println("Node certificate problem:", state.nodecerterr)
	}

	fmt.Printf("Connection should be done with this certificate:\n")
	certShowRaw(state.tlscert().Certificate)

	mqttchans := mqttStartNode()
	mqttchans.params <- state.mqttparams()

	var renewcert <-chan time.Time

	for {
		select {
		case <-mqttchans.didconnect:
			renewDuration := state.certRenewTime()
			if renewDuration > time.Second {
				// Supposedly we intend to keep the current certificate for
				// some time, so let's be nice and clear out any dangling
				// previous CSR.
				mqttchans.csrs <- nil
			}

			// Could be rather immediately, or also quite some time in
			// the future.
			renewcert = time.After(renewDuration)

		case msg := <-mqttchans.messages:
			fmt.Printf("MQTT: %s\n", msg)
		case <-renewcert:
			fmt.Println("Should renew the certificate")
			csr, err := state.csr()
			if err != nil {
				fmt.Printf("Failed to generate CSR: %v\n", err)
			} else {
				fmt.Println("Pushing CSR to channel")
				mqttchans.csrs <- csr
				fmt.Println("Asked MQTT layer to to send a CSR")
			}
			// Will retry after some time, in case there is no reply
			renewcert = time.After(time.Hour)

		case certs := <-mqttchans.certs:
			err = state.setCertificates(certs)
			if err != nil {
				fmt.Printf("Did not accept certificate: %v\n", err)
			} else {
				fmt.Printf("Updated certificate\n")
				mqttchans.csrs <- nil
				mqttchans.params <- state.mqttparams()
				renewcert = time.After(state.certRenewTime())
			}
		}
	}
}

func runSubcommand() *subcommand {
	flagset := flag.NewFlagSet("run", flag.ExitOnError)
	args := commonArgs{}
	commonFlags(flagset, &args)
	run := func() error {
		return runWithConfig(args.config)
	}

	runCommand := subcommand{
		flagset: flagset,
		run:     run,
	}
	return &runCommand
}
