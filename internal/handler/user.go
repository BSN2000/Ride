package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"ride/internal/domain"
	"ride/internal/repository"
)

// UserHandler handles HTTP requests for users.
type UserHandler struct {
	userRepo repository.UserRepository
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(userRepo repository.UserRepository) *UserHandler {
	return &UserHandler{userRepo: userRepo}
}

// RegisterRequest is the HTTP request body for user registration.
type RegisterRequest struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

// UserResponse is the HTTP response for user data.
type UserResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

// Register handles POST /v1/users/register
func (h *UserHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	if req.Name == "" || req.Phone == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "name and phone are required"})
		return
	}

	// Check if user already exists
	existing, err := h.userRepo.GetByPhone(c.Request.Context(), req.Phone)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		respondError(c, err)
		return
	}

	if existing != nil {
		c.JSON(http.StatusConflict, gin.H{
			"message": "User already registered",
			"user":    UserResponse{ID: existing.ID, Name: existing.Name, Phone: existing.Phone},
		})
		return
	}

	// Create new user
	user := &domain.User{
		ID:    uuid.New().String(),
		Name:  req.Name,
		Phone: req.Phone,
	}

	if err := h.userRepo.Create(c.Request.Context(), user); err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, UserResponse{
		ID:    user.ID,
		Name:  user.Name,
		Phone: user.Phone,
	})
}

// GetAll handles GET /v1/users
func (h *UserHandler) GetAll(c *gin.Context) {
	users, err := h.userRepo.GetAll(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}

	var response []UserResponse
	for _, u := range users {
		response = append(response, UserResponse{
			ID:    u.ID,
			Name:  u.Name,
			Phone: u.Phone,
		})
	}

	c.JSON(http.StatusOK, response)
}
