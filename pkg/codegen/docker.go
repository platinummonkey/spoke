package codegen

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// DockerRequest represents a Docker execution request
type DockerRequest struct {
	Image       string
	Tag         string
	ProtoFiles  []ProtoFile
	ProtocFlags []string
	Timeout     time.Duration
}

// DockerResult represents the result of Docker execution
type DockerResult struct {
	GeneratedFiles []GeneratedFile
	Duration       time.Duration
	ExitCode       int
	Stdout         string
	Stderr         string
}

// ExecuteDocker runs protoc in a Docker container using os/exec
func ExecuteDocker(ctx context.Context, req *DockerRequest) (*DockerResult, error) {
	startTime := time.Now()
	result := &DockerResult{}

	// Create temporary directories
	inputDir, err := os.MkdirTemp("", "spoke-input-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create input directory: %w", err)
	}
	defer os.RemoveAll(inputDir)

	outputDir, err := os.MkdirTemp("", "spoke-output-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}
	defer os.RemoveAll(outputDir)

	// Write proto files to input directory
	for _, protoFile := range req.ProtoFiles {
		filePath := filepath.Join(inputDir, protoFile.Path)
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory: %w", err)
		}
		if err := os.WriteFile(filePath, protoFile.Content, 0644); err != nil {
			return nil, fmt.Errorf("failed to write proto file: %w", err)
		}
	}

	// Build Docker command
	image := req.Image
	if req.Tag != "" {
		image = fmt.Sprintf("%s:%s", req.Image, req.Tag)
	}

	// Build protoc command arguments
	protocCmd := []string{"protoc", "--proto_path=/input"}
	protocCmd = append(protocCmd, req.ProtocFlags...)
	for _, protoFile := range req.ProtoFiles {
		protocCmd = append(protocCmd, "/input/"+protoFile.Path)
	}

	// Build docker run command
	args := []string{
		"run",
		"--rm",
		"-v", fmt.Sprintf("%s:/input:ro", inputDir),
		"-v", fmt.Sprintf("%s:/output", outputDir),
		image,
	}
	args = append(args, protocCmd...)

	// Set timeout
	timeout := req.Timeout
	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Execute Docker command
	cmd := exec.CommandContext(execCtx, "docker", args...)
	output, err := cmd.CombinedOutput()
	result.Duration = time.Since(startTime)
	result.Stdout = string(output)
	result.Stderr = string(output)

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		}
		return result, fmt.Errorf("docker execution failed: %w: %s", err, output)
	}

	// Extract generated files
	generatedFiles, err := readGeneratedFiles(outputDir)
	if err != nil {
		return result, fmt.Errorf("failed to read generated files: %w", err)
	}

	if len(generatedFiles) == 0 {
		return result, fmt.Errorf("no files were generated")
	}

	result.GeneratedFiles = generatedFiles
	return result, nil
}

// readGeneratedFiles reads all files from the output directory
func readGeneratedFiles(outputDir string) ([]GeneratedFile, error) {
	var files []GeneratedFile

	err := filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(outputDir, path)
		if err != nil {
			return err
		}

		files = append(files, GeneratedFile{
			Path:    relPath,
			Content: content,
			Size:    info.Size(),
		})

		return nil
	})

	return files, err
}
