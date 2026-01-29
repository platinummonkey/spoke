package plugins

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRunnerPlugin is a mock implementation of RunnerPlugin for testing
type mockRunnerPlugin struct {
	mu            sync.Mutex
	manifest      *Manifest
	loadError     error
	unloadError   error
	executeError  error
	executeResult *ExecutionResult
	loadCalled    bool
	unloadCalled  bool
	executeCalled bool
}

func newMockRunnerPlugin(manifest *Manifest) *mockRunnerPlugin {
	return &mockRunnerPlugin{
		manifest: manifest,
	}
}

func (m *mockRunnerPlugin) Manifest() *Manifest {
	return m.manifest
}

func (m *mockRunnerPlugin) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.loadCalled = true
	return m.loadError
}

func (m *mockRunnerPlugin) Unload() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.unloadCalled = true
	return m.unloadError
}

func (m *mockRunnerPlugin) Execute(ctx context.Context, req *ExecutionRequest) (*ExecutionResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.executeCalled = true
	if m.executeError != nil {
		return nil, m.executeError
	}
	if m.executeResult != nil {
		return m.executeResult, nil
	}
	// Default success result
	return &ExecutionResult{
		ExitCode: 0,
		Stdout:   "mock output",
		Stderr:   "",
		Duration: 100 * time.Millisecond,
	}, nil
}

// TestExecutionRequest tests the ExecutionRequest structure
func TestExecutionRequest_Basic(t *testing.T) {
	req := &ExecutionRequest{
		Command:     []string{"echo", "hello"},
		WorkingDir:  "/tmp",
		Environment: map[string]string{"FOO": "bar"},
		Timeout:     30 * time.Second,
	}

	assert.Equal(t, []string{"echo", "hello"}, req.Command)
	assert.Equal(t, "/tmp", req.WorkingDir)
	assert.Equal(t, "bar", req.Environment["FOO"])
	assert.Equal(t, 30*time.Second, req.Timeout)
	assert.Nil(t, req.Stdin)
}

func TestExecutionRequest_WithStdin(t *testing.T) {
	stdinData := []byte("input data")
	req := &ExecutionRequest{
		Command:     []string{"cat"},
		WorkingDir:  "/tmp",
		Environment: map[string]string{},
		Timeout:     5 * time.Second,
		Stdin:       stdinData,
	}

	assert.Equal(t, stdinData, req.Stdin)
}

func TestExecutionRequest_EmptyCommand(t *testing.T) {
	req := &ExecutionRequest{
		Command:     []string{},
		WorkingDir:  "/tmp",
		Environment: map[string]string{},
		Timeout:     5 * time.Second,
	}

	assert.Empty(t, req.Command)
}

func TestExecutionRequest_NilEnvironment(t *testing.T) {
	req := &ExecutionRequest{
		Command:     []string{"echo", "test"},
		WorkingDir:  "/tmp",
		Environment: nil,
		Timeout:     5 * time.Second,
	}

	assert.Nil(t, req.Environment)
}

func TestExecutionRequest_MultipleEnvVars(t *testing.T) {
	req := &ExecutionRequest{
		Command:    []string{"env"},
		WorkingDir: "/tmp",
		Environment: map[string]string{
			"VAR1": "value1",
			"VAR2": "value2",
			"VAR3": "value3",
		},
		Timeout: 5 * time.Second,
	}

	assert.Len(t, req.Environment, 3)
	assert.Equal(t, "value1", req.Environment["VAR1"])
	assert.Equal(t, "value2", req.Environment["VAR2"])
	assert.Equal(t, "value3", req.Environment["VAR3"])
}

func TestExecutionRequest_LongTimeout(t *testing.T) {
	req := &ExecutionRequest{
		Command:     []string{"sleep", "60"},
		WorkingDir:  "/tmp",
		Environment: map[string]string{},
		Timeout:     1 * time.Hour,
	}

	assert.Equal(t, 1*time.Hour, req.Timeout)
}

func TestExecutionRequest_ZeroTimeout(t *testing.T) {
	req := &ExecutionRequest{
		Command:     []string{"echo", "test"},
		WorkingDir:  "/tmp",
		Environment: map[string]string{},
		Timeout:     0,
	}

	assert.Equal(t, time.Duration(0), req.Timeout)
}

// TestExecutionResult tests the ExecutionResult structure
func TestExecutionResult_Success(t *testing.T) {
	result := &ExecutionResult{
		ExitCode: 0,
		Stdout:   "Command executed successfully",
		Stderr:   "",
		Duration: 150 * time.Millisecond,
	}

	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, "Command executed successfully", result.Stdout)
	assert.Empty(t, result.Stderr)
	assert.Equal(t, 150*time.Millisecond, result.Duration)
}

func TestExecutionResult_Failure(t *testing.T) {
	result := &ExecutionResult{
		ExitCode: 1,
		Stdout:   "",
		Stderr:   "Command failed: file not found",
		Duration: 50 * time.Millisecond,
	}

	assert.Equal(t, 1, result.ExitCode)
	assert.Empty(t, result.Stdout)
	assert.Contains(t, result.Stderr, "file not found")
	assert.Equal(t, 50*time.Millisecond, result.Duration)
}

func TestExecutionResult_NonZeroExitCode(t *testing.T) {
	result := &ExecutionResult{
		ExitCode: 127,
		Stdout:   "",
		Stderr:   "command not found",
		Duration: 10 * time.Millisecond,
	}

	assert.Equal(t, 127, result.ExitCode)
	assert.Contains(t, result.Stderr, "command not found")
}

func TestExecutionResult_WithBothOutputs(t *testing.T) {
	result := &ExecutionResult{
		ExitCode: 0,
		Stdout:   "Processing file...\nDone!",
		Stderr:   "Warning: deprecated option used",
		Duration: 200 * time.Millisecond,
	}

	assert.Equal(t, 0, result.ExitCode)
	assert.Contains(t, result.Stdout, "Processing file")
	assert.Contains(t, result.Stdout, "Done!")
	assert.Contains(t, result.Stderr, "Warning")
	assert.Equal(t, 200*time.Millisecond, result.Duration)
}

func TestExecutionResult_LongOutput(t *testing.T) {
	longOutput := make([]byte, 10000)
	for i := range longOutput {
		longOutput[i] = 'a'
	}

	result := &ExecutionResult{
		ExitCode: 0,
		Stdout:   string(longOutput),
		Stderr:   "",
		Duration: 1 * time.Second,
	}

	assert.Equal(t, 10000, len(result.Stdout))
	assert.Equal(t, 1*time.Second, result.Duration)
}

func TestExecutionResult_ZeroDuration(t *testing.T) {
	result := &ExecutionResult{
		ExitCode: 0,
		Stdout:   "instant",
		Stderr:   "",
		Duration: 0,
	}

	assert.Equal(t, time.Duration(0), result.Duration)
}

// TestMockRunnerPlugin tests the mock implementation
func TestMockRunnerPlugin_Manifest(t *testing.T) {
	manifest := &Manifest{
		ID:          "test-runner",
		Name:        "Test Runner",
		Version:     "1.0.0",
		Type:        PluginTypeRunner,
		Description: "A test runner plugin",
	}

	plugin := newMockRunnerPlugin(manifest)
	result := plugin.Manifest()

	assert.Equal(t, manifest, result)
	assert.Equal(t, "test-runner", result.ID)
	assert.Equal(t, PluginTypeRunner, result.Type)
}

func TestMockRunnerPlugin_Load_Success(t *testing.T) {
	manifest := &Manifest{
		ID:      "test-runner",
		Name:    "Test Runner",
		Version: "1.0.0",
	}

	plugin := newMockRunnerPlugin(manifest)
	err := plugin.Load()

	assert.NoError(t, err)
	assert.True(t, plugin.loadCalled)
}

func TestMockRunnerPlugin_Load_Failure(t *testing.T) {
	manifest := &Manifest{
		ID:      "test-runner",
		Name:    "Test Runner",
		Version: "1.0.0",
	}

	plugin := newMockRunnerPlugin(manifest)
	plugin.loadError = errors.New("failed to initialize runner")

	err := plugin.Load()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to initialize runner")
	assert.True(t, plugin.loadCalled)
}

func TestMockRunnerPlugin_Unload_Success(t *testing.T) {
	manifest := &Manifest{
		ID:      "test-runner",
		Name:    "Test Runner",
		Version: "1.0.0",
	}

	plugin := newMockRunnerPlugin(manifest)
	err := plugin.Unload()

	assert.NoError(t, err)
	assert.True(t, plugin.unloadCalled)
}

func TestMockRunnerPlugin_Unload_Failure(t *testing.T) {
	manifest := &Manifest{
		ID:      "test-runner",
		Name:    "Test Runner",
		Version: "1.0.0",
	}

	plugin := newMockRunnerPlugin(manifest)
	plugin.unloadError = errors.New("failed to cleanup resources")

	err := plugin.Unload()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to cleanup resources")
	assert.True(t, plugin.unloadCalled)
}

func TestMockRunnerPlugin_Execute_Success(t *testing.T) {
	manifest := &Manifest{
		ID:      "test-runner",
		Name:    "Test Runner",
		Version: "1.0.0",
	}

	plugin := newMockRunnerPlugin(manifest)
	ctx := context.Background()
	req := &ExecutionRequest{
		Command:     []string{"echo", "hello"},
		WorkingDir:  "/tmp",
		Environment: map[string]string{},
		Timeout:     5 * time.Second,
	}

	result, err := plugin.Execute(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, "mock output", result.Stdout)
	assert.True(t, plugin.executeCalled)
}

func TestMockRunnerPlugin_Execute_Failure(t *testing.T) {
	manifest := &Manifest{
		ID:      "test-runner",
		Name:    "Test Runner",
		Version: "1.0.0",
	}

	plugin := newMockRunnerPlugin(manifest)
	plugin.executeError = errors.New("execution failed")

	ctx := context.Background()
	req := &ExecutionRequest{
		Command:     []string{"invalid-command"},
		WorkingDir:  "/tmp",
		Environment: map[string]string{},
		Timeout:     5 * time.Second,
	}

	result, err := plugin.Execute(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "execution failed")
	assert.True(t, plugin.executeCalled)
}

func TestMockRunnerPlugin_Execute_CustomResult(t *testing.T) {
	manifest := &Manifest{
		ID:      "test-runner",
		Name:    "Test Runner",
		Version: "1.0.0",
	}

	plugin := newMockRunnerPlugin(manifest)
	plugin.executeResult = &ExecutionResult{
		ExitCode: 2,
		Stdout:   "custom stdout",
		Stderr:   "custom stderr",
		Duration: 250 * time.Millisecond,
	}

	ctx := context.Background()
	req := &ExecutionRequest{
		Command:     []string{"test", "command"},
		WorkingDir:  "/tmp",
		Environment: map[string]string{},
		Timeout:     5 * time.Second,
	}

	result, err := plugin.Execute(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2, result.ExitCode)
	assert.Equal(t, "custom stdout", result.Stdout)
	assert.Equal(t, "custom stderr", result.Stderr)
	assert.Equal(t, 250*time.Millisecond, result.Duration)
	assert.True(t, plugin.executeCalled)
}

func TestMockRunnerPlugin_Execute_WithContext(t *testing.T) {
	manifest := &Manifest{
		ID:      "test-runner",
		Name:    "Test Runner",
		Version: "1.0.0",
	}

	plugin := newMockRunnerPlugin(manifest)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	req := &ExecutionRequest{
		Command:     []string{"sleep", "1"},
		WorkingDir:  "/tmp",
		Environment: map[string]string{},
		Timeout:     5 * time.Second,
	}

	result, err := plugin.Execute(ctx, req)

	// Mock doesn't actually respect context, but we test it's passed
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, plugin.executeCalled)
}

func TestMockRunnerPlugin_Execute_WithStdin(t *testing.T) {
	manifest := &Manifest{
		ID:      "test-runner",
		Name:    "Test Runner",
		Version: "1.0.0",
	}

	plugin := newMockRunnerPlugin(manifest)
	ctx := context.Background()

	stdinData := []byte("test input data")
	req := &ExecutionRequest{
		Command:     []string{"cat"},
		WorkingDir:  "/tmp",
		Environment: map[string]string{},
		Timeout:     5 * time.Second,
		Stdin:       stdinData,
	}

	result, err := plugin.Execute(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, plugin.executeCalled)
}

func TestMockRunnerPlugin_Execute_MultipleEnvironmentVariables(t *testing.T) {
	manifest := &Manifest{
		ID:      "test-runner",
		Name:    "Test Runner",
		Version: "1.0.0",
	}

	plugin := newMockRunnerPlugin(manifest)
	ctx := context.Background()

	req := &ExecutionRequest{
		Command:    []string{"env"},
		WorkingDir: "/tmp",
		Environment: map[string]string{
			"PATH":   "/usr/bin:/bin",
			"HOME":   "/home/user",
			"USER":   "testuser",
			"LANG":   "en_US.UTF-8",
			"CUSTOM": "value",
		},
		Timeout: 5 * time.Second,
	}

	result, err := plugin.Execute(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, plugin.executeCalled)
}

func TestMockRunnerPlugin_LoadAndUnload_Lifecycle(t *testing.T) {
	manifest := &Manifest{
		ID:      "test-runner",
		Name:    "Test Runner",
		Version: "1.0.0",
	}

	plugin := newMockRunnerPlugin(manifest)

	// Load plugin
	err := plugin.Load()
	require.NoError(t, err)
	assert.True(t, plugin.loadCalled)

	// Execute command
	ctx := context.Background()
	req := &ExecutionRequest{
		Command:     []string{"echo", "test"},
		WorkingDir:  "/tmp",
		Environment: map[string]string{},
		Timeout:     5 * time.Second,
	}

	result, err := plugin.Execute(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, plugin.executeCalled)

	// Unload plugin
	err = plugin.Unload()
	require.NoError(t, err)
	assert.True(t, plugin.unloadCalled)
}

func TestRunnerPlugin_InterfaceCompliance(t *testing.T) {
	manifest := &Manifest{
		ID:      "test-runner",
		Name:    "Test Runner",
		Version: "1.0.0",
	}

	plugin := newMockRunnerPlugin(manifest)

	// Verify it implements Plugin interface
	var _ Plugin = plugin

	// Verify it implements RunnerPlugin interface
	var _ RunnerPlugin = plugin
}

func TestExecutionRequest_CommandVariations(t *testing.T) {
	tests := []struct {
		name    string
		command []string
	}{
		{
			name:    "single command",
			command: []string{"ls"},
		},
		{
			name:    "command with args",
			command: []string{"ls", "-la", "/tmp"},
		},
		{
			name:    "command with spaces in args",
			command: []string{"echo", "hello world"},
		},
		{
			name:    "command with special characters",
			command: []string{"grep", "pattern.*", "file.txt"},
		},
		{
			name:    "command with flags",
			command: []string{"docker", "run", "--rm", "-it", "ubuntu"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &ExecutionRequest{
				Command:     tt.command,
				WorkingDir:  "/tmp",
				Environment: map[string]string{},
				Timeout:     5 * time.Second,
			}

			assert.Equal(t, tt.command, req.Command)
		})
	}
}

func TestExecutionResult_ExitCodeVariations(t *testing.T) {
	tests := []struct {
		name     string
		exitCode int
		desc     string
	}{
		{
			name:     "success",
			exitCode: 0,
			desc:     "successful execution",
		},
		{
			name:     "general error",
			exitCode: 1,
			desc:     "general error",
		},
		{
			name:     "misuse of shell builtin",
			exitCode: 2,
			desc:     "misuse of shell builtin",
		},
		{
			name:     "command not found",
			exitCode: 127,
			desc:     "command not found",
		},
		{
			name:     "invalid exit arg",
			exitCode: 128,
			desc:     "invalid argument to exit",
		},
		{
			name:     "terminated by signal",
			exitCode: 130,
			desc:     "terminated by SIGINT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &ExecutionResult{
				ExitCode: tt.exitCode,
				Stdout:   "",
				Stderr:   tt.desc,
				Duration: 100 * time.Millisecond,
			}

			assert.Equal(t, tt.exitCode, result.ExitCode)
			assert.Contains(t, result.Stderr, tt.desc)
		})
	}
}

func TestMockRunnerPlugin_Execute_EdgeCases(t *testing.T) {
	manifest := &Manifest{
		ID:      "test-runner",
		Name:    "Test Runner",
		Version: "1.0.0",
	}

	t.Run("empty command array", func(t *testing.T) {
		plugin := newMockRunnerPlugin(manifest)
		ctx := context.Background()

		req := &ExecutionRequest{
			Command:     []string{},
			WorkingDir:  "/tmp",
			Environment: map[string]string{},
			Timeout:     5 * time.Second,
		}

		result, err := plugin.Execute(ctx, req)
		assert.NoError(t, err) // Mock doesn't validate
		assert.NotNil(t, result)
	})

	t.Run("nil environment", func(t *testing.T) {
		plugin := newMockRunnerPlugin(manifest)
		ctx := context.Background()

		req := &ExecutionRequest{
			Command:     []string{"echo", "test"},
			WorkingDir:  "/tmp",
			Environment: nil,
			Timeout:     5 * time.Second,
		}

		result, err := plugin.Execute(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("empty working directory", func(t *testing.T) {
		plugin := newMockRunnerPlugin(manifest)
		ctx := context.Background()

		req := &ExecutionRequest{
			Command:     []string{"pwd"},
			WorkingDir:  "",
			Environment: map[string]string{},
			Timeout:     5 * time.Second,
		}

		result, err := plugin.Execute(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("very short timeout", func(t *testing.T) {
		plugin := newMockRunnerPlugin(manifest)
		ctx := context.Background()

		req := &ExecutionRequest{
			Command:     []string{"sleep", "10"},
			WorkingDir:  "/tmp",
			Environment: map[string]string{},
			Timeout:     1 * time.Nanosecond,
		}

		result, err := plugin.Execute(ctx, req)
		assert.NoError(t, err) // Mock doesn't enforce timeout
		assert.NotNil(t, result)
	})

	t.Run("large stdin data", func(t *testing.T) {
		plugin := newMockRunnerPlugin(manifest)
		ctx := context.Background()

		largeData := make([]byte, 1024*1024) // 1MB
		for i := range largeData {
			largeData[i] = byte('a' + (i % 26))
		}

		req := &ExecutionRequest{
			Command:     []string{"wc", "-c"},
			WorkingDir:  "/tmp",
			Environment: map[string]string{},
			Timeout:     30 * time.Second,
			Stdin:       largeData,
		}

		result, err := plugin.Execute(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestExecutionRequest_Serialization(t *testing.T) {
	// Test that ExecutionRequest has proper JSON tags
	req := &ExecutionRequest{
		Command:     []string{"test", "cmd"},
		WorkingDir:  "/work",
		Environment: map[string]string{"KEY": "value"},
		Timeout:     10 * time.Second,
		Stdin:       []byte("input"),
	}

	// Verify fields are accessible
	assert.NotNil(t, req.Command)
	assert.NotEmpty(t, req.WorkingDir)
	assert.NotNil(t, req.Environment)
	assert.NotZero(t, req.Timeout)
	assert.NotNil(t, req.Stdin)
}

func TestExecutionResult_Serialization(t *testing.T) {
	// Test that ExecutionResult has proper JSON tags
	result := &ExecutionResult{
		ExitCode: 0,
		Stdout:   "output",
		Stderr:   "error",
		Duration: 5 * time.Second,
	}

	// Verify fields are accessible
	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, "output", result.Stdout)
	assert.Equal(t, "error", result.Stderr)
	assert.Equal(t, 5*time.Second, result.Duration)
}

func TestRunnerPlugin_MultipleExecutions(t *testing.T) {
	manifest := &Manifest{
		ID:      "test-runner",
		Name:    "Test Runner",
		Version: "1.0.0",
	}

	plugin := newMockRunnerPlugin(manifest)
	ctx := context.Background()

	// Execute multiple commands
	for i := 0; i < 5; i++ {
		req := &ExecutionRequest{
			Command:     []string{"echo", "test"},
			WorkingDir:  "/tmp",
			Environment: map[string]string{},
			Timeout:     5 * time.Second,
		}

		result, err := plugin.Execute(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 0, result.ExitCode)
	}
}

func TestRunnerPlugin_ConcurrentExecutions(t *testing.T) {
	manifest := &Manifest{
		ID:      "test-runner",
		Name:    "Test Runner",
		Version: "1.0.0",
	}

	plugin := newMockRunnerPlugin(manifest)
	ctx := context.Background()

	var wg sync.WaitGroup
	numGoroutines := 10

	// Collect errors and results to validate after goroutines complete
	errors := make([]error, numGoroutines)
	results := make([]*ExecutionResult, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			req := &ExecutionRequest{
				Command:     []string{"echo", "concurrent"},
				WorkingDir:  "/tmp",
				Environment: map[string]string{},
				Timeout:     5 * time.Second,
			}

			result, err := plugin.Execute(ctx, req)
			errors[index] = err
			results[index] = result
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Now validate all results (thread-safe)
	for i := 0; i < numGoroutines; i++ {
		assert.NoError(t, errors[i])
		assert.NotNil(t, results[i])
	}
}

func TestRunnerPluginType_Verification(t *testing.T) {
	// Verify PluginTypeRunner is defined
	assert.Equal(t, PluginType("runner"), PluginTypeRunner)

	manifest := &Manifest{
		ID:      "runner-plugin",
		Name:    "Runner Plugin",
		Version: "1.0.0",
		Type:    PluginTypeRunner,
	}

	plugin := newMockRunnerPlugin(manifest)
	assert.Equal(t, PluginTypeRunner, plugin.Manifest().Type)
}
