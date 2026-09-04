package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/kariaranelly/brew-ops/backend/internal/domain"
	"github.com/kariaranelly/brew-ops/backend/internal/service"
)

// AuthHandler exposes /api/v1/auth/*. isDevelopment gates Register, which
// must 404 outside development — it exists only to seed a local dev account
// and has no place being reachable in a deployed environment.
type AuthHandler struct {
	auth          *service.AuthService
	isDevelopment bool
}

func NewAuthHandler(auth *service.AuthService, isDevelopment bool) *AuthHandler {
	return &AuthHandler{auth: auth, isDevelopment: isDevelopment}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type tokenPairResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req loginRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if req.Email == "" || req.Password == "" {
		return fiber.NewError(fiber.StatusBadRequest, "email and password are required")
	}

	pair, err := h.auth.Login(c.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid email or password")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "login failed")
	}

	return c.Status(fiber.StatusOK).JSON(tokenPairResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
	})
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type accessTokenResponse struct {
	AccessToken string `json:"access_token"`
}

func (h *AuthHandler) Refresh(c *fiber.Ctx) error {
	var req refreshRequest
	if err := c.BodyParser(&req); err != nil || req.RefreshToken == "" {
		return fiber.NewError(fiber.StatusBadRequest, "refresh_token is required")
	}

	access, err := h.auth.Refresh(c.Context(), req.RefreshToken)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid or expired refresh token")
	}

	return c.Status(fiber.StatusOK).JSON(accessTokenResponse{AccessToken: access})
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

func (h *AuthHandler) Register(c *fiber.Ctx) error {
	if !h.isDevelopment {
		return fiber.NewError(fiber.StatusNotFound)
	}

	var req registerRequest
	if err := c.BodyParser(&req); err != nil || req.Email == "" || req.Password == "" {
		return fiber.NewError(fiber.StatusBadRequest, "email and password are required")
	}

	user, err := h.auth.Register(c.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, domain.ErrUserAlreadyExists) {
			return fiber.NewError(fiber.StatusConflict, "user already exists")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "registration failed")
	}

	return c.Status(fiber.StatusCreated).JSON(userResponse{ID: user.ID.String(), Email: user.Email})
}
