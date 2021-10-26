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
	run     func() error
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

func mainUsage(dest *os.File, selfName string, subcmds []*subcommand) {
	fmt.Fprintf(dest, "Usage: %s <subcommand> [options]\n", selfName)
	fmt.Fprintln(dest, "subcommands:")
	for _, subcmd := range subcmds {
		fmt.Fprintf(dest, "    %s\n", subcmd.flagset.Name())
	}
}

func runWithArgsAndSubcommands(args []string, subcmds []*subcommand) error {
	argCnt := len(args)

	if argCnt == 0 {
		return fmt.Errorf("expected to have an argument list")
	}

	if argCnt <= 1 {
		mainUsage(os.Stdout, args[0], subcmds)
		return nil
	}

	subcmdName := args[1]

	subcmd := subcmdByName(subcmds, subcmdName)
	if subcmd == nil {
		fmt.Fprintf(os.Stderr, "Unrecognized subcommand \"%s\"\n", subcmdName)
		mainUsage(os.Stderr, args[0], subcmds)
		return nil
	}

	err := subcmd.flagset.Parse(args[2:])
	if err != nil {
		return fmt.Errorf("failed to parse arguments: %w", err)
	}

	return subcmd.run()
}
