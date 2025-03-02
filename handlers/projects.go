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

// projectService is retrieved from the application's dependency container
var projectService *services.ProjectService

// SetProjectService sets the project service for handlers
func SetProjectService(service *services.ProjectService) {
	projectService = service
}

// CreateProjectRequest represents project creation input
type CreateProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	TeamID      string `json:"team_id,omitempty"`
}

// UpdateProjectRequest represents project update input
type UpdateProjectRequest struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status,omitempty"`
}

// ListProjects returns all projects accessible to the authenticated user
func ListProjects(c *router.Context) {
	userID, ok := c.Request.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		c.Status(http.StatusUnauthorized, "User not authenticated")
		return
	}

	// Get query parameters for optional filtering
	teamID := c.Query("team_id")
	status := c.Query("status")

	// Get projects for the user
	var projects []services.ProjectInfo
	var err error

	if teamID != "" {
		// Get team projects if team_id is provided
		projects, err = projectService.GetTeamProjects(c.Request.Context(), teamID, userID)
		if err != nil {
			handleProjectError(c, err)
			return
		}
	} else if status != "" {
		// Get projects by status if status is provided
		projects, err = projectService.GetProjectsByStatus(c.Request.Context(), status, userID)
		if err != nil {
			handleProjectError(c, err)
			return
		}
	} else {
		// Get all user projects
		projects, err = projectService.GetUserProjects(c.Request.Context(), userID)
		if err != nil {
			handleProjectError(c, err)
			return
		}
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"projects": projects,
		"count":    len(projects),
	})
}

// CreateProject creates a new project
func CreateProject(c *router.Context) {
	userID, ok := c.Request.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		c.Status(http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req CreateProjectRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		c.Status(http.StatusBadRequest, "Invalid request format")
		return
	}

	// Validate project name
	if req.Name == "" {
		c.Status(http.StatusBadRequest, "Project name is required")
		return
	}

	params := store.CreateProjectParams{
		Name:        req.Name,
		Description: pgtype.Text{String: req.Description, Valid: req.Description != ""},
		OwnerID:     pgtype.UUID{},
	}

	if req.TeamID != "" {
		var teamUUID pgtype.UUID
		if err := teamUUID.Scan(req.TeamID); err != nil {
			c.Status(http.StatusBadRequest, "Invalid team ID format")
			return
		}
		params.TeamID = pgtype.UUID{Bytes: teamUUID.Bytes, Valid: true}
	}

	project, err := projectService.CreateProject(c.Request.Context(), params, userID)
	if err != nil {
		handleProjectError(c, err)
		return
	}

	c.JSON(http.StatusCreated, project)
}

// GetProject returns a specific project by ID
func GetProject(c *router.Context) {
	userID, ok := c.Request.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		c.Status(http.StatusUnauthorized, "User not authenticated")
		return
	}

	// Get project ID from URL
	projectID := c.Param("id")
	if projectID == "" {
		c.Status(http.StatusBadRequest, "Project ID is required")
		return
	}

	// Get project
	project, err := projectService.GetProjectByID(c.Request.Context(), projectID, userID)
	if err != nil {
		handleProjectError(c, err)
		return
	}

	c.JSON(http.StatusOK, project)
}

// UpdateProject updates a project's details
func UpdateProject(c *router.Context) {
	userID, ok := c.Request.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		c.Status(http.StatusUnauthorized, "User not authenticated")
		return
	}

	// Get project ID from URL
	projectID := c.Param("id")
	if projectID == "" {
		c.Status(http.StatusBadRequest, "Project ID is required")
		return
	}

	// Parse update request
	var req UpdateProjectRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		c.Status(http.StatusBadRequest, "Invalid request format")
		return
	}

	// Create update params
	updates := services.ProjectUpdates{
		Name:        req.Name,
		Description: req.Description,
		Status:      req.Status,
	}

	// Update project
	if err := projectService.UpdateProject(c.Request.Context(), projectID, updates, userID); err != nil {
		handleProjectError(c, err)
		return
	}

	// Get updated project
	project, err := projectService.GetProjectByID(c.Request.Context(), projectID, userID)
	if err != nil {
		handleProjectError(c, err)
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Project updated successfully",
		"project": project,
	})
}

// DeleteProject deletes a project
func DeleteProject(c *router.Context) {
	userID, ok := c.Request.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		c.Status(http.StatusUnauthorized, "User not authenticated")
		return
	}

	// Get project ID from URL
	projectID := c.Param("id")
	if projectID == "" {
		c.Status(http.StatusBadRequest, "Project ID is required")
		return
	}

	// Delete project
	if err := projectService.DeleteProject(c.Request.Context(), projectID, userID); err != nil {
		handleProjectError(c, err)
		return
	}

	c.Status(http.StatusOK, "Project deleted successfully")
}

// Helper function to handle project errors
func handleProjectError(c *router.Context, err error) {
	switch {
	case errors.Is(err, services.ErrProjectNotFound):
		c.Status(http.StatusNotFound, "Project not found")
	case errors.Is(err, services.ErrNotProjectOwner):
		c.Status(http.StatusForbidden, "You don't have permission to access this project")
	case errors.Is(err, services.ErrInvalidProjectData):
		c.Status(http.StatusBadRequest, "Invalid project data")
	default:
		c.Status(http.StatusInternalServerError, "An error occurred processing your request")
	}
}
