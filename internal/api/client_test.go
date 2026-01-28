package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	client := NewClient("http://localhost:8090", "test-token")

	if client.baseURL != "http://localhost:8090" {
		t.Errorf("baseURL = %q, want %q", client.baseURL, "http://localhost:8090")
	}
	if client.token != "test-token" {
		t.Errorf("token = %q, want %q", client.token, "test-token")
	}
}

func TestNewClient_TrimsTrailingSlash(t *testing.T) {
	client := NewClient("http://localhost:8090/", "test-token")

	if client.baseURL != "http://localhost:8090" {
		t.Errorf("baseURL = %q, want %q (trailing slash should be trimmed)", client.baseURL, "http://localhost:8090")
	}
}

func TestCreateDeployment_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("Method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/v1/deployments" {
			t.Errorf("Path = %q, want /v1/deployments", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Authorization = %q, want 'Bearer test-token'", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q, want 'application/json'", r.Header.Get("Content-Type"))
		}

		// Verify request body
		var req CreateDeploymentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}
		if req.ID != "test-deployment" {
			t.Errorf("ID = %q, want 'test-deployment'", req.ID)
		}

		// Send response
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(DeploymentResponse{
			ID:        "test-deployment",
			TenantID:  "tenant-123",
			Name:      "test-deployment",
			ImageURL:  "registry.example.com/test:v1",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	resp, err := client.CreateDeployment(&CreateDeploymentRequest{
		ID:       "test-deployment",
		Name:     "test-deployment",
		ImageURL: "registry.example.com/test:v1",
	})

	if err != nil {
		t.Fatalf("CreateDeployment failed: %v", err)
	}
	if resp.ID != "test-deployment" {
		t.Errorf("ID = %q, want 'test-deployment'", resp.ID)
	}
	if resp.TenantID != "tenant-123" {
		t.Errorf("TenantID = %q, want 'tenant-123'", resp.TenantID)
	}
}

func TestCreateDeployment_Conflict(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "conflict",
			Message: "deployment already exists",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	_, err := client.CreateDeployment(&CreateDeploymentRequest{
		ID: "existing-deployment",
	})

	if err == nil {
		t.Fatal("Expected error for conflict, got nil")
	}
	if err.Error() != "deployment 'existing-deployment' already exists (use 'cozyctl update' to update)" {
		t.Errorf("Error = %q, want conflict message", err.Error())
	}
}

func TestCreateDeployment_WithFunctions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req CreateDeploymentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		if len(req.FunctionRequirements) != 2 {
			t.Errorf("FunctionRequirements length = %d, want 2", len(req.FunctionRequirements))
		}

		// Verify functions
		funcMap := make(map[string]bool)
		for _, fn := range req.FunctionRequirements {
			funcMap[fn.Name] = fn.RequiresGPU
		}
		if gpu, ok := funcMap["generate"]; !ok || !gpu {
			t.Errorf("generate function should require GPU")
		}
		if gpu, ok := funcMap["health"]; !ok || gpu {
			t.Errorf("health function should not require GPU")
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(DeploymentResponse{
			ID:                   req.ID,
			FunctionRequirements: req.FunctionRequirements,
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	resp, err := client.CreateDeployment(&CreateDeploymentRequest{
		ID:       "ml-deployment",
		ImageURL: "registry.example.com/ml:v1",
		FunctionRequirements: []FunctionRequirement{
			{Name: "generate", RequiresGPU: true},
			{Name: "health", RequiresGPU: false},
		},
	})

	if err != nil {
		t.Fatalf("CreateDeployment failed: %v", err)
	}
	if len(resp.FunctionRequirements) != 2 {
		t.Errorf("FunctionRequirements length = %d, want 2", len(resp.FunctionRequirements))
	}
}

func TestUpdateDeployment_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("Method = %q, want PUT", r.Method)
		}
		if r.URL.Path != "/v1/deployments/test-deployment" {
			t.Errorf("Path = %q, want /v1/deployments/test-deployment", r.URL.Path)
		}

		var req UpdateDeploymentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}
		if req.ImageURL != "registry.example.com/test:v2" {
			t.Errorf("ImageURL = %q, want 'registry.example.com/test:v2'", req.ImageURL)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(DeploymentResponse{
			ID:       "test-deployment",
			ImageURL: req.ImageURL,
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	resp, err := client.UpdateDeployment("test-deployment", &UpdateDeploymentRequest{
		ImageURL: "registry.example.com/test:v2",
	})

	if err != nil {
		t.Fatalf("UpdateDeployment failed: %v", err)
	}
	if resp.ImageURL != "registry.example.com/test:v2" {
		t.Errorf("ImageURL = %q, want 'registry.example.com/test:v2'", resp.ImageURL)
	}
}

func TestUpdateDeployment_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	_, err := client.UpdateDeployment("nonexistent", &UpdateDeploymentRequest{
		ImageURL: "registry.example.com/test:v2",
	})

	if err == nil {
		t.Fatal("Expected error for not found, got nil")
	}
	if err.Error() != "deployment 'nonexistent' not found (use 'cozyctl deploy' to create)" {
		t.Errorf("Error = %q, want not found message", err.Error())
	}
}

func TestGetDeployment_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/v1/deployments/test-deployment" {
			t.Errorf("Path = %q, want /v1/deployments/test-deployment", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(DeploymentResponse{
			ID:       "test-deployment",
			TenantID: "tenant-123",
			ImageURL: "registry.example.com/test:v1",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	resp, err := client.GetDeployment("test-deployment")

	if err != nil {
		t.Fatalf("GetDeployment failed: %v", err)
	}
	if resp == nil {
		t.Fatal("Response should not be nil")
	}
	if resp.ID != "test-deployment" {
		t.Errorf("ID = %q, want 'test-deployment'", resp.ID)
	}
}

func TestGetDeployment_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	resp, err := client.GetDeployment("nonexistent")

	if err != nil {
		t.Fatalf("GetDeployment should not return error for not found: %v", err)
	}
	if resp != nil {
		t.Errorf("Response should be nil for not found deployment")
	}
}

func TestListDeployments_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/v1/deployments" {
			t.Errorf("Path = %q, want /v1/deployments", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ListDeploymentsResponse{
			Items: []DeploymentResponse{
				{ID: "deployment-1", TenantID: "tenant-123"},
				{ID: "deployment-2", TenantID: "tenant-123"},
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	deployments, err := client.ListDeployments()

	if err != nil {
		t.Fatalf("ListDeployments failed: %v", err)
	}
	if len(deployments) != 2 {
		t.Errorf("Deployments length = %d, want 2", len(deployments))
	}
}

func TestListDeployments_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ListDeploymentsResponse{
			Items: []DeploymentResponse{},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	deployments, err := client.ListDeployments()

	if err != nil {
		t.Fatalf("ListDeployments failed: %v", err)
	}
	if len(deployments) != 0 {
		t.Errorf("Deployments length = %d, want 0", len(deployments))
	}
}

func TestDeleteDeployment_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("Method = %q, want DELETE", r.Method)
		}
		if r.URL.Path != "/v1/deployments/test-deployment" {
			t.Errorf("Path = %q, want /v1/deployments/test-deployment", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	err := client.DeleteDeployment("test-deployment")

	if err != nil {
		t.Fatalf("DeleteDeployment failed: %v", err)
	}
}

func TestDeleteDeployment_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	err := client.DeleteDeployment("nonexistent")

	if err == nil {
		t.Fatal("Expected error for not found, got nil")
	}
	if err.Error() != "deployment 'nonexistent' not found" {
		t.Errorf("Error = %q, want not found message", err.Error())
	}
}

func TestAPIError_WithMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:   "bad_request",
			Message: "invalid image URL format",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	_, err := client.CreateDeployment(&CreateDeploymentRequest{
		ID:       "test",
		ImageURL: "invalid",
	})

	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if err.Error() != "API error (400): invalid image URL format" {
		t.Errorf("Error = %q, want API error message", err.Error())
	}
}
