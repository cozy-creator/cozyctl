package deploy

import (
	"fmt"

	"github.com/cozy-creator/cozyctl/internal/api"
	"github.com/cozy-creator/cozyctl/internal/config"
)

// Run executes the deploy process: send build-id to gen-builder for promotion.
func Run(buildID string) error {
	// Load config for tenant-id and builder URL
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

	// Get builder URL
	builderURL := profileCfg.Config.BuilderURL
	if builderURL == "" {
		builderURL = config.DefaultConfigData().BuilderURL
	}

	// Create builder API client
	client := api.NewBuilderClient(builderURL, profileCfg.Config.Token)

	// Deploy via gen-builder
	fmt.Println("\nDeploying via gen-builder...")
	deployment, err := client.DeployBuild(buildID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to deploy: %w", err)
	}

	fmt.Printf("\nDeployment successful!\n")
	fmt.Printf("  ID: %s\n", deployment.ID)
	fmt.Printf("  Tenant: %s\n", deployment.TenantID)
	fmt.Printf("  Active Build: %s\n", deployment.ActiveBuildID)
	fmt.Printf("  Image: %s\n", deployment.ImageTag)

	return nil
}
