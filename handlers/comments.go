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

// CreateCommentRequest represents the input for creating a comment
type CreateCommentRequest struct {
	Content string `json:"content"`
	TaskID  string `json:"task_id,omitempty"` // Optional, one of task_id or issue_id must be provided
}

// UpdateCommentRequest represents the input for updating a comment
type UpdateCommentRequest struct {
	Content string `json:"content"`
}

// searchService is retrieved from the application's dependency container
var commentService *services.CommentService

// SetSearchService sets the search service for handlers
func SetCommentService(service *services.CommentService) {
	commentService = service
}

// ListComments returns all comments for a specific issue or task
func ListComments(c *router.Context) {
	if commentService == nil {
		c.Status(http.StatusInternalServerError, "Comment service not initialized")
		return
	}

	issueID := c.Param("ticket_id")
	taskID := c.Param("task_id") // Optional task_id from route or query
	userID, ok := c.Request.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		c.Status(http.StatusUnauthorized, "User not authenticated")
		return
	}

	var comments []services.CommentInfo
	var err error
	if issueID != "" {
		comments, err = commentService.GetIssueComments(c.Request.Context(), issueID, userID)
	} else if taskID != "" {
		comments, err = commentService.GetTaskComments(c.Request.Context(), taskID, userID)
	} else {
		c.Status(http.StatusBadRequest, "Issue ID or Task ID is required")
		return
	}

	if err != nil {
		c.Status(http.StatusInternalServerError, "Failed to retrieve comments")
		return
	}

	c.JSON(http.StatusOK, comments)
}

// CreateComment creates a new comment on an issue or task
func CreateComment(c *router.Context) {
	if commentService == nil {
		c.Status(http.StatusInternalServerError, "Comment service not initialized")
		return
	}

	userID, ok := c.Request.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		c.Status(http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req CreateCommentRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		c.Status(http.StatusBadRequest, "Invalid request format")
		return
	}

	if req.Content == "" {
		c.Status(http.StatusBadRequest, "Comment content is required")
		return
	}

	issueID := c.Param("ticket_id") // From route, if under /tickets
	var scannedIssueID, scannedTaskID pgtype.UUID
	if issueID != "" {
		if err := scannedIssueID.Scan(issueID); err != nil {
			c.Status(http.StatusBadRequest, "Invalid issue ID format")
			return
		}
	}
	if req.TaskID != "" {
		if err := scannedTaskID.Scan(req.TaskID); err != nil {
			c.Status(http.StatusBadRequest, "Invalid task ID format")
			return
		}
	}

	// Ensure exactly one of issueID or taskID is provided
	if (scannedIssueID.Valid && scannedTaskID.Valid) || (!scannedIssueID.Valid && !scannedTaskID.Valid) {
		c.Status(http.StatusBadRequest, "Exactly one of issue ID or task ID must be provided")
		return
	}

	params := store.CreateCommentParams{
		Content: req.Content,
		IssueID: scannedIssueID,
		TaskID:  scannedTaskID,
	}

	comment, err := commentService.CreateComment(c.Request.Context(), params, userID)
	if err != nil {
		if errors.Is(err, services.ErrInvalidCommentData) {
			c.Status(http.StatusBadRequest, err.Error())
			return
		}
		c.Status(http.StatusInternalServerError, "Failed to create comment")
		return
	}

	c.JSON(http.StatusCreated, map[string]interface{}{
		"id":       comment.ID.String(),
		"content":  comment.Content,
		"user_id":  comment.UserID.String(),
		"issue_id": comment.IssueID.String(),
		"task_id":  comment.TaskID.String(),
		"message":  "Comment created successfully",
	})
}

// UpdateComment updates an existing comment
func UpdateComment(c *router.Context) {
	if commentService == nil {
		c.Status(http.StatusInternalServerError, "Comment service not initialized")
		return
	}

	commentID := c.Param("id")
	if commentID == "" {
		c.Status(http.StatusBadRequest, "Comment ID is required")
		return
	}

	userID, ok := c.Request.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		c.Status(http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req UpdateCommentRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		c.Status(http.StatusBadRequest, "Invalid request format")
		return
	}

	if req.Content == "" {
		c.Status(http.StatusBadRequest, "Comment content is required")
		return
	}

	var scannedCommentID pgtype.UUID
	if err := scannedCommentID.Scan(commentID); err != nil {
		c.Status(http.StatusBadRequest, "Invalid comment ID format")
		return
	}

	params := store.UpdateCommentParams{
		ID:      scannedCommentID,
		Content: req.Content,
	}

	if err := commentService.UpdateComment(c.Request.Context(), params, userID); err != nil {
		if errors.Is(err, services.ErrInvalidCommentData) {
			c.Status(http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, services.ErrNotCommentAuthor) {
			c.Status(http.StatusForbidden, "Only the comment author can update this comment")
			return
		}
		c.Status(http.StatusInternalServerError, "Failed to update comment")
		return
	}

	c.JSON(http.StatusOK, map[string]string{
		"message": "Comment updated successfully",
	})
}

// DeleteComment deletes an existing comment
func DeleteComment(c *router.Context) {
	if commentService == nil {
		c.Status(http.StatusInternalServerError, "Comment service not initialized")
		return
	}

	commentID := c.Param("id")
	if commentID == "" {
		c.Status(http.StatusBadRequest, "Comment ID is required")
		return
	}

	userID, ok := c.Request.Context().Value(middleware.UserIDKey).(string)
	if !ok || userID == "" {
		c.Status(http.StatusUnauthorized, "User not authenticated")
		return
	}

	if err := commentService.DeleteComment(c.Request.Context(), commentID, userID); err != nil {
		if errors.Is(err, services.ErrNotCommentAuthor) {
			c.Status(http.StatusForbidden, "Only the comment author or project owner can delete this comment")
			return
		}
		c.Status(http.StatusInternalServerError, "Failed to delete comment")
		return
	}

	c.Status(http.StatusOK, "Comment deleted successfully")
}
