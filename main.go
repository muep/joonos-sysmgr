package main

import (
	"flag"
	"fmt"
	"os"
)

func scRun() *subcommand {
	flagset := flag.NewFlagSet("run", flag.ExitOnError)
	args := commonArgs{}
	commonFlags(flagset, &args)
	run := func() error {
		fmt.Println("Hello run!")
		fmt.Printf("Should use config from %s\n", args.config)
		return nil
	}

	runCommand := subcommand{
		flagset: flagset,
		run:     run,
	}
	return &runCommand

}

func subcommands() []*subcommand {
	return []*subcommand{certShowSubcommand(), scRun()}
}

func main() {
	err := runWithArgsAndSubcommands(os.Args, subcommands())

	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to run %s: %v\n", os.Args, err)
	}

}
