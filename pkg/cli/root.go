package cli

import (
	"flag"
	"fmt"
	"os"
)

// Command represents a CLI command
type Command struct {
	Name        string
	Description string
	Run         func(args []string) error
	Subcommands map[string]*Command
	Flags       *flag.FlagSet
}

// NewRootCommand creates the root command
func NewRootCommand() *Command {
	root := &Command{
		Name:        "spoke",
		Description: "Spoke - A Protobuf Schema Registry CLI",
		Subcommands: make(map[string]*Command),
		Flags:       flag.NewFlagSet("spoke", flag.ExitOnError),
	}

	// Add subcommands
	root.Subcommands["push"] = newPushCommand()
	root.Subcommands["pull"] = newPullCommand()
	root.Subcommands["compile"] = newCompileCommand()
	root.Subcommands["validate"] = newValidateCommand()

	return root
}

// Execute runs the command
func (c *Command) Execute() error {
	args := os.Args[1:]
	if len(args) == 0 {
		return c.usage()
	}

	// Check for help flag
	if args[0] == "-h" || args[0] == "--help" {
		return c.usage()
	}

	// Check for subcommand
	if subcmd, ok := c.Subcommands[args[0]]; ok {
		return subcmd.Run(args[1:])
	}

	return fmt.Errorf("unknown command: %s", args[0])
}

// usage prints the command usage
func (c *Command) usage() error {
	fmt.Printf("Usage: %s <command> [args]\n\n", c.Name)
	fmt.Printf("Commands:\n")
	for name, cmd := range c.Subcommands {
		fmt.Printf("  %-15s %s\n", name, cmd.Description)
	}
	return nil
} 