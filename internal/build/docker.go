package build

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

// DockerBuilder wraps Docker CLI commands
type DockerBuilder struct {
	registryURL    string
	registryUser   string
	registryPass   string
	registryPrefix string
}

// DockerBuilderOption is a functional option for configuring DockerBuilder
type DockerBuilderOption func(*DockerBuilder)

// WithRegistryURL sets the registry URL for docker login
func WithRegistryURL(url string) DockerBuilderOption {
	return func(d *DockerBuilder) {
		d.registryURL = url
	}
}

// WithRegistryCredentials sets the registry username and password
func WithRegistryCredentials(user, pass string) DockerBuilderOption {
	return func(d *DockerBuilder) {
		d.registryUser = user
		d.registryPass = pass
	}
}

// WithRegistryPrefix sets the registry prefix for image tagging (e.g., "docker.io/myuser/")
func WithRegistryPrefix(prefix string) DockerBuilderOption {
	return func(d *DockerBuilder) {
		d.registryPrefix = prefix
	}
}

// NewDockerBuilder creates a new DockerBuilder with functional options
func NewDockerBuilder(opts ...DockerBuilderOption) *DockerBuilder {
	d := &DockerBuilder{}

	for _, opt := range opts {
		opt(d)
	}

	return d
}

// Login performs docker login to the private registry (if credentials provided)
func (d *DockerBuilder) Login(ctx context.Context) error {
	if d.registryUser == "" || d.registryPass == "" {
		return nil // No credentials, skip login
	}

	cmd := exec.CommandContext(ctx, "docker", "login",
		"-u", d.registryUser,
		"--password-stdin",
		d.registryURL,
	)
	cmd.Stdin = strings.NewReader(d.registryPass)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker login failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// BuildResult contains the result of a docker build
type BuildResult struct {
	ImageTag string
	Logs     string
	Duration time.Duration
	Error    error
}

// Build executes docker build in the specified directory
func (d *DockerBuilder) Build(ctx context.Context, buildDir string, imageTag string, timeout time.Duration) *BuildResult {
	result := &BuildResult{
		ImageTag: imageTag,
	}

	start := time.Now()

	// Create context with timeout
	buildCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(buildCtx, "docker", "build",
		"-t", imageTag,
		"--progress=plain", // Plain output for logs
		".",
	)
	cmd.Dir = buildDir

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = io.MultiWriter(&stderr, &stdout) // Combine for logs

	err := cmd.Run()
	result.Duration = time.Since(start)
	result.Logs = stdout.String()

	if buildCtx.Err() == context.DeadlineExceeded {
		result.Error = fmt.Errorf("build timed out after %v", timeout)
		return result
	}

	if err != nil {
		result.Error = fmt.Errorf("docker build failed: %w\nStderr: %s",
			err, stderr.String())
		return result
	}

	return result
}

// GenerateImageTag creates a unique image tag for the build
func GenerateImageTag(buildID string, deploymentID string) string {
	// Format: cozy-build-{deployment-id}-{build-id-short}
	shortBuildID := buildID
	if len(buildID) > 8 {
		shortBuildID = buildID[:8]
	}

	if deploymentID != "" {
		return fmt.Sprintf("cozy-build-%s-%s", deploymentID, shortBuildID)
	}
	return fmt.Sprintf("cozy-build-%s", shortBuildID)
}

// TagResult contains the result of a docker tag operation
type TagResult struct {
	SourceTag string
	TargetTag string
	Error     error
}

// Tag creates a new tag for an existing image
func (d *DockerBuilder) Tag(ctx context.Context, sourceTag, targetTag string) *TagResult {
	result := &TagResult{
		SourceTag: sourceTag,
		TargetTag: targetTag,
	}

	cmd := exec.CommandContext(ctx, "docker", "tag", sourceTag, targetTag)

	output, err := cmd.CombinedOutput()
	if err != nil {
		result.Error = fmt.Errorf("docker tag failed: %w\nOutput: %s", err, string(output))
		return result
	}

	return result
}

// PushResult contains the result of a docker push operation
type PushResult struct {
	ImageTag string
	Logs     string
	Duration time.Duration
	Error    error
}

// Push pushes an image to the registry
func (d *DockerBuilder) Push(ctx context.Context, imageTag string, timeout time.Duration) *PushResult {
	result := &PushResult{
		ImageTag: imageTag,
	}

	start := time.Now()

	// Create context with timeout
	pushCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(pushCtx, "docker", "push", imageTag)

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = io.MultiWriter(&stderr, &stdout)

	err := cmd.Run()
	result.Duration = time.Since(start)
	result.Logs = stdout.String()

	if pushCtx.Err() == context.DeadlineExceeded {
		result.Error = fmt.Errorf("push timed out after %v", timeout)
		return result
	}

	if err != nil {
		result.Error = fmt.Errorf("docker push failed: %w\nStderr: %s", err, stderr.String())
		return result
	}

	return result
}

// GetRegistryTag returns the full registry-prefixed tag for an image
func (d *DockerBuilder) GetRegistryTag(localTag string) string {
	if d.registryPrefix == "" {
		return localTag
	}
	// Ensure prefix ends with /
	prefix := d.registryPrefix
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	return prefix + localTag
}

// HasRegistryConfig returns true if registry push is configured
func (d *DockerBuilder) HasRegistryConfig() bool {
	return d.registryPrefix != ""
}
