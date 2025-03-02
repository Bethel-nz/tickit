package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Bethel-nz/tickit/app/middleware"
	"github.com/Bethel-nz/tickit/app/router"
	"github.com/Bethel-nz/tickit/internal/database/store"
	"github.com/Bethel-nz/tickit/internal/services"
	"github.com/jackc/pgx/v5/pgtype"
)

// teamService is retrieved from the application's dependency container
var teamService *services.TeamService

// SetTeamService sets the team service for handlers
func SetTeamService(service *services.TeamService) {
	teamService = service
}

// TeamRequest represents team creation/update input
type TeamRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	AvatarURL   string `json:"avatar_url,omitempty"`
}

// TeamMemberRequest represents a request to add a member to a team
type TeamMemberRequest struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

// ListTeams returns all teams a user is a member of
func ListTeams(c *router.Context) {
	userID, ok := c.Request.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		c.Status(http.StatusUnauthorized, "User not authenticated")
		return
	}

	teams, err := teamService.GetUserTeams(c.Request.Context(), userID)
	if err != nil {
		handleTeamError(c, err)
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"teams": teams,
		"count": len(teams),
	})
}

// CreateTeam creates a new team
func CreateTeam(c *router.Context) {
	userID, ok := c.Request.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		c.Status(http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req TeamRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		c.Status(http.StatusBadRequest, "Invalid request format")
		return
	}

	// Validate team name
	if req.Name == "" {
		c.Status(http.StatusBadRequest, "Team name is required")
		return
	}

	// Create params for team creation
	params := store.CreateTeamParams{
		Name:        req.Name,
		Description: pgtype.Text{String: req.Description, Valid: req.Description != ""},
		AvatarUrl:   pgtype.Text{String: req.AvatarURL, Valid: req.AvatarURL != ""},
	}

	// Create team and add creator as admin
	team, err := teamService.CreateTeam(c.Request.Context(), params, userID)
	if err != nil {
		handleTeamError(c, err)
		return
	}

	c.JSON(http.StatusCreated, team)
}

// GetTeam returns a specific team
func GetTeam(c *router.Context) {
	userID, ok := c.Request.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		c.Status(http.StatusUnauthorized, "User not authenticated")
		return
	}

	teamID := c.Param("id")
	if teamID == "" {
		c.Status(http.StatusBadRequest, "Team ID is required")
		return
	}

	team, err := teamService.GetTeamByID(c.Request.Context(), teamID)
	if err != nil {
		handleTeamError(c, err)
		return
	}

	c.JSON(http.StatusOK, team)
}

// UpdateTeam updates a team
func UpdateTeam(c *router.Context) {
	userID, ok := c.Request.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		c.Status(http.StatusUnauthorized, "User not authenticated")
		return
	}

	teamID := c.Param("id")
	if teamID == "" {
		c.Status(http.StatusBadRequest, "Team ID is required")
		return
	}

	var req TeamRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		c.Status(http.StatusBadRequest, "Invalid request format")
		return
	}

	params := store.UpdateTeamParams{
		Name:        req.Name,
		Description: pgtype.Text{String: req.Description, Valid: req.Description != ""},
		AvatarUrl:   pgtype.Text{String: req.AvatarURL, Valid: req.AvatarURL != ""},
	}

	if err := teamService.UpdateTeam(c.Request.Context(), params, userID); err != nil {
		handleTeamError(c, err)
		return
	}

	team, err := teamService.GetTeamByID(c.Request.Context(), teamID)
	if err != nil {
		handleTeamError(c, err)
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Team updated successfully",
		"team":    team,
	})
}

// DeleteTeam deletes a team
func DeleteTeam(c *router.Context) {
	userID, ok := c.Request.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		c.Status(http.StatusUnauthorized, "User not authenticated")
		return
	}

	teamID := c.Param("id")
	if teamID == "" {
		c.Status(http.StatusBadRequest, "Team ID is required")
		return
	}

	if err := teamService.DeleteTeam(c.Request.Context(), teamID, userID); err != nil {
		handleTeamError(c, err)
		return
	}

	c.Status(http.StatusOK, "Team deleted successfully")
}

// AddTeamMember adds a user to a team
func AddTeamMember(c *router.Context) {
	userID, ok := c.Request.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		c.Status(http.StatusUnauthorized, "User not authenticated")
		return
	}

	teamID := c.Param("id")
	if teamID == "" {
		c.Status(http.StatusBadRequest, "Team ID is required")
		return
	}

	var req TeamMemberRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		c.Status(http.StatusBadRequest, "Invalid request format")
		return
	}

	if req.UserID == "" {
		c.Status(http.StatusBadRequest, "User ID is required")
		return
	}

	if req.Role == "" {
		req.Role = "member" // Default role
	}

	if err := teamService.AddMember(c.Request.Context(), teamID, req.UserID, req.Role, userID); err != nil {
		handleTeamError(c, err)
		return
	}

	c.JSON(http.StatusOK, map[string]string{
		"message": "Member added successfully",
	})
}

// RemoveTeamMember removes a user from a team
func RemoveTeamMember(c *router.Context) {
	userID, ok := c.Request.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		c.Status(http.StatusUnauthorized, "User not authenticated")
		return
	}

	teamID := c.Param("id")
	if teamID == "" {
		c.Status(http.StatusBadRequest, "Team ID is required")
		return
	}

	memberID := c.Param("user_id")
	if memberID == "" {
		c.Status(http.StatusBadRequest, "Member ID is required")
		return
	}

	if err := teamService.RemoveMember(c.Request.Context(), teamID, memberID, userID); err != nil {
		handleTeamError(c, err)
		return
	}

	c.JSON(http.StatusOK, map[string]string{
		"message": "Member removed successfully",
	})
}

// ListTeamMembers returns all members of a team
func ListTeamMembers(c *router.Context) {
	userID, ok := c.Request.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		c.Status(http.StatusUnauthorized, "User not authenticated")
		return
	}

	teamID := c.Param("id")
	if teamID == "" {
		c.Status(http.StatusBadRequest, "Team ID is required")
		return
	}

	members, err := teamService.GetTeamMembers(c.Request.Context(), teamID, userID)
	if err != nil {
		handleTeamError(c, err)
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"members": members,
		"count":   len(members),
	})
}

func handleTeamError(c *router.Context, err error) {
	switch {
	case errors.Is(err, services.ErrTeamNotFound):
		c.Status(http.StatusNotFound, "Team not found")
	case errors.Is(err, services.ErrUnauthorized):
		c.Status(http.StatusForbidden, "Only team admins can perform this action")
	case errors.Is(err, services.ErrNotMember):
		c.Status(http.StatusForbidden, "You are not a member of this team")
	default:
		c.Status(http.StatusInternalServerError, "An error occurred processing your request")
	}
}
