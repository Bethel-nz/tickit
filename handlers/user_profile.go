package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Bethel-nz/tickit/app/middleware"
	"github.com/Bethel-nz/tickit/app/router"
	"github.com/Bethel-nz/tickit/internal/services"
)

// GetUserProfile returns the authenticated user's profile
func GetUserProfile(c *router.Context) {
	userID, ok := c.Request.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		c.Status(http.StatusUnauthorized, "User not authenticated")
		return
	}

	profile, err := userService.GetUserProfile(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			c.Status(http.StatusNotFound, "User not found")
			return
		}
		c.Status(http.StatusInternalServerError, "Failed to retrieve user profile")
		return
	}

	c.JSON(http.StatusOK, profile)
}

// UpdateUserProfile updates the authenticated user's profile
func UpdateUserProfile(c *router.Context) {
	// Get user ID from context
	userID, ok := c.Request.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		c.Status(http.StatusUnauthorized, "User not authenticated")
		return
	}

	// Parse request body
	var req services.UserProfileUpdate
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		c.Status(http.StatusBadRequest, "Invalid request format")
		return
	}

	// Update profile
	if err := userService.UpdateUserProfile(c.Request.Context(), userID, req); err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			c.Status(http.StatusNotFound, "User not found")
			return
		}
		c.Status(http.StatusInternalServerError, "Failed to update profile")
		return
	}

	c.JSON(http.StatusOK, map[string]string{
		"message": "Profile updated successfully",
	})
}

// ChangePassword handles password change for authenticated users
func ChangePassword(c *router.Context) {
	// Get user ID from context
	userID, ok := c.Request.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		c.Status(http.StatusUnauthorized, "User not authenticated")
		return
	}

	// Parse request
	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		c.Status(http.StatusBadRequest, "Invalid request format")
		return
	}

	// Validate input
	if req.CurrentPassword == "" || req.NewPassword == "" {
		c.Status(http.StatusBadRequest, "Current password and new password are required")
		return
	}

	// Change password
	err := userService.ChangePassword(c.Request.Context(), userID, req.CurrentPassword, req.NewPassword)
	if err != nil {
		if errors.Is(err, services.ErrInvalidCredentials) {
			c.Status(http.StatusUnauthorized, "Current password is incorrect")
			return
		}
		c.Status(http.StatusInternalServerError, "Failed to change password")
		return
	}

	c.JSON(http.StatusOK, map[string]string{
		"message": "Password changed successfully",
	})
}

// DeleteAccount handles account deletion for authenticated users
func DeleteAccount(c *router.Context) {
	userID, ok := c.Request.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		c.Status(http.StatusUnauthorized, "User not authenticated")
		return
	}

	// Delete account
	if err := userService.DeleteAccount(c.Request.Context(), userID); err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			c.Status(http.StatusNotFound, "User not found")
			return
		}
		c.Status(http.StatusInternalServerError, "Failed to delete account")
		return
	}

	c.Status(http.StatusOK, "Account deleted successfully")
}
