package build

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// DetectedFunction represents a detected worker function from Python source.
type DetectedFunction struct {
	Name        string
	RequiresGPU bool
}

// DetectWorkerFunctions scans Python files in a directory for @worker_function() decorated functions.
// It analyzes function signatures to determine GPU requirements based on model injection annotations.
func DetectWorkerFunctions(projectDir string) ([]DetectedFunction, error) {
	var functions []DetectedFunction

	// Find all Python files
	pythonFiles, err := findPythonFiles(projectDir)
	if err != nil {
		return nil, err
	}

	for _, pyFile := range pythonFiles {
		fileFunctions, err := parseWorkerFunctions(pyFile)
		if err != nil {
			// Skip files that can't be parsed
			continue
		}
		functions = append(functions, fileFunctions...)
	}

	return functions, nil
}

// findPythonFiles finds all .py files in a directory (excluding common non-source dirs).
func findPythonFiles(dir string) ([]string, error) {
	var files []string

	skipDirs := map[string]bool{
		".git":          true,
		".venv":         true,
		"venv":          true,
		"__pycache__":   true,
		".pytest_cache": true,
		"node_modules":  true,
		".tox":          true,
		"dist":          true,
		"build":         true,
		"*.egg-info":    true,
	}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip excluded directories
		if info.IsDir() {
			if skipDirs[info.Name()] || strings.HasSuffix(info.Name(), ".egg-info") {
				return filepath.SkipDir
			}
			return nil
		}

		// Only process .py files
		if strings.HasSuffix(info.Name(), ".py") {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// parseWorkerFunctions parses a Python file and extracts worker functions.
func parseWorkerFunctions(filePath string) ([]DetectedFunction, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	fileContent := string(content)
	var functions []DetectedFunction

	// Regular expression to find @worker_function() decorator followed by def
	// This handles multi-line function signatures
	decoratorPattern := regexp.MustCompile(`@worker_function\s*\([^)]*\)\s*\n\s*def\s+(\w+)\s*\(`)

	matches := decoratorPattern.FindAllStringSubmatchIndex(fileContent, -1)

	for _, match := range matches {
		if len(match) < 4 {
			continue
		}

		// Extract function name
		funcName := fileContent[match[2]:match[3]]

		// Find the end of the function signature (closing parenthesis before colon)
		sigStart := match[0]
		sigEnd := findSignatureEnd(fileContent, match[1])
		if sigEnd == -1 {
			sigEnd = min(match[1]+500, len(fileContent)) // Fallback
		}

		signature := fileContent[sigStart:sigEnd]

		// Analyze signature for GPU indicators
		requiresGPU := detectGPURequirementFromSignature(signature)

		functions = append(functions, DetectedFunction{
			Name:        funcName,
			RequiresGPU: requiresGPU,
		})
	}

	return functions, nil
}

// findSignatureEnd finds the position after the closing ) and : of a function signature.
func findSignatureEnd(content string, start int) int {
	depth := 0
	inString := false
	stringChar := byte(0)

	for i := start; i < len(content) && i < start+1000; i++ {
		c := content[i]

		// Handle string literals
		if !inString && (c == '"' || c == '\'') {
			// Check for triple quotes
			if i+2 < len(content) && content[i:i+3] == `"""` || content[i:i+3] == `'''` {
				inString = true
				stringChar = c
				i += 2
				continue
			}
			inString = true
			stringChar = c
			continue
		}

		if inString {
			if c == stringChar {
				// Check for triple quotes end
				if i+2 < len(content) && content[i:i+3] == string([]byte{stringChar, stringChar, stringChar}) {
					inString = false
					i += 2
					continue
				}
				// Check for escape
				if i > 0 && content[i-1] != '\\' {
					inString = false
				}
			}
			continue
		}

		// Track parentheses depth
		if c == '(' {
			depth++
		} else if c == ')' {
			depth--
			if depth < 0 {
				// Found the closing paren of the function signature
				// Look for the colon
				for j := i + 1; j < len(content) && j < i+50; j++ {
					if content[j] == ':' {
						return j + 1
					}
				}
				return i + 1
			}
		}
	}

	return -1
}

// detectGPURequirementFromSignature checks if function signature indicates GPU requirement.
func detectGPURequirementFromSignature(signature string) bool {
	lowerSig := strings.ToLower(signature)

	// GPU indicators - if any of these are present, the function likely needs GPU
	gpuIndicators := []string{
		"modelref",           // Model injection annotation
		"torch",              // PyTorch usage
		"cuda",               // CUDA usage
		"gpu",                // GPU keyword
		"autopipelinefor",    // Diffusers pipelines
		"stablediffusion",    // Stable Diffusion
		"pipeline",           // Generic pipeline
		"annotated[",         // Type annotation with potential model injection
	}

	for _, indicator := range gpuIndicators {
		if strings.Contains(lowerSig, indicator) {
			return true
		}
	}

	return false
}

// ParseFunctionsFromFlag parses a comma-separated function specification string.
// Format: "func1:true,func2:false" where the boolean indicates GPU requirement.
func ParseFunctionsFromFlag(spec string) ([]DetectedFunction, error) {
	if spec == "" {
		return nil, nil
	}

	var functions []DetectedFunction
	pairs := strings.Split(spec, ",")

	for _, pair := range pairs {
		parts := strings.SplitN(strings.TrimSpace(pair), ":", 2)
		if len(parts) == 0 || parts[0] == "" {
			continue
		}

		funcName := strings.TrimSpace(parts[0])
		requiresGPU := true // Default to GPU required

		if len(parts) == 2 {
			val := strings.ToLower(strings.TrimSpace(parts[1]))
			requiresGPU = val == "true" || val == "1" || val == "yes"
		}

		functions = append(functions, DetectedFunction{
			Name:        funcName,
			RequiresGPU: requiresGPU,
		})
	}

	return functions, nil
}
