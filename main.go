package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/vulsio/go-cpe-dictionary/commands"
	"github.com/vulsio/go-cpe-dictionary/config"
)

func main() {
	var v = flag.Bool("v", false, "Show version")

	if envArgs := os.Getenv("GO_CPE_DICTIONARY_ARGS"); 0 < len(envArgs) {
		commands.RootCmd.SetArgs(strings.Fields(envArgs))
	} else {
		flag.Parse()
	}

	if *v {
		fmt.Printf("go-cpe-dictionary %s %s\n", config.Version, config.Revision)
		os.Exit(0)
	}

	if err := commands.RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	os.Exit(0)
}
