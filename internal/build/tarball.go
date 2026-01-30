package build

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// excludedDirs are directories to skip when creating the tarball.
var excludedDirs = map[string]bool{
	".git":         true,
	"__pycache__":  true,
	"node_modules": true,
	".venv":        true,
	"venv":         true,
	".tox":         true,
	".mypy_cache":  true,
	".pytest_cache": true,
	".ruff_cache":  true,
}

// excludedFiles are files to skip when creating the tarball.
var excludedFiles = map[string]bool{
	".env":        true,
	".DS_Store":   true,
	"Dockerfile":  true,
	"Thumbs.db":   true,
}

// CreateTarball creates a gzip-compressed tar archive from a project directory.
// It excludes common non-essential directories and files.
func CreateTarball(projectDir string) (*bytes.Buffer, error) {
	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve project path: %w", err)
	}

	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	err = filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip symlinks
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		name := info.Name()

		// Skip excluded directories
		if info.IsDir() && excludedDirs[name] {
			return filepath.SkipDir
		}

		// Skip hidden directories (except the root)
		if info.IsDir() && strings.HasPrefix(name, ".") && path != absDir {
			return filepath.SkipDir
		}

		// Skip excluded files
		if !info.IsDir() && excludedFiles[name] {
			return nil
		}

		// Skip .pyc files
		if !info.IsDir() && strings.HasSuffix(name, ".pyc") {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(absDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// Skip the root directory itself
		if relPath == "." {
			return nil
		}

		// Security: no path traversal
		if strings.HasPrefix(relPath, "..") {
			return fmt.Errorf("path traversal detected: %s", relPath)
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("failed to create tar header for %s: %w", relPath, err)
		}
		header.Name = relPath

		if err := tw.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write tar header for %s: %w", relPath, err)
		}

		// Write file content
		if !info.IsDir() {
			f, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("failed to open %s: %w", relPath, err)
			}
			defer f.Close()

			if _, err := io.Copy(tw, f); err != nil {
				return fmt.Errorf("failed to write %s to tarball: %w", relPath, err)
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create tarball: %w", err)
	}

	if err := tw.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize tar: %w", err)
	}
	if err := gzw.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize gzip: %w", err)
	}

	return &buf, nil
}
