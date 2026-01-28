package build

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFunctionsFromFlag(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []DetectedFunction
	}{
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:  "single function with GPU",
			input: "generate:true",
			expected: []DetectedFunction{
				{Name: "generate", RequiresGPU: true},
			},
		},
		{
			name:  "single function without GPU",
			input: "health:false",
			expected: []DetectedFunction{
				{Name: "health", RequiresGPU: false},
			},
		},
		{
			name:  "multiple functions",
			input: "generate:true,health:false,process:true",
			expected: []DetectedFunction{
				{Name: "generate", RequiresGPU: true},
				{Name: "health", RequiresGPU: false},
				{Name: "process", RequiresGPU: true},
			},
		},
		{
			name:  "function without GPU spec defaults to true",
			input: "generate",
			expected: []DetectedFunction{
				{Name: "generate", RequiresGPU: true},
			},
		},
		{
			name:  "mixed with and without GPU spec",
			input: "generate,health:false,process",
			expected: []DetectedFunction{
				{Name: "generate", RequiresGPU: true},
				{Name: "health", RequiresGPU: false},
				{Name: "process", RequiresGPU: true},
			},
		},
		{
			name:  "with whitespace",
			input: " generate : true , health : false ",
			expected: []DetectedFunction{
				{Name: "generate", RequiresGPU: true},
				{Name: "health", RequiresGPU: false},
			},
		},
		{
			name:  "yes/no values",
			input: "func1:yes,func2:no",
			expected: []DetectedFunction{
				{Name: "func1", RequiresGPU: true},
				{Name: "func2", RequiresGPU: false},
			},
		},
		{
			name:  "1/0 values",
			input: "func1:1,func2:0",
			expected: []DetectedFunction{
				{Name: "func1", RequiresGPU: true},
				{Name: "func2", RequiresGPU: false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseFunctionsFromFlag(tt.input)
			if err != nil {
				t.Fatalf("ParseFunctionsFromFlag(%q) returned error: %v", tt.input, err)
			}

			if len(result) != len(tt.expected) {
				t.Fatalf("ParseFunctionsFromFlag(%q) returned %d functions, want %d", tt.input, len(result), len(tt.expected))
			}

			for i, fn := range result {
				if fn.Name != tt.expected[i].Name {
					t.Errorf("Function[%d].Name = %q, want %q", i, fn.Name, tt.expected[i].Name)
				}
				if fn.RequiresGPU != tt.expected[i].RequiresGPU {
					t.Errorf("Function[%d].RequiresGPU = %v, want %v", i, fn.RequiresGPU, tt.expected[i].RequiresGPU)
				}
			}
		})
	}
}

func TestDetectGPURequirementFromSignature(t *testing.T) {
	tests := []struct {
		name      string
		signature string
		wantGPU   bool
	}{
		{
			name:      "empty signature",
			signature: "def func():",
			wantGPU:   false,
		},
		{
			name:      "ModelRef annotation",
			signature: "def generate(model: Annotated[Pipeline, ModelRef('sdxl')]):",
			wantGPU:   true,
		},
		{
			name:      "torch parameter",
			signature: "def process(tensor: torch.Tensor):",
			wantGPU:   true,
		},
		{
			name:      "cuda keyword",
			signature: "def train(device: str = 'cuda'):",
			wantGPU:   true,
		},
		{
			name:      "gpu keyword",
			signature: "def inference(use_gpu: bool = True):",
			wantGPU:   true,
		},
		{
			name:      "StableDiffusion pipeline",
			signature: "def generate(pipe: StableDiffusionPipeline):",
			wantGPU:   true,
		},
		{
			name:      "AutoPipelineFor",
			signature: "def generate(pipe: AutoPipelineForImage2Image):",
			wantGPU:   true,
		},
		{
			name:      "generic pipeline",
			signature: "def run(pipeline: DiffusionPipeline):",
			wantGPU:   true,
		},
		{
			name:      "Annotated type hint",
			signature: "def process(model: Annotated[Model, Inject]):",
			wantGPU:   true,
		},
		{
			name:      "simple string parameter",
			signature: "def health(status: str):",
			wantGPU:   false,
		},
		{
			name:      "int parameter",
			signature: "def count(n: int):",
			wantGPU:   false,
		},
		{
			name:      "case insensitive GPU",
			signature: "def func(GPU_ENABLED: bool):",
			wantGPU:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectGPURequirementFromSignature(tt.signature)
			if got != tt.wantGPU {
				t.Errorf("detectGPURequirementFromSignature(%q) = %v, want %v", tt.signature, got, tt.wantGPU)
			}
		})
	}
}

func TestDetectWorkerFunctions(t *testing.T) {
	// Create a temporary directory with test Python files
	tmpDir, err := os.MkdirTemp("", "cozyctl-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test case 1: Simple worker function
	simpleWorker := `
from cozy_runtime import worker_function

@worker_function()
def health():
    return {"status": "ok"}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "simple.py"), []byte(simpleWorker), 0644); err != nil {
		t.Fatalf("Failed to write simple.py: %v", err)
	}

	// Test case 2: GPU worker function with ModelRef
	gpuWorker := `
from typing import Annotated
from cozy_runtime import worker_function, ModelRef
from diffusers import StableDiffusionPipeline

@worker_function()
def generate(
    prompt: str,
    pipeline: Annotated[StableDiffusionPipeline, ModelRef("sdxl-turbo")]
) -> bytes:
    return pipeline(prompt).images[0]
`
	if err := os.WriteFile(filepath.Join(tmpDir, "gpu_worker.py"), []byte(gpuWorker), 0644); err != nil {
		t.Fatalf("Failed to write gpu_worker.py: %v", err)
	}

	// Test case 3: Multiple functions in one file
	multiWorker := `
from cozy_runtime import worker_function

@worker_function()
def func_a():
    pass

def not_a_worker():
    pass

@worker_function()
def func_b(x: int):
    return x * 2
`
	if err := os.WriteFile(filepath.Join(tmpDir, "multi.py"), []byte(multiWorker), 0644); err != nil {
		t.Fatalf("Failed to write multi.py: %v", err)
	}

	// Test case 4: File without worker functions
	noWorker := `
def regular_function():
    pass

class SomeClass:
    def method(self):
        pass
`
	if err := os.WriteFile(filepath.Join(tmpDir, "no_worker.py"), []byte(noWorker), 0644); err != nil {
		t.Fatalf("Failed to write no_worker.py: %v", err)
	}

	// Run detection
	functions, err := DetectWorkerFunctions(tmpDir)
	if err != nil {
		t.Fatalf("DetectWorkerFunctions failed: %v", err)
	}

	// Should find: health, generate, func_a, func_b (4 functions)
	if len(functions) != 4 {
		t.Errorf("Found %d functions, want 4", len(functions))
		for _, fn := range functions {
			t.Logf("  - %s (GPU: %v)", fn.Name, fn.RequiresGPU)
		}
	}

	// Create a map for easier lookup
	funcMap := make(map[string]DetectedFunction)
	for _, fn := range functions {
		funcMap[fn.Name] = fn
	}

	// Verify health function (no GPU)
	if fn, ok := funcMap["health"]; ok {
		if fn.RequiresGPU {
			t.Errorf("health function should not require GPU")
		}
	} else {
		t.Error("health function not found")
	}

	// Verify generate function (GPU required due to ModelRef)
	if fn, ok := funcMap["generate"]; ok {
		if !fn.RequiresGPU {
			t.Errorf("generate function should require GPU (has ModelRef)")
		}
	} else {
		t.Error("generate function not found")
	}

	// Verify func_a (no GPU)
	if fn, ok := funcMap["func_a"]; ok {
		if fn.RequiresGPU {
			t.Errorf("func_a should not require GPU")
		}
	} else {
		t.Error("func_a not found")
	}

	// Verify func_b (no GPU)
	if fn, ok := funcMap["func_b"]; ok {
		if fn.RequiresGPU {
			t.Errorf("func_b should not require GPU")
		}
	} else {
		t.Error("func_b not found")
	}
}

func TestDetectWorkerFunctions_SkipsExcludedDirs(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cozyctl-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a worker function in the main directory
	mainWorker := `
from cozy_runtime import worker_function

@worker_function()
def main_func():
    pass
`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.py"), []byte(mainWorker), 0644); err != nil {
		t.Fatalf("Failed to write main.py: %v", err)
	}

	// Create excluded directories with worker functions that should be skipped
	excludedDirs := []string{".git", ".venv", "venv", "__pycache__", "node_modules"}
	for _, dir := range excludedDirs {
		dirPath := filepath.Join(tmpDir, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			t.Fatalf("Failed to create %s: %v", dir, err)
		}

		excludedWorker := `
from cozy_runtime import worker_function

@worker_function()
def excluded_func():
    pass
`
		if err := os.WriteFile(filepath.Join(dirPath, "excluded.py"), []byte(excludedWorker), 0644); err != nil {
			t.Fatalf("Failed to write excluded.py in %s: %v", dir, err)
		}
	}

	functions, err := DetectWorkerFunctions(tmpDir)
	if err != nil {
		t.Fatalf("DetectWorkerFunctions failed: %v", err)
	}

	// Should only find main_func
	if len(functions) != 1 {
		t.Errorf("Found %d functions, want 1", len(functions))
		for _, fn := range functions {
			t.Logf("  - %s", fn.Name)
		}
	}

	if len(functions) > 0 && functions[0].Name != "main_func" {
		t.Errorf("Function name = %q, want 'main_func'", functions[0].Name)
	}
}

func TestDetectWorkerFunctions_EmptyDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cozyctl-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	functions, err := DetectWorkerFunctions(tmpDir)
	if err != nil {
		t.Fatalf("DetectWorkerFunctions failed: %v", err)
	}

	if len(functions) != 0 {
		t.Errorf("Found %d functions, want 0", len(functions))
	}
}

func TestDetectWorkerFunctions_MultilineSignature(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cozyctl-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Worker function with multi-line signature
	multilineWorker := `
from typing import Annotated
from cozy_runtime import worker_function, ModelRef

@worker_function()
def generate_image(
    prompt: str,
    negative_prompt: str = "",
    num_inference_steps: int = 50,
    guidance_scale: float = 7.5,
    pipeline: Annotated[
        StableDiffusionPipeline,
        ModelRef("sdxl-turbo")
    ] = None
) -> bytes:
    return pipeline(prompt).images[0]
`
	if err := os.WriteFile(filepath.Join(tmpDir, "multiline.py"), []byte(multilineWorker), 0644); err != nil {
		t.Fatalf("Failed to write multiline.py: %v", err)
	}

	functions, err := DetectWorkerFunctions(tmpDir)
	if err != nil {
		t.Fatalf("DetectWorkerFunctions failed: %v", err)
	}

	if len(functions) != 1 {
		t.Fatalf("Found %d functions, want 1", len(functions))
	}

	if functions[0].Name != "generate_image" {
		t.Errorf("Function name = %q, want 'generate_image'", functions[0].Name)
	}

	if !functions[0].RequiresGPU {
		t.Errorf("generate_image should require GPU (has ModelRef)")
	}
}

func TestFindPythonFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cozyctl-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create Python files
	pyFiles := []string{"main.py", "utils.py", "src/helper.py"}
	for _, f := range pyFiles {
		path := filepath.Join(tmpDir, f)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("Failed to create directory for %s: %v", f, err)
		}
		if err := os.WriteFile(path, []byte("# Python file"), 0644); err != nil {
			t.Fatalf("Failed to write %s: %v", f, err)
		}
	}

	// Create non-Python files (should be ignored)
	nonPyFiles := []string{"README.md", "config.yaml", "main.go"}
	for _, f := range nonPyFiles {
		path := filepath.Join(tmpDir, f)
		if err := os.WriteFile(path, []byte("# Not Python"), 0644); err != nil {
			t.Fatalf("Failed to write %s: %v", f, err)
		}
	}

	files, err := findPythonFiles(tmpDir)
	if err != nil {
		t.Fatalf("findPythonFiles failed: %v", err)
	}

	if len(files) != 3 {
		t.Errorf("Found %d Python files, want 3", len(files))
		for _, f := range files {
			t.Logf("  - %s", f)
		}
	}

	// Verify all files end with .py
	for _, f := range files {
		if filepath.Ext(f) != ".py" {
			t.Errorf("Non-Python file found: %s", f)
		}
	}
}
