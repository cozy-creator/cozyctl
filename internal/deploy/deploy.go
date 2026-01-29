package deploy

import (
	"fmt"

	"github.com/cozy-creator/cozyctl/internal/api"
	"github.com/cozy-creator/cozyctl/internal/config"
)

// Run executes the deploy process: send build-id to orchestrator.
func Run(buildID string) error {
	// Load config for tenant-id and orchestrator URL
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

	tenantID := profileCfg.Config.TenantID
	fmt.Printf("Tenant ID: %s\n", tenantID)
	fmt.Printf("Build ID: %s\n", buildID)

	// Get orchestrator URL
	orchestratorURL := profileCfg.Config.OrchestratorURL
	if orchestratorURL == "" {
		orchestratorURL = config.DefaultConfigData().OrchestratorURL
	}

	// Create API client
	client := api.NewClient(orchestratorURL, profileCfg.Config.Token)

	// Deploy with build ID
	fmt.Println("\nDeploying to orchestrator...")
	req := &api.DeployWithBuildIDRequest{
		BuildID:  buildID,
		TenantID: tenantID,
	}

	deployment, err := client.DeployWithBuildID(req)
	if err != nil {
		return fmt.Errorf("failed to deploy: %w", err)
	}

	fmt.Printf("\nDeployment successful!\n")
	fmt.Printf("  ID: %s\n", deployment.ID)
	fmt.Printf("  Tenant: %s\n", deployment.TenantID)
	fmt.Printf("  Name: %s\n", deployment.Name)
	fmt.Printf("  Image: %s\n", deployment.ImageURL)
	if len(deployment.FunctionRequirements) > 0 {
		fmt.Printf("  Functions: %d\n", len(deployment.FunctionRequirements))
		for _, fn := range deployment.FunctionRequirements {
			gpuStr := "CPU"
			if fn.RequiresGPU {
				gpuStr = "GPU"
			}
			fmt.Printf("    - %s (%s)\n", fn.Name, gpuStr)
		}
	}

	return nil
}
