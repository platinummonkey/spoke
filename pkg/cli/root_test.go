package cli

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRootCommand(t *testing.T) {
	root := NewRootCommand()

	// Test basic properties
	assert.Equal(t, "spoke", root.Name)
	assert.Equal(t, "Spoke - A Protobuf Schema Registry CLI", root.Description)
	assert.NotNil(t, root.Subcommands)
	assert.NotNil(t, root.Flags)

	// Test that all expected subcommands are registered
	expectedCommands := []string{
		"push",
		"batch-push",
		"pull",
		"compile",
		"validate",
		"check-compatibility",
		"lint",
		"languages",
	}

	for _, cmdName := range expectedCommands {
		assert.Contains(t, root.Subcommands, cmdName, "Expected subcommand %s to be registered", cmdName)
		assert.NotNil(t, root.Subcommands[cmdName], "Expected subcommand %s to be non-nil", cmdName)
	}

	// Verify the exact number of subcommands
	assert.Equal(t, len(expectedCommands), len(root.Subcommands))
}

func TestCommandUsage(t *testing.T) {
	root := NewRootCommand()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := root.usage()

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify no error
	assert.NoError(t, err)

	// Verify output contains expected content
	assert.Contains(t, output, "Usage: spoke <command> [args]")
	assert.Contains(t, output, "Commands:")
	assert.Contains(t, output, "push")
	assert.Contains(t, output, "batch-push")
	assert.Contains(t, output, "pull")
	assert.Contains(t, output, "compile")
	assert.Contains(t, output, "validate")
	assert.Contains(t, output, "check-compatibility")
	assert.Contains(t, output, "lint")
	assert.Contains(t, output, "languages")
}

func TestCommandExecute_NoArgs(t *testing.T) {
	root := NewRootCommand()

	// Save and override os.Args
	oldArgs := os.Args
	os.Args = []string{"spoke"}
	defer func() { os.Args = oldArgs }()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := root.Execute()

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Should show usage when no args provided
	assert.NoError(t, err)
	assert.Contains(t, output, "Usage: spoke <command> [args]")
}

func TestCommandExecute_HelpFlag(t *testing.T) {
	root := NewRootCommand()

	testCases := []struct {
		name     string
		helpFlag string
	}{
		{"lowercase -h", "-h"},
		{"uppercase -H", "-H"},
		{"lowercase --help", "--help"},
		{"uppercase --HELP", "--HELP"},
		{"mixed case --Help", "--Help"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Save and override os.Args
			oldArgs := os.Args
			os.Args = []string{"spoke", tc.helpFlag}
			defer func() { os.Args = oldArgs }()

			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := root.Execute()

			// Restore stdout
			w.Close()
			os.Stdout = oldStdout

			// Read captured output
			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			// Should show usage for help flag
			assert.NoError(t, err)
			assert.Contains(t, output, "Usage: spoke <command> [args]")
		})
	}
}

func TestCommandExecute_ValidSubcommand(t *testing.T) {
	root := NewRootCommand()

	// Create a mock subcommand for testing
	mockCalled := false
	mockRun := func(args []string) error {
		mockCalled = true
		return nil
	}

	root.Subcommands["test"] = &Command{
		Name:        "test",
		Description: "Test command",
		Run:         mockRun,
	}

	// Save and override os.Args
	oldArgs := os.Args
	os.Args = []string{"spoke", "test"}
	defer func() { os.Args = oldArgs }()

	err := root.Execute()

	assert.NoError(t, err)
	assert.True(t, mockCalled, "Expected mock subcommand to be called")
}

func TestCommandExecute_UnknownCommand(t *testing.T) {
	root := NewRootCommand()

	// Save and override os.Args
	oldArgs := os.Args
	os.Args = []string{"spoke", "nonexistent"}
	defer func() { os.Args = oldArgs }()

	err := root.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown command: nonexistent")
}

func TestCommandExecute_SubcommandWithArgs(t *testing.T) {
	root := NewRootCommand()

	// Create a mock subcommand that validates args
	var receivedArgs []string
	mockRun := func(args []string) error {
		receivedArgs = args
		return nil
	}

	root.Subcommands["test"] = &Command{
		Name:        "test",
		Description: "Test command",
		Run:         mockRun,
	}

	// Save and override os.Args
	oldArgs := os.Args
	os.Args = []string{"spoke", "test", "arg1", "arg2", "-flag"}
	defer func() { os.Args = oldArgs }()

	err := root.Execute()

	assert.NoError(t, err)
	require.Equal(t, []string{"arg1", "arg2", "-flag"}, receivedArgs)
}
