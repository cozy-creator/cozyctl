package update

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cozy-creator/cozyctl/internal/api"
	"github.com/cozy-creator/cozyctl/internal/build"
	"github.com/cozy-creator/cozyctl/internal/config"
	"github.com/google/uuid"
)

// Options contains the options for updating a deployment.
type Options struct {
	ProjectPath string
	DryRun      bool
	Functions   string
	MinWorkers  int
	MaxWorkers  int
	ImageOnly   bool
}

// Run executes the update process: rebuild image and update existing deployment.
func Run(opts Options) error {
	// Get absolute path
	absPath, err := filepath.Abs(opts.ProjectPath)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Verify directory exists
	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("cannot access path: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", absPath)
	}

	// Check for pyproject.toml
	pyprojectPath := filepath.Join(absPath, "pyproject.toml")
	if _, err := os.Stat(pyprojectPath); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("pyproject.toml not found in %s", absPath)
	}

	// Parse pyproject.toml
	cozyConfig, err := build.GetToolsCozyConfig(pyprojectPath)
	if err != nil {
		return fmt.Errorf("failed to parse pyproject.toml: %w", err)
	}

	if cozyConfig.DeploymentID == "" {
		return fmt.Errorf("[tool.cozy] deployment-id is required in pyproject.toml")
	}

	fmt.Printf("Deployment ID: %s\n", cozyConfig.DeploymentID)

	// Load config for API access
	defaultCfg, err := config.GetDefaultConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	profileCfg, err := config.GetProfileConfig(defaultCfg.CurrentName, defaultCfg.CurrentProfile)
	if err != nil {
		return fmt.Errorf("failed to load profile config: %w", err)
	}

	if profileCfg.Config == nil || profileCfg.Config.Token == "" {
		return fmt.Errorf("not logged in (run 'cozyctl login' first)")
	}

	orchestratorURL := profileCfg.Config.OrchestratorURL
	if orchestratorURL == "" {
		orchestratorURL = config.DefaultConfigData().OrchestratorURL
	}

	// Create API client
	client := api.NewClient(orchestratorURL, profileCfg.Config.Token)

	// Check if deployment exists
	existing, err := client.GetDeployment(cozyConfig.DeploymentID)
	if err != nil {
		return fmt.Errorf("failed to check deployment: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("deployment '%s' not found (use 'cozyctl deploy' to create)", cozyConfig.DeploymentID)
	}

	fmt.Printf("Found existing deployment: %s\n", existing.ID)

	// Detect or parse functions (priority: flag > pyproject.toml > auto-detect)
	var functions []build.DetectedFunction
	if !opts.ImageOnly {
		if opts.Functions != "" {
			// 1. From command-line flag
			functions, err = build.ParseFunctionsFromFlag(opts.Functions)
			if err != nil {
				return fmt.Errorf("failed to parse --functions: %w", err)
			}
			fmt.Printf("Using functions from flag: %d function(s)\n", len(functions))
		} else if len(cozyConfig.Functions) > 0 {
			// 2. From pyproject.toml [tool.cozy.functions]
			for name, cfg := range cozyConfig.Functions {
				functions = append(functions, build.DetectedFunction{
					Name:        name,
					RequiresGPU: cfg.RequiresGPU,
				})
			}
			fmt.Printf("Using functions from pyproject.toml: %d function(s)\n", len(functions))
			for _, fn := range functions {
				gpuStr := "CPU"
				if fn.RequiresGPU {
					gpuStr = "GPU"
				}
				fmt.Printf("  - %s (%s)\n", fn.Name, gpuStr)
			}
		} else {
			// 3. Auto-detect from Python source
			functions, err = build.DetectWorkerFunctions(absPath)
			if err != nil {
				return fmt.Errorf("failed to detect functions: %w", err)
			}
			if len(functions) == 0 {
				fmt.Println("Warning: No @worker_function() decorated functions detected")
			} else {
				fmt.Printf("Auto-detected %d function(s):\n", len(functions))
				for _, fn := range functions {
					gpuStr := "CPU"
					if fn.RequiresGPU {
						gpuStr = "GPU"
					}
					fmt.Printf("  - %s (%s)\n", fn.Name, gpuStr)
				}
			}
		}
	}

	// Resolve base image
	baseImage, err := build.ResolveBaseImage(cozyConfig)
	if err != nil {
		return fmt.Errorf("failed to resolve base image: %w", err)
	}
	fmt.Printf("Base image: %s\n", baseImage)

	// Generate Dockerfile
	dockerfile, err := build.GenerateDockerfile(baseImage, cozyConfig)
	if err != nil {
		return fmt.Errorf("failed to generate Dockerfile: %w", err)
	}

	// Generate build ID and image tag
	buildID := uuid.New().String()
	imageTag := build.GenerateImageTag(buildID, cozyConfig.DeploymentID)
	fmt.Printf("Image tag: %s\n", imageTag)

	if opts.DryRun {
		fmt.Println("\n--- Dry Run Mode ---")
		fmt.Println("Would generate Dockerfile:")
		fmt.Println(dockerfile)
		fmt.Println("\nWould build image:", imageTag)
		fmt.Println("Would update deployment:", cozyConfig.DeploymentID)
		return nil
	}

	// Write Dockerfile
	dockerfilePath := filepath.Join(absPath, "Dockerfile")
	if err := os.WriteFile(dockerfilePath, []byte(dockerfile), 0644); err != nil {
		return fmt.Errorf("failed to write Dockerfile: %w", err)
	}
	fmt.Printf("Generated Dockerfile: %s\n", dockerfilePath)

	// Build Docker image
	fmt.Println("\nBuilding Docker image...")
	builder := build.NewDockerBuilder()
	ctx := context.Background()
	buildTimeout := 30 * time.Minute

	result := builder.Build(ctx, absPath, imageTag, buildTimeout)

	if result.Logs != "" {
		fmt.Println("\n--- Build Logs ---")
		fmt.Println(result.Logs)
		fmt.Println("--- End Build Logs ---")
	}

	if result.Error != nil {
		return fmt.Errorf("docker build failed: %w", result.Error)
	}

	fmt.Printf("\nBuild completed in %v\n", result.Duration)
	fmt.Printf("Image: %s\n", result.ImageTag)

	// Update deployment
	fmt.Println("\nUpdating deployment...")

	req := &api.UpdateDeploymentRequest{
		ImageURL: imageTag,
	}

	// Update functions if not image-only
	if !opts.ImageOnly && len(functions) > 0 {
		funcReqs := make([]api.FunctionRequirement, len(functions))
		for i, fn := range functions {
			funcReqs[i] = api.FunctionRequirement{
				Name:        fn.Name,
				RequiresGPU: fn.RequiresGPU,
			}
		}
		req.FunctionRequirements = funcReqs
	}

	// Update worker counts if specified
	if opts.MinWorkers >= 0 {
		req.MinWorkers = &opts.MinWorkers
	}
	if opts.MaxWorkers >= 0 {
		req.MaxWorkers = &opts.MaxWorkers
	}

	deployment, err := client.UpdateDeployment(cozyConfig.DeploymentID, req)
	if err != nil {
		return fmt.Errorf("failed to update deployment: %w", err)
	}

	fmt.Printf("\nDeployment updated successfully!\n")
	fmt.Printf("  ID: %s\n", deployment.ID)
	fmt.Printf("  Tenant: %s\n", deployment.TenantID)
	fmt.Printf("  Image: %s\n", deployment.ImageURL)
	fmt.Printf("  Functions: %d\n", len(deployment.FunctionRequirements))

	fmt.Println("\nUpdate completed successfully!")
	return nil
}
