package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"os"
)

func runCheckDatadir(datadir string) error {
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

func runWithConfig(configPath string) error {
	fmt.Printf("Loading configuration from %s...\n", configPath)

	config, err := configLoad(configPath)
	if err != nil {
		return fmt.Errorf("Failed to load configuration: %w", err)
	}
	fmt.Println("Configuration loaded")

	err = runCheckDatadir(config.Datadir)
	if err != nil {
		return fmt.Errorf("Data directory is not available: %w", err)
	}

	cacert, err := certLoadOneFromPath(config.Cacert)
	if err != nil {
		return err
	}

	fmt.Printf("From %s loaded root CA certificate:\n", config.Cacert)
	certShow(cacert)

	provcert, err := tls.LoadX509KeyPair(config.Provcert, config.Provkey)
	if err != nil {
		return err
	}

	usecert := &provcert

	fmt.Printf(
		"From\n %s and\n %s\nloaded provisioning certificate:\n",
		config.Provcert,
		config.Provkey,
	)

	certShowRaw(provcert.Certificate)

	nodecert, err := tls.LoadX509KeyPair(config.Nodecert(), config.Nodekey())
	if err == nil {
		usecert = &nodecert

		fmt.Printf(
			"From\n %s and\n %s\nloaded node certificate:\n",
			config.Nodecert(),
			config.Nodekey(),
		)
		certShowRaw(nodecert.Certificate)
	} else {
		if !os.IsNotExist(err) {
			return err
		}

		fmt.Printf("Could not load the node certificate\n")
		fmt.Printf("Should attempt to provision a new one\n")
		// In case the certificate was simply absent, we can should attempt
		// continuing with the provisioning cert
	}

	fmt.Printf("Connection should be done with this certificate:\n")
	certShowRaw(usecert.Certificate)

	return nil
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
