package build

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cozy-creator/cozyctl/internal/config"
	"github.com/google/uuid"
)

const (
	PyProjectTomlPath = "pyproject.toml"
)

func BuildProjectLocally(directoryPath string) error {

	// First sanitize the directoryPath and find the directory.
	directoryPath, err := filepath.Abs(directoryPath)
	if err != nil {
		return err
	}

	// Exists or not verify
	info, err := os.Stat(directoryPath)
	if err != nil {
		return fmt.Errorf("cannot access path: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", directoryPath)
	}

	// Find the pyproject.toml file, and send it to build via template
	if _, err = os.Stat(filepath.Join(directoryPath, PyProjectTomlPath)); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("the directory does not contain %sfile. Please check it again.", PyProjectTomlPath)
	}

	// Send it to parse this toml, return contents for tools.cozy so that build template data can be validated.
	toolsCozyConfig, err := getToolsCozyConfig(filepath.Join(directoryPath, PyProjectTomlPath))
	if err != nil {
		return err
	}

	// Resolve the appropriate base image
	baseImage, err := ResolveBaseImage(toolsCozyConfig)
	if err != nil {
		return fmt.Errorf("failed to resolve base image: %w", err)
	}
	fmt.Printf("Using base image: %s\n", baseImage)

	// Generate Dockerfile from template
	dockerfile, err := GenerateDockerfile(baseImage, toolsCozyConfig)
	if err != nil {
		return fmt.Errorf("failed to generate Dockerfile: %w", err)
	}

	// Write Dockerfile to the project directory
	dockerfilePath := filepath.Join(directoryPath, "Dockerfile")
	if err := os.WriteFile(dockerfilePath, []byte(dockerfile), 0644); err != nil {
		return fmt.Errorf("failed to write Dockerfile: %w", err)
	}
	fmt.Printf("Generated Dockerfile at: %s\n", dockerfilePath)

	// Generate unique build ID and image tag
	buildID := uuid.New().String()
	imageTag := GenerateImageTag(buildID, toolsCozyConfig.DeploymentID)
	fmt.Printf("Building image: %s\n", imageTag)

	// Build the Docker image
	builder := NewDockerBuilder()
	ctx := context.Background()
	buildTimeout := 30 * time.Minute

	fmt.Println("Starting Docker build...")
	result := builder.Build(ctx, directoryPath, imageTag, buildTimeout)

	// Print build logs
	if result.Logs != "" {
		fmt.Println("\n--- Build Logs ---")
		fmt.Println(result.Logs)
		fmt.Println("--- End Build Logs ---")
	}

	if result.Error != nil {
		return fmt.Errorf("docker build failed: %w", result.Error)
	}

	fmt.Printf("Build completed successfully in %v\n", result.Duration)
	fmt.Printf("Image tag: %s\n", result.ImageTag)

	return nil
}

func BuildProjectOnServer(config *config.DefaultConfig) error {
	return nil
}
