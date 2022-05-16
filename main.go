package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/vulsio/go-cpe-dictionary/commands"
)

func main() {
	if envArgs := os.Getenv("GO_CPE_DICTIONARY_ARGS"); 0 < len(envArgs) {
		commands.RootCmd.SetArgs(strings.Fields(envArgs))
	}

	if err := commands.RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}
