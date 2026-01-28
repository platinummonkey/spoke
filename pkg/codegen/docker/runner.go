package docker

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/platinummonkey/spoke/pkg/codegen"
)

// DockerRunner implements the Runner interface using Docker
type DockerRunner struct {
	client      *client.Client
	imageCache  map[string]bool // Track pulled images
	cleanupIDs  []string        // Container IDs to cleanup
}

// NewDockerRunner creates a new Docker runner
func NewDockerRunner() (*DockerRunner, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDockerNotAvailable, err)
	}

	// Verify Docker is available
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = cli.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDockerNotAvailable, err)
	}

	return &DockerRunner{
		client:     cli,
		imageCache: make(map[string]bool),
		cleanupIDs: make([]string, 0),
	}, nil
}

// Execute runs a compilation in a Docker container
func (r *DockerRunner) Execute(ctx context.Context, req *ExecutionRequest) (*ExecutionResult, error) {
	result := &ExecutionResult{
		Success: false,
	}

	startTime := time.Now()
	defer func() {
		result.Duration = time.Since(startTime)
	}()

	// Set defaults
	if req.MemoryLimit == 0 {
		req.MemoryLimit = DefaultMemoryLimit
	}
	if req.CPULimit == 0 {
		req.CPULimit = DefaultCPULimit
	}
	if req.Timeout == 0 {
		req.Timeout = DefaultTimeout
	}

	// Pull image if not cached
	fullImage := req.Image
	if req.Tag != "" {
		fullImage = req.Image + ":" + req.Tag
	}

	if err := r.PullImage(ctx, fullImage); err != nil {
		result.Error = fmt.Errorf("%w: %v", ErrImagePullFailed, err)
		return result, result.Error
	}

	// Create temporary directories for input and output
	inputDir, err := os.MkdirTemp("", "spoke-docker-input-*")
	if err != nil {
		result.Error = fmt.Errorf("failed to create input directory: %v", err)
		return result, result.Error
	}
	defer os.RemoveAll(inputDir)

	outputDir, err := os.MkdirTemp("", "spoke-docker-output-*")
	if err != nil {
		result.Error = fmt.Errorf("failed to create output directory: %v", err)
		return result, result.Error
	}
	defer os.RemoveAll(outputDir)

	// Write proto files to input directory
	for _, protoFile := range req.ProtoFiles {
		filePath := filepath.Join(inputDir, protoFile.Path)
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			result.Error = fmt.Errorf("failed to create proto file directory: %v", err)
			return result, result.Error
		}
		if err := os.WriteFile(filePath, protoFile.Content, 0644); err != nil {
			result.Error = fmt.Errorf("failed to write proto file: %v", err)
			return result, result.Error
		}
	}

	// Build protoc command
	cmd := r.buildProtocCommand(req)

	// Create container
	containerID, err := r.createContainer(ctx, fullImage, cmd, inputDir, outputDir, req)
	if err != nil {
		result.Error = fmt.Errorf("%w: %v", ErrContainerFailed, err)
		return result, result.Error
	}
	r.cleanupIDs = append(r.cleanupIDs, containerID)

	// Start container
	if err := r.client.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		result.Error = fmt.Errorf("%w: start failed: %v", ErrContainerFailed, err)
		return result, result.Error
	}

	// Wait for container with timeout
	execCtx, cancel := context.WithTimeout(ctx, req.Timeout)
	defer cancel()

	statusCh, errCh := r.client.ContainerWait(execCtx, containerID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			result.Error = fmt.Errorf("%w: wait failed: %v", ErrContainerFailed, err)
			return result, result.Error
		}
	case status := <-statusCh:
		result.ExitCode = int(status.StatusCode)
	case <-execCtx.Done():
		result.Error = ErrTimeout
		return result, result.Error
	}

	// Get container logs
	logs, err := r.client.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err == nil {
		var stdout, stderr bytes.Buffer
		stdcopy.StdCopy(&stdout, &stderr, logs)
		result.Stdout = stdout.String()
		result.Stderr = stderr.String()
		logs.Close()
	}

	// Check exit code
	if result.ExitCode != 0 {
		result.Error = fmt.Errorf("%w: exit code %d: %s", ErrContainerFailed, result.ExitCode, result.Stderr)
		return result, result.Error
	}

	// Extract generated files from output directory
	generatedFiles, err := r.extractGeneratedFiles(outputDir)
	if err != nil {
		result.Error = fmt.Errorf("failed to extract generated files: %v", err)
		return result, result.Error
	}

	if len(generatedFiles) == 0 {
		result.Error = ErrNoGeneratedFiles
		return result, result.Error
	}

	result.GeneratedFiles = generatedFiles
	result.Success = true
	return result, nil
}

// PullImage ensures the Docker image is available locally
func (r *DockerRunner) PullImage(ctx context.Context, imageRef string) error {
	// Check cache
	if r.imageCache[imageRef] {
		return nil
	}

	// Check if image exists locally
	_, err := r.client.ImageInspect(ctx, imageRef)
	if err == nil {
		r.imageCache[imageRef] = true
		return nil
	}

	// Pull image
	pullCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	reader, err := r.client.ImagePull(pullCtx, imageRef, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %v", imageRef, err)
	}
	defer reader.Close()

	// Read pull output to completion
	io.Copy(io.Discard, reader)

	r.imageCache[imageRef] = true
	return nil
}

// Cleanup removes stopped containers and unused images
func (r *DockerRunner) Cleanup(ctx context.Context) error {
	for _, containerID := range r.cleanupIDs {
		// Remove container (force remove if still running)
		r.client.ContainerRemove(ctx, containerID, container.RemoveOptions{
			Force:         true,
			RemoveVolumes: true,
		})
	}
	r.cleanupIDs = make([]string, 0)
	return nil
}

// Close releases resources
func (r *DockerRunner) Close() error {
	// Cleanup any remaining containers
	if err := r.Cleanup(context.Background()); err != nil {
		return err
	}

	if r.client != nil {
		return r.client.Close()
	}
	return nil
}

// createContainer creates a Docker container with the specified configuration
func (r *DockerRunner) createContainer(ctx context.Context, imageRef string, cmd []string,
	inputDir, outputDir string, req *ExecutionRequest) (string, error) {

	// Build environment variables
	env := make([]string, 0)
	for k, v := range req.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	// Container configuration
	config := &container.Config{
		Image:        imageRef,
		Cmd:          cmd,
		Env:          env,
		WorkingDir:   "/workspace",
		AttachStdout: true,
		AttachStderr: true,
	}

	// Host configuration with resource limits
	hostConfig := &container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:/input:ro", inputDir),
			fmt.Sprintf("%s:/output", outputDir),
		},
		Resources: container.Resources{
			Memory:   req.MemoryLimit,
			NanoCPUs: int64(req.CPULimit * 1e9),
		},
		AutoRemove: false, // We'll remove manually after extracting files
	}

	// Create container
	resp, err := r.client.ContainerCreate(ctx, config, hostConfig, nil, nil, "")
	if err != nil {
		return "", fmt.Errorf("failed to create container: %v", err)
	}

	return resp.ID, nil
}

// buildProtocCommand builds the protoc command based on the execution request
func (r *DockerRunner) buildProtocCommand(req *ExecutionRequest) []string {
	cmd := []string{"protoc"}

	// Add proto path
	cmd = append(cmd, "--proto_path=/input")

	// Add custom flags
	cmd = append(cmd, req.ProtocFlags...)

	// Add proto files
	for _, protoFile := range req.ProtoFiles {
		cmd = append(cmd, "/input/"+protoFile.Path)
	}

	return cmd
}

// extractGeneratedFiles reads all files from the output directory
func (r *DockerRunner) extractGeneratedFiles(outputDir string) ([]codegen.GeneratedFile, error) {
	var files []codegen.GeneratedFile

	err := filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %v", path, err)
		}

		// Get relative path
		relPath, err := filepath.Rel(outputDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path for %s: %v", path, err)
		}

		files = append(files, codegen.GeneratedFile{
			Path:    relPath,
			Content: content,
			Size:    info.Size(),
		})

		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

// copyToContainer copies data to a container (alternative to volume mounts)
func (r *DockerRunner) copyToContainer(ctx context.Context, containerID string, srcPath, dstPath string) error {
	// Create tar archive
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	err := filepath.Walk(srcPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(srcPath, path)
		if err != nil {
			return err
		}
		header.Name = strings.TrimPrefix(filepath.Join(dstPath, relPath), "/")

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			if _, err := io.Copy(tw, file); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	if err := tw.Close(); err != nil {
		return err
	}

	// Copy tar to container
	return r.client.CopyToContainer(ctx, containerID, "/", &buf, container.CopyToContainerOptions{})
}
