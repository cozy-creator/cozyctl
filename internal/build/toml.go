package build

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type PyProjectToml struct {
	Tool struct {
		Cozy ToolsCozyConfig `toml:"cozy"`
	} `toml:"tool"`
}

type ToolsCozyConfig struct {
	DeploymentID string            `toml:"deployment-id"`
	Name         string            `toml:"name"`
	Python       string            `toml:"python"`
	Pytorch      string            `toml:"pytorch"`
	Cuda         string            `toml:"cuda"`
	Root         string            `toml:"root"`
	Predict      string            `toml:"predict"`
	Environment  map[string]string `toml:"environment"`
}

// [tool.cozy]
// deployment-id = "my-deployment"
// name = "worker"
// python = "3.11"
// pytorch = "2.5"           # Enables PyTorch base image
// cuda = "12.6"             # Enables CUDA support
// root = "src/app"          # Project root within tarball (optional)
// predict = "from worker import predict; predict()"
// ```
func getToolsCozyConfig(filepath string) (*ToolsCozyConfig, error) {
	var config PyProjectToml

	// Read the file
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("error reading the contents of the file %v", err)
	}

	if _, err := toml.Decode(string(data), &config); err != nil {
		return nil, fmt.Errorf("error decoding data from %s: %v", filepath, err)
	}

	return &config.Tool.Cozy, nil
}
