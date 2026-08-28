package storage

import (
	"context"
	"errors"
	"fmt"

	"cloud.google.com/go/storage"
)

// GCSStorageClient implements domain.StorageClient against a real GCP
// Cloud Storage bucket. Authentication is handled entirely by the SDK's
// Application Default Credentials resolution: locally it reads the file at
// GOOGLE_APPLICATION_CREDENTIALS, and on Cloud Run it automatically uses
// the service account attached to the service — no branching code needed
// for that here.
//
// Upload assumes bucket is configured for public object read access
// (uniform bucket-level access + an allUsers objectViewer binding, set up
// in Terraform) so the returned storage.googleapis.com URL is directly
// fetchable, matching how LocalDiskStorageClient's URLs behave.
type GCSStorageClient struct {
	client *storage.Client
	bucket string
}

// NewGCSStorageClient dials GCP Cloud Storage. ctx is only used for the
// client's own setup, not for later per-call timeouts.
func NewGCSStorageClient(ctx context.Context, bucket string) (*GCSStorageClient, error) {
	if bucket == "" {
		return nil, errors.New("gcs storage: GCP_STORAGE_BUCKET is required")
	}
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("gcs storage: create client: %w", err)
	}
	return &GCSStorageClient{client: client, bucket: bucket}, nil
}

func (c *GCSStorageClient) Upload(ctx context.Context, filename string, data []byte, contentType string) (string, error) {
	obj := c.client.Bucket(c.bucket).Object(filename)
	w := obj.NewWriter(ctx)
	w.ContentType = contentType

	if _, err := w.Write(data); err != nil {
		_ = w.Close()
		return "", fmt.Errorf("gcs storage: write %q: %w", filename, err)
	}
	if err := w.Close(); err != nil {
		return "", fmt.Errorf("gcs storage: finalize upload of %q: %w", filename, err)
	}

	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", c.bucket, filename), nil
}

func (c *GCSStorageClient) Delete(ctx context.Context, filename string) error {
	if err := c.client.Bucket(c.bucket).Object(filename).Delete(ctx); err != nil {
		return fmt.Errorf("gcs storage: delete %q: %w", filename, err)
	}
	return nil
}
