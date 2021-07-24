package main

import (
	"flag"
	"fmt"
	"os"
)

func scCertShow() *subcommand {
	flagset := flag.NewFlagSet("cert-show", flag.ExitOnError)
	args := commonArgs{}
	commonFlags(flagset, &args)

	certIn := flagset.String("in", "", "path to PEM file")

	run := func() {
		if len(*certIn) == 0 {
			fmt.Fprintln(os.Stderr, "ERR: -in is required")
			return
		}
		fmt.Printf("Loading cert from %s\n", *certIn)
	}

	certShowCommand := subcommand{
		flagset: flagset,
		run:     run,
	}
	return &certShowCommand
}

func scRun() *subcommand {
	flagset := flag.NewFlagSet("run", flag.ExitOnError)
	args := commonArgs{}
	commonFlags(flagset, &args)
	run := func() {
		fmt.Println("Hello run!")
		fmt.Printf("Should use config from %s\n", args.config)

	}

	runCommand := subcommand{
		flagset: flagset,
		run:     run,
	}
	return &runCommand

}

func subcommands() []*subcommand {
	return []*subcommand{scCertShow(), scRun()}
}

func main() {
	runWithArgsAndSubcommands(os.Args, subcommands())
}
