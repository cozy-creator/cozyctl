package build

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetToolsCozyConfig(t *testing.T) {

	t.Run("Test toml parsing", func(t *testing.T) {
		cwd, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get working directory: %v", err)
		}

		// Go up two directories from internal/build to reach project root
		projectRoot := filepath.Join(cwd, "..", "..")
		testFile := filepath.Join(projectRoot, "test", "config", "sdxl-turbo-worker", "pyproject.toml")

		config, err := getToolsCozyConfig(testFile)
		if err != nil {
			t.Fatalf("failed to parse config: %v", err)
		}

		// Verify parsed values match pyproject.toml
		if config.DeploymentID != "sdxl-turbo-test" {
			t.Errorf("DeploymentID = %q, want %q", config.DeploymentID, "sdxl-turbo-test")
		}
		if config.Name != "worker" {
			t.Errorf("Name = %q, want %q", config.Name, "worker")
		}
		if config.Python != "3.11" {
			t.Errorf("Python = %q, want %q", config.Python, "3.11")
		}
		if config.Pytorch != "2.5" {
			t.Errorf("Pytorch = %q, want %q", config.Pytorch, "2.5")
		}
		if config.Cuda != "12.6" {
			t.Errorf("Cuda = %q, want %q", config.Cuda, "12.6")
		}
		if config.Predict != "from worker import generate; generate('a beautiful sunset over mountains')" {
			t.Errorf("Predict = %q, want %q", config.Predict, "from worker import generate; generate('a beautiful sunset over mountains')")
		}
	})

}
