package handler

import (
	"errors"
	"io"
	"net/http"

	"github.com/gofiber/fiber/v2"

	"github.com/kariaranelly/brew-ops/backend/internal/domain"
	"github.com/kariaranelly/brew-ops/backend/internal/service"
)

// UploadHandler exposes the generic POST /api/v1/uploads/image endpoint —
// shared by both the Product form (image_url) and MarketingAsset creation,
// so there's exactly one image-upload code path in the frontend, not two.
type UploadHandler struct {
	images *service.ImageService
}

func NewUploadHandler(images *service.ImageService) *UploadHandler {
	return &UploadHandler{images: images}
}

type uploadImageResponse struct {
	URL string `json:"url"`
}

// readImageUpload extracts the "image" multipart field and sniffs its real
// Content-Type from the bytes (via http.DetectContentType) rather than
// trusting the client-supplied header, so the "Content-Type is actually an
// image" validation holds against a spoofed header too.
func readImageUpload(c *fiber.Ctx) (data []byte, contentType string, err error) {
	fileHeader, err := c.FormFile("image")
	if err != nil {
		return nil, "", errors.New("missing \"image\" file field")
	}
	file, err := fileHeader.Open()
	if err != nil {
		return nil, "", errors.New("could not read uploaded file")
	}
	defer func() { _ = file.Close() }()

	data, err = io.ReadAll(file)
	if err != nil {
		return nil, "", errors.New("could not read uploaded file")
	}
	return data, http.DetectContentType(data), nil
}

func (h *UploadHandler) Upload(c *fiber.Ctx) error {
	data, contentType, err := readImageUpload(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	url, err := h.images.Upload(c.Context(), data, contentType)
	if err != nil {
		return mapImageError(err)
	}

	return c.Status(fiber.StatusCreated).JSON(uploadImageResponse{URL: url})
}

func mapImageError(err error) error {
	if errors.Is(err, domain.ErrInvalidImageContentType) {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return fiber.NewError(fiber.StatusInternalServerError, "upload failed")
}
