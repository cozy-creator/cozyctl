package deploy

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/cozy-creator/cozyctl/internal/config"
	"github.com/spf13/cobra"
)

var (
	deployDeployment string
	deployPush       bool
	deployDryRun     bool
)

func DeployCmd(getConfig func() *config.ProfileConfig) *cobra.Command {
	deployCmd := &cobra.Command{
		Use:   "deploy [path]",
		Short: "Deploy a project to Cozy",
		Long: `Deploy a Python project to the Cozy platform.

The project must have a pyproject.toml with [tool.cozy] configuration.

Example:
  cozy deploy .
  cozy deploy ./my-project
  cozy deploy --deployment my-model ./my-project`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDeploy(cmd, args, getConfig())
		},
	}

	deployCmd.Flags().StringVar(&deployDeployment, "deployment", "", "deployment name (defaults to project name)")
	deployCmd.Flags().BoolVar(&deployPush, "push", true, "push image to registry")
	deployCmd.Flags().BoolVar(&deployDryRun, "dry-run", false, "validate only, don't build")

	return deployCmd
}

func runDeploy(cmd *cobra.Command, args []string, profileCfg *config.ProfileConfig) error {
	if err := profileCfg.Config.Validate(); err != nil {
		return err
	}

	sourcePath := "."
	if len(args) > 0 {
		sourcePath = args[0]
	}

	// Resolve to absolute path
	absPath, err := filepath.Abs(sourcePath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Validate source directory
	if err := validateSource(absPath); err != nil {
		return err
	}

	// Create tarball
	fmt.Printf("Packaging %s...\n", absPath)
	archivePath, err := createTarball(absPath)
	if err != nil {
		return fmt.Errorf("failed to package source: %w", err)
	}
	defer os.Remove(archivePath)

	// Upload and trigger build
	fmt.Printf("Uploading to %s...\n", profileCfg.Config.BuilderURL)
	buildID, err := uploadAndTriggerBuild(archivePath, absPath)
	if err != nil {
		return err
	}

	fmt.Printf("Build started: %s\n", buildID)
	fmt.Println("Streaming logs...")
	fmt.Println(strings.Repeat("-", 60))

	// Stream logs until complete
	if err := streamBuildLogs(buildID); err != nil {
		return err
	}

	return nil
}

func validateSource(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("cannot access path: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", path)
	}

	pyproject := filepath.Join(path, "pyproject.toml")
	if _, err := os.Stat(pyproject); err != nil {
		return fmt.Errorf("pyproject.toml not found in %s", path)
	}

	return nil
}

func createTarball(sourceDir string) (string, error) {
	tmpFile, err := os.CreateTemp("", "cozy-deploy-*.tar.gz")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	gw := gzip.NewWriter(tmpFile)
	tw := tar.NewWriter(gw)
	defer func() {
		_ = tw.Close()
		_ = gw.Close()
	}()

	gitignore := loadGitignore(sourceDir)
	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if shouldSkip(rel, info, gitignore) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = filepath.ToSlash(rel)
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			if _, err := io.Copy(tw, f); err != nil {
				f.Close()
				return err
			}
			f.Close()
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return tmpFile.Name(), nil
}

func loadGitignore(sourceDir string) []string {
	path := filepath.Join(sourceDir, ".gitignore")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var patterns []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			patterns = append(patterns, line)
		}
	}
	return patterns
}

func shouldSkip(rel string, info os.FileInfo, gitignore []string) bool {
	name := info.Name()

	// Always skip these
	skip := []string{".git", "__pycache__", ".venv", "venv", ".mypy_cache", ".pytest_cache", "node_modules", ".DS_Store"}
	for _, s := range skip {
		if name == s {
			return true
		}
	}

	// Check gitignore patterns (simple matching)
	for _, pattern := range gitignore {
		if matchPattern(rel, pattern) {
			return true
		}
	}

	return false
}

func matchPattern(path, pattern string) bool {
	// Very simple pattern matching
	pattern = strings.TrimPrefix(pattern, "/")
	if strings.HasSuffix(pattern, "/") {
		return strings.HasPrefix(path+"/", pattern)
	}
	if strings.Contains(pattern, "*") {
		matched, _ := filepath.Match(pattern, filepath.Base(path))
		return matched
	}
	return path == pattern || strings.HasPrefix(path, pattern+"/")
}

func uploadAndTriggerBuild(archivePath, sourceDir string) (string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	_ = writer.WriteField("tenant_id", profileCfg.Config.TenantID)
	if deployDeployment != "" {
		_ = writer.WriteField("deployment", deployDeployment)
	} else {
		// Use directory name as deployment name
		_ = writer.WriteField("deployment", filepath.Base(sourceDir))
	}
	if deployPush {
		_ = writer.WriteField("push", "true")
	}
	if deployDryRun {
		_ = writer.WriteField("dry_run", "true")
	}

	file, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	part, err := writer.CreateFormFile("archive", filepath.Base(archivePath))
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(part, file); err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}

	url := strings.TrimRight(profileCfg.Config.BuilderURL, "/") + "/v1/builds"
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+profileCfg.Config.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to connect to builder: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("build request failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(msg)))
	}

	var result struct {
		BuildID string `json:"build_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return result.BuildID, nil
}

func streamBuildLogs(buildID string) error {
	url := fmt.Sprintf("%s/v1/builds/%s/logs?follow=true", strings.TrimRight(profileCfg.Config.BuilderURL, "/"), buildID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+profileCfg.Config.Token)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to log stream: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to get logs (%d): %s", resp.StatusCode, strings.TrimSpace(string(msg)))
	}

	// Check if SSE or plain text
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/event-stream") {
		return readSSELogs(resp.Body)
	}

	// Plain text logs
	_, err = io.Copy(os.Stdout, resp.Body)
	return err
}

func readSSELogs(reader io.Reader) error {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				fmt.Println(strings.Repeat("-", 60))
				fmt.Println("Build complete.")
				return nil
			}
			fmt.Println(data)
		} else if strings.HasPrefix(line, "event: ") {
			event := strings.TrimPrefix(line, "event: ")
			if event == "error" {
				return fmt.Errorf("build failed")
			}
		}
	}
	return scanner.Err()
}
