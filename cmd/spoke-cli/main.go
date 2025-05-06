package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/platinummonkey/spoke/pkg/cli"
)

func main() {
	// Create root command
	rootCmd := cli.NewRootCommand()

	// Parse flags
	flag.Parse()

	// Execute command
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
} 