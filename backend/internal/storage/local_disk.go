// Package storage contains the concrete implementations of
// domain.StorageClient: LocalDiskStorageClient (default, for local
// development with no GCP dependency) and GCSStorageClient (GCP Cloud
// Storage, used in production). Selecting between them is a config-time
// decision made in cmd/server/main.go — nothing else in the codebase
// depends on this package directly, only on domain.StorageClient.
package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// LocalDiskStorageClient implements domain.StorageClient by writing files
// to a directory on local disk, and serving them back via a Fiber static
// route (see cmd/server/main.go's app.Static registration) so the URLs it
// returns are fetchable exactly like a real GCS public URL would be. This
// is the default backend — it exists so image upload work isn't blocked on
// having an active GCP billing account (see CLAUDE.md's Session 4 notes).
type LocalDiskStorageClient struct {
	dir       string
	publicURL string
}

// NewLocalDiskStorageClient creates dir if it doesn't exist and returns a
// client that builds URLs as "<publicBaseURL>/local-storage/<filename>".
func NewLocalDiskStorageClient(dir, publicBaseURL string) (*LocalDiskStorageClient, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("local storage: create directory %q: %w", dir, err)
	}
	return &LocalDiskStorageClient{dir: dir, publicURL: publicBaseURL}, nil
}

func (c *LocalDiskStorageClient) Upload(_ context.Context, filename string, data []byte, _ string) (string, error) {
	path := filepath.Join(c.dir, filename)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("local storage: write %q: %w", filename, err)
	}
	return fmt.Sprintf("%s/local-storage/%s", c.publicURL, filename), nil
}

func (c *LocalDiskStorageClient) Delete(_ context.Context, filename string) error {
	path := filepath.Join(c.dir, filename)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("local storage: delete %q: %w", filename, err)
	}
	return nil
}
