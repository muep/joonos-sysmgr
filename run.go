package main

import (
	"flag"
	"fmt"
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
		fmt.Println("Loading failed because:", state.nodecerterr)
	}

	fmt.Printf("Connection should be done with this certificate:\n")
	certShowRaw(state.tlscert().Certificate)

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
