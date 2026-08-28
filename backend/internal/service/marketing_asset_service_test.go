package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/kariaranelly/brew-ops/backend/internal/domain"
)

func TestValidateMarketingAssetType_VariousTypes_ReturnsExpectedError(t *testing.T) {
	tests := []struct {
		name      string
		assetType string
		wantErr   error
	}{
		{"product is valid", domain.MarketingAssetTypeProduct, nil},
		{"promotion is valid", domain.MarketingAssetTypePromotion, nil},
		{"unknown type is rejected", "BANNER", domain.ErrInvalidMarketingAssetType},
		{"empty type is rejected", "", domain.ErrInvalidMarketingAssetType},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange (table row is the arrangement)

			// Act
			err := validateMarketingAssetType(tt.assetType)

			// Assert
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestMarketingAssetCreate_ValidInput_UploadsImageAndCreatesRecord(t *testing.T) {
	// Arrange
	createdBy := uuid.New()
	uploadCalled := false
	storageMock := &mockStorageClient{
		uploadFunc: func(_ context.Context, filename string, _ []byte, _ string) (string, error) {
			uploadCalled = true
			return "https://example.com/" + filename, nil
		},
	}
	images := NewImageService(storageMock)

	var created domain.MarketingAsset
	assets := &mockMarketingAssetRepository{
		createFunc: func(_ context.Context, a *domain.MarketingAsset) error {
			a.ID = uuid.New()
			created = *a
			return nil
		},
	}
	svc := NewMarketingAssetService(assets, images)

	// Act
	asset, err := svc.Create(context.Background(), CreateMarketingAssetInput{
		Name:        "Promo de verano",
		Type:        domain.MarketingAssetTypePromotion,
		ImageData:   []byte{0xFF, 0xD8, 0xFF}, // not a full JPEG, content type is passed explicitly below
		ContentType: "image/jpeg",
		CreatedBy:   createdBy,
	})

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !uploadCalled {
		t.Fatal("expected the image to be uploaded via ImageService")
	}
	if asset.ImageURL == "" {
		t.Fatal("expected a non-empty ImageURL")
	}
	if created.Name != "Promo de verano" || created.Type != domain.MarketingAssetTypePromotion {
		t.Fatalf("unexpected created asset: %+v", created)
	}
}

func TestMarketingAssetCreate_InvalidType_RejectsWithoutUploadingOrPersisting(t *testing.T) {
	// Arrange
	storageMock := &mockStorageClient{
		uploadFunc: func(_ context.Context, _ string, _ []byte, _ string) (string, error) {
			t.Fatal("image should not be uploaded when the asset type is invalid")
			return "", nil
		},
	}
	images := NewImageService(storageMock)
	assets := &mockMarketingAssetRepository{
		createFunc: func(_ context.Context, _ *domain.MarketingAsset) error {
			t.Fatal("repository Create should not be called when the asset type is invalid")
			return nil
		},
	}
	svc := NewMarketingAssetService(assets, images)

	// Act
	_, err := svc.Create(context.Background(), CreateMarketingAssetInput{
		Name:        "Anuncio",
		Type:        "BANNER",
		ImageData:   []byte{0xFF, 0xD8, 0xFF},
		ContentType: "image/jpeg",
		CreatedBy:   uuid.New(),
	})

	// Assert
	if !errors.Is(err, domain.ErrInvalidMarketingAssetType) {
		t.Fatalf("expected ErrInvalidMarketingAssetType, got %v", err)
	}
}

func TestMarketingAssetSoftDelete_ExistingAsset_NeverCallsStorageDelete(t *testing.T) {
	// Arrange
	id, deletedBy := uuid.New(), uuid.New()
	deleteCalled := false
	storageMock := &mockStorageClient{
		uploadFunc: func(_ context.Context, filename string, _ []byte, _ string) (string, error) {
			return "https://example.com/" + filename, nil
		},
		deleteFunc: func(_ context.Context, _ string) error {
			deleteCalled = true
			return nil
		},
	}
	images := NewImageService(storageMock)
	softDeleteCalled := false
	assets := &mockMarketingAssetRepository{
		softDeleteFunc: func(_ context.Context, gotID, gotDeletedBy uuid.UUID) error {
			softDeleteCalled = true
			if gotID != id || gotDeletedBy != deletedBy {
				t.Fatalf("unexpected args: id=%v deletedBy=%v", gotID, gotDeletedBy)
			}
			return nil
		},
	}
	svc := NewMarketingAssetService(assets, images)

	// Act
	err := svc.SoftDelete(context.Background(), id, deletedBy)

	// Assert — soft-deleting a marketing asset must never touch the
	// underlying file, so Restore can bring the record back without
	// re-uploading the image.
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !softDeleteCalled {
		t.Fatal("expected repository SoftDelete to be called")
	}
	if deleteCalled {
		t.Fatal("expected storage Delete to NEVER be called on soft-delete")
	}
}

func TestMarketingAssetRestore_ExistingAsset_ReturnsRecordWithoutReuploading(t *testing.T) {
	// Arrange
	id := uuid.New()
	uploadCalled := false
	storageMock := &mockStorageClient{
		uploadFunc: func(_ context.Context, filename string, _ []byte, _ string) (string, error) {
			uploadCalled = true
			return "https://example.com/" + filename, nil
		},
	}
	images := NewImageService(storageMock)
	restored := domain.MarketingAsset{ID: id, ImageURL: "https://example.com/existing.jpg"}
	assets := &mockMarketingAssetRepository{
		restoreFunc: func(_ context.Context, gotID uuid.UUID) (*domain.MarketingAsset, error) {
			if gotID != id {
				t.Fatalf("expected restore for %v, got %v", id, gotID)
			}
			return &restored, nil
		},
	}
	svc := NewMarketingAssetService(assets, images)

	// Act
	result, err := svc.Restore(context.Background(), id)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.ImageURL != "https://example.com/existing.jpg" {
		t.Fatalf("expected the original ImageURL to be preserved, got %q", result.ImageURL)
	}
	if uploadCalled {
		t.Fatal("expected restore to never re-upload the image")
	}
}
