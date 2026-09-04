package service

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png" // registers the PNG decoder with image.Decode

	_ "golang.org/x/image/webp" // registers the WebP decoder with image.Decode

	"github.com/google/uuid"

	"github.com/kariaranelly/brew-ops/backend/internal/domain"
)

const (
	// maxImageWidthPixels is the width ceiling applied during server-side
	// optimization.
	maxImageWidthPixels = 2000
	// optimizedJPEGQuality is the recompression quality used whenever the
	// backend re-encodes an oversized image.
	optimizedJPEGQuality = 85
	// serverOptimizeThresholdBytes is the ">15MB" tier: above this, the
	// backend optimizes as defense-in-depth even though the frontend
	// should already have compressed anything over 5MB client-side.
	serverOptimizeThresholdBytes = 15 * 1024 * 1024
)

// allowedImageContentTypes maps every accepted Content-Type to the file
// extension used for its stored filename.
var allowedImageContentTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// ImageService is the single reusable upload path for every image in
// BrewOps — both Product.image_url and MarketingAsset go through it, so
// the validation/optimization logic never gets duplicated between them.
type ImageService struct {
	storage domain.StorageClient
}

func NewImageService(storage domain.StorageClient) *ImageService {
	return &ImageService{storage: storage}
}

// validateImageContentType is the only hard validation applied at any
// size: pure and exhaustively tested.
func validateImageContentType(contentType string) (extension string, err error) {
	ext, ok := allowedImageContentTypes[contentType]
	if !ok {
		return "", domain.ErrInvalidImageContentType
	}
	return ext, nil
}

// needsServerSideOptimization reports whether the backend must apply its
// own resize/recompress pass, i.e. data crossed the ">15MB" tier.
func needsServerSideOptimization(sizeBytes int) bool {
	return sizeBytes > serverOptimizeThresholdBytes
}

// optimizeImage decodes data (JPEG/PNG/WebP), downsamples it to at most
// maxImageWidthPixels wide (preserving aspect ratio) if wider than that,
// and always re-encodes the result as JPEG at ~85 quality, regardless of
// the original format.
func optimizeImage(data []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("image: decode for optimization: %w", err)
	}

	if img.Bounds().Dx() > maxImageWidthPixels {
		img = resizeToMaxWidth(img, maxImageWidthPixels)
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: optimizedJPEGQuality}); err != nil {
		return nil, fmt.Errorf("image: encode after optimization: %w", err)
	}
	return buf.Bytes(), nil
}

// resizeToMaxWidth downsamples img to maxWidth wide (preserving aspect
// ratio) using nearest-neighbor sampling — simple and dependency-free,
// appropriate for a defensive "shrink an oversized upload" pass rather than
// a quality-critical resize.
func resizeToMaxWidth(img image.Image, maxWidth int) image.Image {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	newHeight := height * maxWidth / width
	if newHeight < 1 {
		newHeight = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, maxWidth, newHeight))
	for y := 0; y < newHeight; y++ {
		srcY := bounds.Min.Y + y*height/newHeight
		for x := 0; x < maxWidth; x++ {
			srcX := bounds.Min.X + x*width/maxWidth
			dst.Set(x, y, img.At(srcX, srcY))
		}
	}
	return dst
}

// Upload validates the content type, optimizes server-side when data
// exceeds the 15MB threshold, and stores the result under a unique
// filename. It never rejects an upload for being large — only for not
// being one of the allowed image types.
func (s *ImageService) Upload(ctx context.Context, data []byte, contentType string) (string, error) {
	ext, err := validateImageContentType(contentType)
	if err != nil {
		return "", err
	}

	if needsServerSideOptimization(len(data)) {
		optimized, err := optimizeImage(data)
		if err != nil {
			return "", err
		}
		data = optimized
		contentType = "image/jpeg"
		ext = ".jpg"
	}

	filename := uuid.NewString() + ext
	url, err := s.storage.Upload(ctx, filename, data, contentType)
	if err != nil {
		return "", fmt.Errorf("image: upload: %w", err)
	}
	return url, nil
}
