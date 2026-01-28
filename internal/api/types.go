package api

import "time"

// FunctionRequirement describes a function provided by a deployment.
type FunctionRequirement struct {
	Name        string `json:"name"`
	RequiresGPU bool   `json:"requires_gpu"`
}

// CreateDeploymentRequest is the request body for creating a deployment.
type CreateDeploymentRequest struct {
	ID                   string              `json:"id"`
	Name                 string              `json:"name,omitempty"`
	ImageURL             string              `json:"image_url"`
	FunctionRequirements []FunctionRequirement `json:"function_requirements,omitempty"`
	SupportedModelIDs    []string            `json:"supported_model_ids,omitempty"`
	RunpodSecretMapping  map[string]string   `json:"runpod_secret_mapping,omitempty"`
	MinWorkers           *int                `json:"min_workers,omitempty"`
	MaxWorkers           *int                `json:"max_workers,omitempty"`
}

// UpdateDeploymentRequest is the request body for updating a deployment.
type UpdateDeploymentRequest struct {
	Name                 string              `json:"name,omitempty"`
	ImageURL             string              `json:"image_url,omitempty"`
	FunctionRequirements []FunctionRequirement `json:"function_requirements,omitempty"`
	SupportedModelIDs    []string            `json:"supported_model_ids,omitempty"`
	RunpodSecretMapping  map[string]string   `json:"runpod_secret_mapping,omitempty"`
	MinWorkers           *int                `json:"min_workers,omitempty"`
	MaxWorkers           *int                `json:"max_workers,omitempty"`
}

// DeploymentResponse is the response from deployment operations.
type DeploymentResponse struct {
	ID                   string              `json:"id"`
	TenantID             string              `json:"tenant_id"`
	Name                 string              `json:"name"`
	ImageURL             string              `json:"image_url"`
	FunctionRequirements []FunctionRequirement `json:"function_requirements,omitempty"`
	SupportedModelIDs    []string            `json:"supported_model_ids,omitempty"`
	RunpodSecretMapping  map[string]string   `json:"runpod_secret_mapping,omitempty"`
	MinWorkers           int                 `json:"min_workers"`
	MaxWorkers           int                 `json:"max_workers"`
	CreatedAt            time.Time           `json:"created_at"`
	UpdatedAt            time.Time           `json:"updated_at"`
}

// ListDeploymentsResponse is the response for listing deployments.
type ListDeploymentsResponse struct {
	Items []DeploymentResponse `json:"items"`
}

// ErrorResponse represents an API error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}
