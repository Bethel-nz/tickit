package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/Bethel-nz/tickit/app/middleware"
	"github.com/Bethel-nz/tickit/app/router"
	"github.com/Bethel-nz/tickit/internal/database/store"
	"github.com/Bethel-nz/tickit/internal/services"
	"github.com/jackc/pgx/v5/pgtype"
)

// The service is used to interact with issue/ticket data
var issueService *services.IssueService

// SetIssueService sets the issue service for handlers
func SetIssueService(service *services.IssueService) {
	issueService = service
}

// TicketRequest represents the data structure for creating/updating tickets (issues)
type TicketRequest struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status,omitempty"`
	AssigneeID  string `json:"assignee_id,omitempty"`
	DueDate     string `json:"due_date,omitempty"` // RFC3339 format
}

// ListTickets returns all tickets for a project
func ListTickets(c *router.Context) {
	if issueService == nil {
		c.Status(http.StatusInternalServerError, "Issue service not initialized")
		return
	}
	userID, ok := c.Request.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		c.Status(http.StatusUnauthorized, "User not authenticated")
		return
	}

	projectID := c.Param("project_id")
	if projectID == "" {
		c.Status(http.StatusBadRequest, "Project ID is required")
		return
	}

	// Optional status filter
	status := c.Query("status")

	var tickets []services.IssueInfo
	var err error

	if status != "" {
		tickets, err = issueService.GetIssuesByStatus(c.Request.Context(), projectID, status, userID)
	} else {
		tickets, err = issueService.GetProjectIssues(c.Request.Context(), projectID, userID)
	}

	if err != nil {
		handleIssueError(c, err)
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"tickets": tickets,
		"count":   len(tickets),
	})
}

// CreateTicket creates a new ticket
func CreateTicket(c *router.Context) {
	if issueService == nil {
		c.Status(http.StatusInternalServerError, "Issue service not initialized")
		return
	}
	userID, ok := c.Request.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		c.Status(http.StatusUnauthorized, "User not authenticated")
		return
	}

	projectID := c.Param("project_id")
	if projectID == "" {
		c.Status(http.StatusBadRequest, "Project ID is required")
		return
	}

	var req TicketRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		c.Status(http.StatusBadRequest, "Invalid request format")
		return
	}

	// Validate required fields
	if req.Title == "" {
		c.Status(http.StatusBadRequest, "Title is required")
		return
	}

	// Create issue parameters
	var projectUUID pgtype.UUID
	if err := projectUUID.Scan(projectID); err != nil {
		c.Status(http.StatusBadRequest, "Invalid project ID format")
		return
	}

	var userUUID pgtype.UUID
	if err := userUUID.Scan(userID); err != nil {
		c.Status(http.StatusBadRequest, "Invalid user ID format")
		return
	}

	params := store.CreateIssueParams{
		ProjectID:   projectUUID,
		Title:       req.Title,
		Description: pgtype.Text{String: req.Description, Valid: req.Description != ""},
		Status:      pgtype.Text{String: req.Status, Valid: req.Status != ""},
		ReporterID:  userUUID,
	}

	// Set assignee if provided
	if req.AssigneeID != "" {
		var assigneeUUID pgtype.UUID
		if err := assigneeUUID.Scan(req.AssigneeID); err != nil {
			c.Status(http.StatusBadRequest, "Invalid assignee ID format")
			return
		}
		params.AssigneeID = assigneeUUID
	}

	// Set due date if provided
	if req.DueDate != "" {
		dueDate, err := time.Parse(time.RFC3339, req.DueDate)
		if err != nil {
			c.Status(http.StatusBadRequest, "Invalid due date format, use RFC3339")
			return
		}
		params.DueDate = pgtype.Timestamp{Time: dueDate, Valid: true}
	}

	// Create the issue
	ticket, err := issueService.CreateIssue(c.Request.Context(), params, userID)
	if err != nil {
		handleIssueError(c, err)
		return
	}

	c.JSON(http.StatusCreated, map[string]interface{}{
		"message": "Ticket created successfully",
		"ticket":  ticket,
	})
}

// GetTicket returns a specific ticket
func GetTicket(c *router.Context) {
	if issueService == nil {
		c.Status(http.StatusInternalServerError, "Issue service not initialized")
		return
	}
	userID, ok := c.Request.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		c.Status(http.StatusUnauthorized, "User not authenticated")
		return
	}

	ticketID := c.Param("id")
	if ticketID == "" {
		c.Status(http.StatusBadRequest, "Ticket ID is required")
		return
	}

	ticket, err := issueService.GetIssueByID(c.Request.Context(), ticketID, userID)
	if err != nil {
		handleIssueError(c, err)
		return
	}

	c.JSON(http.StatusOK, ticket)
}

// UpdateTicket updates an existing ticket
func UpdateTicket(c *router.Context) {
	if issueService == nil {
		c.Status(http.StatusInternalServerError, "Issue service not initialized")
		return
	}
	userID, ok := c.Request.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		c.Status(http.StatusUnauthorized, "User not authenticated")
		return
	}

	ticketID := c.Param("id")
	if ticketID == "" {
		c.Status(http.StatusBadRequest, "Ticket ID is required")
		return
	}

	var req TicketRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		c.Status(http.StatusBadRequest, "Invalid request format")
		return
	}

	// Create updates
	updates := services.IssueUpdates{
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
		AssigneeID:  req.AssigneeID,
	}

	// Parse due date if provided
	if req.DueDate != "" {
		dueDate, err := time.Parse(time.RFC3339, req.DueDate)
		if err != nil {
			c.Status(http.StatusBadRequest, "Invalid due date format, use RFC3339")
			return
		}
		updates.DueDate = &dueDate
	}

	if err := issueService.UpdateIssue(c.Request.Context(), ticketID, updates, userID); err != nil {
		handleIssueError(c, err)
		return
	}

	// Get updated ticket
	ticket, err := issueService.GetIssueByID(c.Request.Context(), ticketID, userID)
	if err != nil {
		handleIssueError(c, err)
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Ticket updated successfully",
		"ticket":  ticket,
	})
}

// DeleteTicket deletes a ticket
func DeleteTicket(c *router.Context) {
	if issueService == nil {
		c.Status(http.StatusInternalServerError, "Issue service not initialized")
		return
	}
	userID, ok := c.Request.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		c.Status(http.StatusUnauthorized, "User not authenticated")
		return
	}

	ticketID := c.Param("id")
	if ticketID == "" {
		c.Status(http.StatusBadRequest, "Ticket ID is required")
		return
	}

	if err := issueService.DeleteIssue(c.Request.Context(), ticketID, userID); err != nil {
		handleIssueError(c, err)
		return
	}

	c.Status(http.StatusOK, "Ticket deleted successfully")
}

// AssignTicket assigns a ticket to a user
func AssignTicket(c *router.Context) {
	if issueService == nil {
		c.Status(http.StatusInternalServerError, "Issue service not initialized")
		return
	}
	userID, ok := c.Request.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		c.Status(http.StatusUnauthorized, "User not authenticated")
		return
	}

	ticketID := c.Param("id")
	if ticketID == "" {
		c.Status(http.StatusBadRequest, "Ticket ID is required")
		return
	}

	var req struct {
		AssigneeID string `json:"assignee_id"`
	}
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		c.Status(http.StatusBadRequest, "Invalid request format")
		return
	}

	if req.AssigneeID == "" {
		c.Status(http.StatusBadRequest, "Assignee ID is required")
		return
	}

	// Create updates with just the assignee
	updates := services.IssueUpdates{
		AssigneeID: req.AssigneeID,
	}

	if err := issueService.UpdateIssue(c.Request.Context(), ticketID, updates, userID); err != nil {
		handleIssueError(c, err)
		return
	}

	// Get updated ticket
	ticket, err := issueService.GetIssueByID(c.Request.Context(), ticketID, userID)
	if err != nil {
		handleIssueError(c, err)
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Ticket assigned successfully",
		"ticket":  ticket,
	})
}

// Helper function to handle issue errors
func handleIssueError(c *router.Context, err error) {
	switch {
	case errors.Is(err, services.ErrIssueNotFound):
		c.Status(http.StatusNotFound, "Ticket not found")
	case errors.Is(err, services.ErrProjectNotFound):
		c.Status(http.StatusNotFound, "Project not found")
	case errors.Is(err, services.ErrNotProjectOwner):
		c.Status(http.StatusForbidden, "You don't have permission to access this project")
	case errors.Is(err, services.ErrInvalidIssueData):
		c.Status(http.StatusBadRequest, "Invalid ticket data")
	default:
		c.Status(http.StatusInternalServerError, "An error occurred processing your request")
	}
}
