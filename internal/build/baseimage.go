package build

import (
	"fmt"
	"slices"
	"strings"
)

const (
	DefaultRegistry = "cozycreator/gen-worker"
	DefaultPython   = "3.11"
	DefaultCuda     = "12.6"
	DefaultTorchTag = "torch2.9"
)

var SupportedCudaVersions = []string{"13", "12.8", "12.6"}

// returns the appropriate base image for the config.
func ResolveBaseImage(cfg *ToolsCozyConfig) (string, error) {
	hasPytorch := cfg.Pytorch != ""
	hasCuda := cfg.Cuda != ""

	switch {
	case hasPytorch && hasCuda:
		// GPU: cozycreator/gen-worker:cuda12.6-torch2.9
		cuda := normalizeCuda(cfg.Cuda)
		if !isSupportedCuda(cuda) {
			return "", fmt.Errorf("unsupported CUDA version: %s (supported: %v)", cuda, SupportedCudaVersions)
		}
		return fmt.Sprintf("%s:cuda%s-%s", DefaultRegistry, cuda, DefaultTorchTag), nil

	case hasPytorch:
		// CPU PyTorch: cozycreator/gen-worker:cpu-torch2.9
		return fmt.Sprintf("%s:cpu-%s", DefaultRegistry, DefaultTorchTag), nil

	case hasCuda:
		// CUDA without pytorch - default to pytorch anyway
		cuda := normalizeCuda(cfg.Cuda)
		if !isSupportedCuda(cuda) {
			return "", fmt.Errorf("unsupported CUDA version: %s (supported: %v)", cuda, SupportedCudaVersions)
		}
		return fmt.Sprintf("%s:cuda%s-%s", DefaultRegistry, cuda, DefaultTorchTag), nil

	default:
		// Plain Python: python:3.11-slim
		py := normalizePython(cfg.Python)
		if py == "" {
			py = DefaultPython
		}
		return fmt.Sprintf("python:%s-slim", py), nil
	}
}

// ImageDescription returns a human-readable description.
func ImageDescription(cfg *ToolsCozyConfig) string {
	hasPytorch := cfg.Pytorch != ""
	hasCuda := cfg.Cuda != ""

	switch {
	case hasPytorch && hasCuda, hasCuda:
		cuda := normalizeCuda(cfg.Cuda)
		if cuda == "" {
			cuda = DefaultCuda
		}
		return fmt.Sprintf("PyTorch 2.9 + CUDA %s", cuda)

	case hasPytorch:
		return "PyTorch 2.9 (CPU)"

	default:
		py := cfg.Python
		if py == "" {
			py = DefaultPython
		}
		return fmt.Sprintf("Python %s (CPU)", py)
	}
}

func normalizePython(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "python")
	v = strings.TrimPrefix(v, "py")
	if parts := strings.Split(v, "."); len(parts) >= 2 {
		return parts[0] + "." + parts[1]
	}
	return v
}

func normalizeCuda(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "cuda")
	v = strings.TrimPrefix(v, "cu")
	if parts := strings.Split(v, "."); len(parts) >= 2 {
		if parts[1] == "0" {
			return parts[0]
		}
		return parts[0] + "." + parts[1]
	}
	return v
}

func isSupportedCuda(v string) bool {
	return slices.Contains(SupportedCudaVersions, v)
}
