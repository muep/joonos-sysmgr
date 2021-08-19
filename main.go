package main

import (
	"fmt"
	"os"
)

func main() {
	subcommands := []*subcommand{
		certShowSubcommand(),
		offlineSubcommand(),
		runSubcommand(),
	}

	err := runWithArgsAndSubcommands(os.Args, subcommands)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to run %s: %v\n", os.Args, err)
	}
}
