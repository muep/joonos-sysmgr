package main

import (
	"flag"
	"fmt"
	"os"
)

type commonArgs struct {
	config string
}

type subcommand struct {
	flagset *flag.FlagSet
	run     func()
}

func subcmdByName(subcmds []*subcommand, name string) *subcommand {
	for _, subcmd := range subcmds {
		if subcmd.flagset.Name() == name {
			return subcmd
		}
	}

	return nil
}

func commonFlags(flagset *flag.FlagSet, args *commonArgs) {
	flagset.StringVar(
		&args.config,
		"config",
		"/etc/joonos/joonos.conf",
		"path to config file")
}

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

func mainUsage(dest *os.File, selfName string, subcmds []*subcommand) {
	fmt.Fprintf(dest, "Usage: %s <subcommand> [options]\n", selfName)
	fmt.Fprintln(dest, "subcommands:")
	for _, subcmd := range subcmds {
		fmt.Fprintf(dest, "    %s\n", subcmd.flagset.Name())
	}
}

func runWithArgs(args []string) {
	argCnt := len(args)

	if argCnt == 0 {
		fmt.Fprintln(os.Stderr, "ERR: expected to have an argument list")
		return
	}

	subcmds := subcommands()

	if argCnt <= 1 {
		mainUsage(os.Stderr, args[0], subcmds)
		return
	}

	subcmdName := args[1]

	subcmd := subcmdByName(subcmds, subcmdName)
	if subcmd == nil {
		fmt.Fprintf(os.Stderr, "Unrecognized subcommand \"%s\"\n", subcmdName)
		mainUsage(os.Stderr, args[0], subcmds)
		return
	}

	err := subcmd.flagset.Parse(args[2:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "Got err")
	}

	subcmd.run()
}

func main() {
	args := os.Args
	runWithArgs(args)
}
