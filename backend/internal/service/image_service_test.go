package service

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/kariaranelly/brew-ops/backend/internal/domain"
)

func TestValidateImageContentType_VariousTypes_ReturnsExpectedError(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		wantExt     string
		wantErr     error
	}{
		{"jpeg is allowed", "image/jpeg", ".jpg", nil},
		{"png is allowed", "image/png", ".png", nil},
		{"webp is allowed", "image/webp", ".webp", nil},
		{"pdf is rejected", "application/pdf", "", domain.ErrInvalidImageContentType},
		{"plain text is rejected", "text/plain", "", domain.ErrInvalidImageContentType},
		{"empty content type is rejected", "", "", domain.ErrInvalidImageContentType},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange (table row is the arrangement)

			// Act
			ext, err := validateImageContentType(tt.contentType)

			// Assert
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}
			if err == nil && ext != tt.wantExt {
				t.Fatalf("expected extension %q, got %q", tt.wantExt, ext)
			}
		})
	}
}

func TestNeedsServerSideOptimization_VariousSizes_ReturnsExpected(t *testing.T) {
	const fifteenMB = 15 * 1024 * 1024

	tests := []struct {
		name string
		size int
		want bool
	}{
		{"well under threshold", 1024, false},
		{"just under threshold", fifteenMB, false},
		{"just over threshold", fifteenMB + 1, true},
		{"way over threshold", fifteenMB * 3, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange (table row is the arrangement)

			// Act
			got := needsServerSideOptimization(tt.size)

			// Assert
			if got != tt.want {
				t.Fatalf("expected %v for size %d, got %v", tt.want, tt.size, got)
			}
		})
	}
}

// newTestPNG builds a tiny valid PNG in memory — no fixture files needed.
func newTestPNG(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 100, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("failed to build test PNG: %v", err)
	}
	return buf.Bytes()
}

// simulateOversizedUpload pads a small, valid PNG with trailing zero bytes
// so its total byte length exceeds 15MB without needing to actually encode
// a huge image — image/png's decoder stops at the IEND chunk and never
// looks at what comes after, so decoding is still fast and correct.
func simulateOversizedUpload(t *testing.T) []byte {
	t.Helper()
	small := newTestPNG(t, 50, 50)
	padding := make([]byte, 16*1024*1024)
	return append(small, padding...)
}

func TestImageServiceUpload_ValidSmallImage_UploadsAsIsWithoutOptimization(t *testing.T) {
	// Arrange
	pngData := newTestPNG(t, 100, 100)
	var gotFilename, gotContentType string
	var gotData []byte
	storageMock := &mockStorageClient{
		uploadFunc: func(_ context.Context, filename string, data []byte, contentType string) (string, error) {
			gotFilename, gotContentType, gotData = filename, contentType, data
			return "https://example.com/" + filename, nil
		},
	}
	svc := NewImageService(storageMock)

	// Act
	url, err := svc.Upload(context.Background(), pngData, "image/png")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if url == "" {
		t.Fatal("expected a non-empty URL")
	}
	if gotContentType != "image/png" {
		t.Fatalf("expected content type to remain image/png (no optimization needed), got %q", gotContentType)
	}
	if len(gotFilename) < 5 || gotFilename[len(gotFilename)-4:] != ".png" {
		t.Fatalf("expected a .png filename, got %q", gotFilename)
	}
	if !bytes.Equal(gotData, pngData) {
		t.Fatal("expected the original bytes to be uploaded unchanged when under the optimization threshold")
	}
}

func TestImageServiceUpload_InvalidContentType_RejectsWithoutCallingStorage(t *testing.T) {
	// Arrange
	storageMock := &mockStorageClient{
		uploadFunc: func(_ context.Context, _ string, _ []byte, _ string) (string, error) {
			t.Fatal("storage Upload should not be called for a non-image content type")
			return "", nil
		},
	}
	svc := NewImageService(storageMock)

	// Act
	_, err := svc.Upload(context.Background(), []byte("not an image"), "text/plain")

	// Assert
	if !errors.Is(err, domain.ErrInvalidImageContentType) {
		t.Fatalf("expected ErrInvalidImageContentType, got %v", err)
	}
}

func TestImageServiceUpload_OversizedFile_OptimizesBeforeUploading(t *testing.T) {
	// Arrange
	oversized := simulateOversizedUpload(t)
	if !needsServerSideOptimization(len(oversized)) {
		t.Fatalf("test fixture must exceed the optimization threshold, got %d bytes", len(oversized))
	}

	var gotFilename, gotContentType string
	var gotData []byte
	storageMock := &mockStorageClient{
		uploadFunc: func(_ context.Context, filename string, data []byte, contentType string) (string, error) {
			gotFilename, gotContentType, gotData = filename, contentType, data
			return "https://example.com/" + filename, nil
		},
	}
	svc := NewImageService(storageMock)

	// Act
	_, err := svc.Upload(context.Background(), oversized, "image/png")

	// Assert
	if err != nil {
		t.Fatalf("expected optimization+upload to succeed, got %v", err)
	}
	if gotContentType != "image/jpeg" {
		t.Fatalf("expected optimized output to always be re-encoded as JPEG, got content type %q", gotContentType)
	}
	if len(gotFilename) < 5 || gotFilename[len(gotFilename)-4:] != ".jpg" {
		t.Fatalf("expected a .jpg filename after optimization (even though the source was PNG), got %q", gotFilename)
	}
	if len(gotData) >= len(oversized) {
		t.Fatalf("expected optimized output (%d bytes) to be smaller than the padded input (%d bytes)", len(gotData), len(oversized))
	}
	if _, err := jpeg.Decode(bytes.NewReader(gotData)); err != nil {
		t.Fatalf("expected optimized output to be a valid JPEG, got decode error: %v", err)
	}
}

func TestResizeToMaxWidth_WiderThanMax_PreservesAspectRatio(t *testing.T) {
	// Arrange
	src := image.NewRGBA(image.Rect(0, 0, 4000, 2000))

	// Act
	resized := resizeToMaxWidth(src, 2000)

	// Assert
	bounds := resized.Bounds()
	if bounds.Dx() != 2000 {
		t.Fatalf("expected width 2000, got %d", bounds.Dx())
	}
	if bounds.Dy() != 1000 {
		t.Fatalf("expected height 1000 (aspect ratio preserved), got %d", bounds.Dy())
	}
}
