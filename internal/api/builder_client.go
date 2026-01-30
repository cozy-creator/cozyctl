package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

// BuilderClient is an HTTP client for the gen-builder API.
type BuilderClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewBuilderClient creates a new gen-builder API client.
func NewBuilderClient(baseURL, token string) *BuilderClient {
	return &BuilderClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// DeployBuildRequest is the request body for POST /v1/builds/{id}/deploy.
type DeployBuildRequest struct {
	TenantID string `json:"tenant_id"`
}

// BuilderDeployResponse is the response from the gen-builder deploy endpoint.
type BuilderDeployResponse struct {
	ID              string `json:"id"`
	TenantID        string `json:"tenant_id"`
	ActiveBuildID   string `json:"active_build_id"`
	PreviousBuildID string `json:"previous_build_id,omitempty"`
	ImageTag        string `json:"image_tag"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

// BuildUploadResponse is the response from POST /v1/build.
type BuildUploadResponse struct {
	BuildID string `json:"build_id"`
	Status  string `json:"status"`
}

// BuildStatusResponse is the response from GET /v1/build/{id}.
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

// UploadBuild uploads a tarball to gen-builder via POST /v1/build.
func (c *BuilderClient) UploadBuild(tarball *bytes.Buffer, buildName string) (*BuildUploadResponse, error) {
	// Build multipart form
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Add tarball file
	part, err := writer.CreateFormFile("file", "project.tar.gz")
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := io.Copy(part, tarball); err != nil {
		return nil, fmt.Errorf("failed to write tarball to form: %w", err)
	}

	// Add config field
	config := fmt.Sprintf(`{"build_name":%q}`, buildName)
	if err := writer.WriteField("config", config); err != nil {
		return nil, fmt.Errorf("failed to write config field: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	url := fmt.Sprintf("%s/v1/build", c.baseURL)
	httpReq, err := http.NewRequest("POST", url, &body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	if c.token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.token)
	}

	// Use a longer timeout for uploads
	uploadClient := &http.Client{Timeout: 5 * time.Minute}
	resp, err := uploadClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("upload request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		var errResp ErrorResponse
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error != "" {
			return nil, fmt.Errorf("upload failed (%d): %s", resp.StatusCode, errResp.Error)
		}
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Message != "" {
			return nil, fmt.Errorf("upload failed (%d): %s", resp.StatusCode, errResp.Message)
		}
		return nil, fmt.Errorf("upload failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var buildResp BuildUploadResponse
	if err := json.Unmarshal(respBody, &buildResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &buildResp, nil
}

// GetBuildStatus fetches the current status of a build via GET /v1/build/{id}.
func (c *BuilderClient) GetBuildStatus(buildID string) (*BuildStatusResponse, error) {
	url := fmt.Sprintf("%s/v1/build/%s", c.baseURL, buildID)
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

	var status BuildStatusResponse
	if err := json.Unmarshal(respBody, &status); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &status, nil
}

// DeployBuild calls POST /v1/builds/{id}/deploy on gen-builder.
func (c *BuilderClient) DeployBuild(buildID, tenantID string) (*BuilderDeployResponse, error) {
	req := DeployBuildRequest{TenantID: tenantID}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/v1/builds/%s/deploy", c.baseURL, buildID)
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

	var deployment BuilderDeployResponse
	if err := json.Unmarshal(respBody, &deployment); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &deployment, nil
}
