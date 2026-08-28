package domain

import (
	"context"
	"errors"
)

// ErrInvalidImageContentType is returned when an upload's Content-Type is
// not one of the allowed image types — the only hard validation in
// CLAUDE.md's image upload scheme; size never causes a rejection.
var ErrInvalidImageContentType = errors.New("file must be a JPEG, PNG, or WebP image")

// StorageClient is the persistence boundary for uploaded binary assets
// (product and marketing images). Two implementations exist —
// storage.LocalDiskStorageClient (default, no GCP dependency needed) and
// storage.GCSStorageClient — selected in main.go via the STORAGE_BACKEND
// environment variable. Nothing above this interface (service, handler)
// needs to know which one is active.
type StorageClient interface {
	Upload(ctx context.Context, filename string, data []byte, contentType string) (url string, err error)
	Delete(ctx context.Context, filename string) error
}
