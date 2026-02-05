package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// BuilderClient is an HTTP client for the cozy-hub builder API.
type BuilderClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewBuilderClient creates a new cozy-hub builder API client.
func NewBuilderClient(baseURL, token string) *BuilderClient {
	return &BuilderClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Build represents a build in cozy-hub.
type Build struct {
	ID           string  `json:"id"`
	TenantID     string  `json:"tenant_id"`
	DeploymentID string  `json:"deployment_id,omitempty"`
	Status       string  `json:"status"`
	TarballPath  string  `json:"tarball_path,omitempty"`
	ImageTag     string  `json:"image_tag,omitempty"`
	ErrorMessage string  `json:"error_message,omitempty"`
	StartedAt    *string `json:"started_at,omitempty"`
	FinishedAt   *string `json:"finished_at,omitempty"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

// BuildLog represents a single log entry from a build.
type BuildLog struct {
	ID      int64  `json:"id"`
	BuildID string `json:"build_id"`
	TS      string `json:"ts"`
	Level   string `json:"level"`
	Phase   string `json:"phase"`
	Message string `json:"message"`
}

// BuildLogsResponse is the response from GET /api/v1/builds/:id/logs.
type BuildLogsResponse struct {
	Logs  []BuildLog `json:"logs"`
	Count int        `json:"count"`
}

// Deployment represents a deployment in cozy-hub.
type HubDeployment struct {
	ID              string  `json:"id"`
	TenantID        string  `json:"tenant_id"`
	Name            string  `json:"name,omitempty"`
	ActiveBuildID   *string `json:"active_build_id,omitempty"`
	PreviousBuildID *string `json:"previous_build_id,omitempty"`
	ImageURL        string  `json:"image_url,omitempty"`
	Backend         string  `json:"backend,omitempty"`
	DeploymentType  string  `json:"deployment_type,omitempty"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

// BuildUploadResponse is returned after creating a build.
type BuildUploadResponse struct {
	BuildID string `json:"build_id"`
	Status  string `json:"status"`
}

// BuildStatusResponse is the response from GET /api/v1/builds/:id.
type BuildStatusResponse struct {
	ID          string  `json:"id"`
	Status      string  `json:"status"`
	ImageTag    string  `json:"image_tag,omitempty"`
	LogsPath    string  `json:"logs_path,omitempty"`
	Error       string  `json:"error,omitempty"`
	CreatedAt   string  `json:"created_at"`
	StartedAt   *string `json:"started_at,omitempty"`
	CompletedAt *string `json:"completed_at,omitempty"`
}

// BuilderDeployResponse is the response from the deploy endpoint.
type BuilderDeployResponse struct {
	ID              string `json:"id"`
	TenantID        string `json:"tenant_id"`
	ActiveBuildID   string `json:"active_build_id"`
	PreviousBuildID string `json:"previous_build_id,omitempty"`
	ImageTag        string `json:"image_tag"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

// UploadTarball uploads a tarball to cozy-hub's file store.
// Returns the S3 path (tarball_path) to use when creating a build.
func (c *BuilderClient) UploadTarball(tarball *bytes.Buffer, buildName string) (string, error) {
	// Generate a unique path for the tarball
	tarballPath := fmt.Sprintf("builds/%s/%d.tar.gz", buildName, time.Now().UnixNano())

	url := fmt.Sprintf("%s/api/v1/file/%s", c.baseURL, tarballPath)
	httpReq, err := http.NewRequest("PUT", url, tarball)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/gzip")
	if c.token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.token)
	}

	// Use a longer timeout for uploads
	uploadClient := &http.Client{Timeout: 5 * time.Minute}
	resp, err := uploadClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("upload request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var errResp ErrorResponse
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error != "" {
			return "", fmt.Errorf("upload failed (%d): %s", resp.StatusCode, errResp.Error)
		}
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Message != "" {
			return "", fmt.Errorf("upload failed (%d): %s", resp.StatusCode, errResp.Message)
		}
		return "", fmt.Errorf("upload failed (%d): %s", resp.StatusCode, string(respBody))
	}

	return tarballPath, nil
}

// UploadBuild uploads a tarball and creates a build in cozy-hub.
func (c *BuilderClient) UploadBuild(tarball *bytes.Buffer, buildName string) (*BuildUploadResponse, error) {
	// Step 1: Upload tarball to file store
	tarballPath, err := c.UploadTarball(tarball, buildName)
	if err != nil {
		return nil, fmt.Errorf("failed to upload tarball: %w", err)
	}

	// Step 2: Create build with tarball path
	return c.CreateBuild(tarballPath)
}

// CreateBuild creates a new build in cozy-hub with an already-uploaded tarball.
func (c *BuilderClient) CreateBuild(tarballPath string) (*BuildUploadResponse, error) {
	reqBody := map[string]string{
		"tarball_path": tarballPath,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/builds", c.baseURL)
	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error != "" {
			return nil, fmt.Errorf("create build failed (%d): %s", resp.StatusCode, errResp.Error)
		}
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Message != "" {
			return nil, fmt.Errorf("create build failed (%d): %s", resp.StatusCode, errResp.Message)
		}
		return nil, fmt.Errorf("create build failed (%d): %s", resp.StatusCode, string(respBody))
	}

	// Parse cozy-hub Build response
	var build Build
	if err := json.Unmarshal(respBody, &build); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Map to legacy response format
	return &BuildUploadResponse{
		BuildID: build.ID,
		Status:  build.Status,
	}, nil
}

// GetBuildStatus fetches the current status of a build.
func (c *BuilderClient) GetBuildStatus(buildID string) (*BuildStatusResponse, error) {
	url := fmt.Sprintf("%s/api/v1/builds/%s", c.baseURL, buildID)
	httpReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if c.token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error != "" {
			return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, errResp.Error)
		}
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	// Parse cozy-hub Build response
	var build Build
	if err := json.Unmarshal(respBody, &build); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Map to legacy response format
	return &BuildStatusResponse{
		ID:          build.ID,
		Status:      build.Status,
		ImageTag:    build.ImageTag,
		Error:       build.ErrorMessage,
		CreatedAt:   build.CreatedAt,
		StartedAt:   build.StartedAt,
		CompletedAt: build.FinishedAt,
	}, nil
}

// GetBuildLogs fetches the logs for a build.
func (c *BuilderClient) GetBuildLogs(buildID string, afterID int64, limit int) (*BuildLogsResponse, error) {
	url := fmt.Sprintf("%s/api/v1/builds/%s/logs?after_id=%d&limit=%d", c.baseURL, buildID, afterID, limit)
	httpReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if c.token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error != "" {
			return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, errResp.Error)
		}
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	var logsResp BuildLogsResponse
	if err := json.Unmarshal(respBody, &logsResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &logsResp, nil
}

// DeployBuild calls POST /api/v1/builds/:id/deploy on cozy-hub.
func (c *BuilderClient) DeployBuild(buildID, tenantID string) (*BuilderDeployResponse, error) {
	url := fmt.Sprintf("%s/api/v1/builds/%s/deploy", c.baseURL, buildID)
	httpReq, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var errResp ErrorResponse
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Message != "" {
			return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, errResp.Message)
		}
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error != "" {
			return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, errResp.Error)
		}
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	// Try to parse as HubDeployment first
	var deployment HubDeployment
	if err := json.Unmarshal(respBody, &deployment); err == nil && deployment.ID != "" {
		activeBuildID := ""
		previousBuildID := ""
		if deployment.ActiveBuildID != nil {
			activeBuildID = *deployment.ActiveBuildID
		}
		if deployment.PreviousBuildID != nil {
			previousBuildID = *deployment.PreviousBuildID
		}
		return &BuilderDeployResponse{
			ID:              deployment.ID,
			TenantID:        deployment.TenantID,
			ActiveBuildID:   activeBuildID,
			PreviousBuildID: previousBuildID,
			ImageTag:        deployment.ImageURL,
			CreatedAt:       deployment.CreatedAt,
			UpdatedAt:       deployment.UpdatedAt,
		}, nil
	}

	// Fallback: try to parse as simple status response
	var simpleResp struct {
		Status  string `json:"status"`
		BuildID string `json:"build_id"`
	}
	if err := json.Unmarshal(respBody, &simpleResp); err == nil && simpleResp.Status == "deployed" {
		return &BuilderDeployResponse{
			ID:            simpleResp.BuildID,
			ActiveBuildID: simpleResp.BuildID,
		}, nil
	}

	return nil, fmt.Errorf("unexpected response format: %s", string(respBody))
}

// GetHubDeployment fetches a deployment by ID from cozy-hub.
func (c *BuilderClient) GetHubDeployment(deploymentID string) (*HubDeployment, error) {
	url := fmt.Sprintf("%s/api/v1/deployments/%s", c.baseURL, deploymentID)
	httpReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if c.token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error != "" {
			return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, errResp.Error)
		}
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	var deployment HubDeployment
	if err := json.Unmarshal(respBody, &deployment); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &deployment, nil
}
