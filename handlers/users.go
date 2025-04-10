package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Bethel-nz/tickit/app/router"
	"github.com/Bethel-nz/tickit/internal/auth"
	"github.com/Bethel-nz/tickit/internal/database/store"
	"github.com/Bethel-nz/tickit/internal/services"
	"github.com/Bethel-nz/tickit/internal/validator"
	"github.com/jackc/pgx/v5/pgtype"
)

// userService is retrieved from the application's dependency container
var userService *services.UserService

// SetUserService sets the user service for handlers
func SetUserService(service *services.UserService) {
	userService = service
}

// RegisterRequest represents user registration input
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name,omitempty"`
	Username string `json:"username,omitempty"`
}

// LoginRequest represents login input
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// ForgotPasswordRequest represents a password reset request
type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

// ResetPasswordRequest represents a password reset with token
type ResetPasswordRequest struct {
	NewPassword string `json:"new_password"`
}

// RegisterUser handles user registration
func RegisterUser(c *router.Context) {
	if userService == nil {
		c.Status(http.StatusInternalServerError, "User service not initialized")
		return
	}
	var req RegisterRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		c.Status(http.StatusBadRequest, "Invalid request format")
		return
	}

	// Validate input
	if err := validateRegisterRequest(req); err != nil {
		c.Status(http.StatusBadRequest, err.Error())
		return
	}

	// Create user params
	params := store.CreateUserParams{
		Email:    req.Email,
		Password: req.Password,
		Name:     pgtype.Text{String: req.Name, Valid: req.Name != ""},
		Username: pgtype.Text{String: req.Username, Valid: req.Username != ""},
	}

	// Call service
	user, err := userService.CreateUser(c.Request.Context(), params)
	if err != nil {
		if errors.Is(err, services.ErrDuplicateEmail) {
			c.Status(http.StatusConflict, "Email already registered")
			return
		}
		c.Status(http.StatusInternalServerError, "Failed to create user")
		return
	}

	// Return success with user info
	c.JSON(http.StatusCreated, map[string]interface{}{
		"id":       user.ID.String(),
		"email":    user.Email,
		"name":     user.Name.String,
		"username": user.Username.String,
		"message":  "User registered successfully",
	})
}

// LoginUser handles user login
func LoginUser(c *router.Context) {
	if userService == nil {
		c.Status(http.StatusInternalServerError, "User service not initialized")
		return
	}
	var req LoginRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		c.Status(http.StatusBadRequest, "Invalid request format")
		return
	}

	// Validate credentials
	if req.Email == "" || req.Password == "" {
		c.Status(http.StatusBadRequest, "Email and password are required")
		return
	}

	// Authenticate user
	user, err := userService.AuthenticateUser(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, services.ErrInvalidCredentials) {
			c.Status(http.StatusUnauthorized, "Invalid email or password")
			return
		}
		c.Status(http.StatusInternalServerError, "Authentication failed")
		return
	}

	// Generate token
	token, err := auth.GenerateToken(user.ID.String())
	if err != nil {
		c.Status(http.StatusInternalServerError, "Failed to generate token")
		return
	}

	// Return token and user info
	c.JSON(http.StatusOK, map[string]interface{}{
		"token": token,
		"user": map[string]interface{}{
			"id":       user.ID.String(),
			"email":    user.Email,
			"name":     user.Name.String,
			"username": user.Username.String,
		},
		"message": "Login successful",
	})
}

// ForgotPassword initiates password reset
func ForgotPassword(c *router.Context) {
	if userService == nil {
		c.Status(http.StatusInternalServerError, "User service not initialized")
		return
	}
	var req ForgotPasswordRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		c.Status(http.StatusBadRequest, "Invalid request format")
		return
	}

	if req.Email == "" || !validator.Matches(req.Email, validator.EmailRX) {
		c.Status(http.StatusBadRequest, "Valid email is required")
		return
	}

	// Call service to initiate password reset
	err := userService.ForgotPassword(c.Request.Context(), req.Email)
	if err != nil {
		// We don't reveal if the email exists or not for security reasons
		c.Status(http.StatusInternalServerError, "Failed to process request")
		return
	}

	// Always return success even if email not found for security
	c.JSON(http.StatusOK, map[string]string{
		"message": "If your email exists in our system, you will receive password reset instructions",
	})
}

// ResetPassword completes password reset with token
func ResetPassword(c *router.Context) {
	if userService == nil {
		c.Status(http.StatusInternalServerError, "User service not initialized")
		return
	}
	// Get token from URL parameter
	token := c.Param("token")
	if token == "" {
		c.Status(http.StatusBadRequest, "Reset token is required")
		return
	}

	var req ResetPasswordRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		c.Status(http.StatusBadRequest, "Invalid request format")
		return
	}

	// Validate password
	if req.NewPassword == "" || !validator.MinChars(req.NewPassword, 8) {
		c.Status(http.StatusBadRequest, "Password must be at least 8 characters")
		return
	}

	// Call service to reset password
	err := userService.ResetPassword(c.Request.Context(), token, req.NewPassword)
	if err != nil {
		c.Status(http.StatusBadRequest, "Invalid or expired reset token")
		return
	}

	c.JSON(http.StatusOK, map[string]string{
		"message": "Password has been reset successfully",
	})
}

// Helper function to validate registration data
func validateRegisterRequest(req RegisterRequest) error {
	if req.Email == "" || !validator.Matches(req.Email, validator.EmailRX) {
		return errors.New("valid email address is required")
	}

	if req.Password == "" || !validator.MinChars(req.Password, 8) {
		return errors.New("password must be at least 8 characters")
	}

	if req.Name != "" && !validator.MaxChars(req.Name, 100) {
		return errors.New("name cannot exceed 100 characters")
	}

	if req.Username != "" && !validator.MaxChars(req.Username, 50) {
		return errors.New("username cannot exceed 50 characters")
	}

	return nil
}
