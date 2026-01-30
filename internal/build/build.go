package build

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cozy-creator/cozyctl/internal/api"
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
	toolsCozyConfig, err := GetToolsCozyConfig(filepath.Join(directoryPath, PyProjectTomlPath))
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

func BuildProjectOnServer(projectDir string) error {
	// Validate directory
	projectDir, err := filepath.Abs(projectDir)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	info, err := os.Stat(projectDir)
	if err != nil {
		return fmt.Errorf("cannot access path: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", projectDir)
	}

	// Check pyproject.toml exists
	pyprojectPath := filepath.Join(projectDir, PyProjectTomlPath)
	if _, err := os.Stat(pyprojectPath); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("directory does not contain %s", PyProjectTomlPath)
	}

	// Load config for builder URL and token
	defaultCfg, err := config.GetDefaultConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	profileCfg, err := config.GetProfileConfig(defaultCfg.CurrentName, defaultCfg.CurrentProfile)
	if err != nil {
		return fmt.Errorf("failed to load profile config: %w", err)
	}

	if profileCfg.Config == nil {
		return fmt.Errorf("not logged in (run 'cozyctl login' first)")
	}

	if err := profileCfg.Config.Validate(); err != nil {
		return err
	}

	builderURL := profileCfg.Config.BuilderURL
	if builderURL == "" {
		builderURL = config.DefaultConfigData().BuilderURL
	}

	// Create tarball
	fmt.Println("Creating tarball...")
	tarball, err := CreateTarball(projectDir)
	if err != nil {
		return fmt.Errorf("failed to create tarball: %w", err)
	}
	fmt.Printf("Tarball size: %d bytes\n", tarball.Len())

	// Use directory name as build name
	buildName := filepath.Base(projectDir)

	// Upload to gen-builder
	client := api.NewBuilderClient(builderURL, profileCfg.Config.Token)

	fmt.Printf("Uploading to gen-builder at %s...\n", builderURL)
	buildResp, err := client.UploadBuild(tarball, buildName)
	if err != nil {
		return fmt.Errorf("failed to upload build: %w", err)
	}

	fmt.Printf("Build submitted: ID=%s, Status=%s\n", buildResp.BuildID, buildResp.Status)

	// Poll for completion
	fmt.Println("\nWaiting for build to complete...")
	pollInterval := 5 * time.Second
	pollTimeout := 4 * time.Hour
	deadline := time.Now().Add(pollTimeout)
	lastStatus := ""

	for time.Now().Before(deadline) {
		status, err := client.GetBuildStatus(buildResp.BuildID)
		if err != nil {
			fmt.Printf("  Warning: failed to get status: %v\n", err)
			time.Sleep(pollInterval)
			continue
		}

		if status.Status != lastStatus {
			fmt.Printf("  Status: %s\n", status.Status)
			lastStatus = status.Status
		}

		switch status.Status {
		case "success":
			fmt.Printf("\nBuild completed successfully!\n")
			fmt.Printf("  Build ID:  %s\n", status.ID)
			fmt.Printf("  Image Tag: %s\n", status.ImageTag)
			if status.LogsPath != "" {
				fmt.Printf("  Logs:      %s\n", status.LogsPath)
			}
			return nil

		case "failed":
			errMsg := status.Error
			if errMsg == "" {
				errMsg = "unknown error"
			}
			return fmt.Errorf("build failed: %s", errMsg)

		case "pending", "running":
			time.Sleep(pollInterval)
			continue

		default:
			fmt.Printf("  Unknown status: %s\n", status.Status)
			time.Sleep(pollInterval)
		}
	}

	return fmt.Errorf("build timed out after %v (build ID: %s)", pollTimeout, buildResp.BuildID)
}
