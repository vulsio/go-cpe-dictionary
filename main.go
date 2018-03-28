package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/google/subcommands"
	"github.com/kotakanbe/go-cpe-dictionary/commands"
)

// Name ... Name
const Name string = "go-cpe-dictionary"

// Version ... Version
var version = "0.2.0"

// Revision of Git
var revision string

func main() {
	subcommands.Register(subcommands.HelpCommand(), "")
	subcommands.Register(subcommands.FlagsCommand(), "")
	subcommands.Register(subcommands.CommandsCommand(), "")
	subcommands.Register(&commands.FetchNvdCmd{}, "fetchnvd")

	var v = flag.Bool("v", false, "Show version")

	if envArgs := os.Getenv("GO_CPE_DICTIONARY_ARGS"); 0 < len(envArgs) {
		flag.CommandLine.Parse(strings.Fields(envArgs))
	} else {
		flag.Parse()
	}

	if *v {
		fmt.Printf("go-cve-dictionary %s %s\n", version, revision)
		os.Exit(int(subcommands.ExitSuccess))
	}

	ctx := context.Background()
	os.Exit(int(subcommands.Execute(ctx)))
}
