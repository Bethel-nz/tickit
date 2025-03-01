package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/Bethel-nz/tickit/internal/database/store"
	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v5/pgtype"
)

// Comment service errors
var (
	ErrCommentNotFound    = errors.New("comment not found")
	ErrInvalidCommentData = errors.New("invalid comment data")
	ErrNotCommentAuthor   = errors.New("user is not the comment author")
)

// CommentInfo represents comment information returned to clients
type CommentInfo struct {
	ID        string `json:"id"`
	Content   string `json:"content"`
	UserID    string `json:"user_id"`
	IssueID   string `json:"issue_id,omitempty"`
	TaskID    string `json:"task_id,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at,omitempty"`
	// Additional user info for display
	UserName     string `json:"user_name,omitempty"`
	UserEmail    string `json:"user_email,omitempty"`
	UserUsername string `json:"user_username,omitempty"`
	UserAvatar   string `json:"user_avatar,omitempty"`
}

type CommentService struct {
	queries        *store.Queries
	cache          *redis.Client
	projectService *ProjectService
}

func NewCommentService(queries *store.Queries, cache *redis.Client, projectService *ProjectService) *CommentService {
	return &CommentService{
		queries:        queries,
		cache:          cache,
		projectService: projectService,
	}
}

// CreateComment creates a new comment for an issue or task
func (s *CommentService) CreateComment(ctx context.Context, params store.CreateCommentParams, userID string) (*store.Comment, error) {
	// Validate comment data
	if params.Content == "" {
		return nil, fmt.Errorf("%w: comment content is required", ErrInvalidCommentData)
	}

	// Make sure user ID matches
	var userUUID pgtype.UUID
	if err := userUUID.Scan(userID); err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}
	params.UserID = userUUID

	// Verify the user has access to the issue or task being commented on
	if err := s.verifyCommentableAccess(ctx, params.IssueID, params.TaskID, userID); err != nil {
		return nil, err
	}

	// Create comment in database
	comment, err := s.queries.CreateComment(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create comment: %w", err)
	}

	// Invalidate comments list cache
	if comment.IssueID.Valid {
		s.invalidateCommentsCache(ctx, "issue", comment.IssueID.String())
	} else if comment.TaskID.Valid {
		s.invalidateCommentsCache(ctx, "task", comment.TaskID.String())
	}

	return &comment, nil
}

// GetIssueComments retrieves all comments for an issue
func (s *CommentService) GetIssueComments(ctx context.Context, issueID string, userID string) ([]CommentInfo, error) {
	var issueUUID pgtype.UUID
	if err := issueUUID.Scan(issueID); err != nil {
		return nil, fmt.Errorf("invalid issue ID: %w", err)
	}

	// Verify the user has access to the issue
	issue, err := s.queries.GetIssueByID(ctx, issueUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get issue: %w", err)
	}

	// Check access to the project this issue belongs to
	if err := s.projectService.verifyProjectAccess(ctx, &store.Project{ID: issue.ProjectID}, userID); err != nil {
		return nil, err
	}

	// Try to get from cache
	cacheKey := fmt.Sprintf("issue:%s:comments", issueID)
	cachedComments, err := s.cache.Get(ctx, cacheKey).Result()
	if err == nil {
		var comments []CommentInfo
		if err := json.Unmarshal([]byte(cachedComments), &comments); err == nil {
			return comments, nil
		}
	}

	// Get from database
	dbComments, err := s.queries.GetIssueComments(ctx, issueUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get issue comments: %w", err)
	}

	// Convert to our response format
	comments := make([]CommentInfo, len(dbComments))
	for i, c := range dbComments {
		comments[i] = CommentInfo{
			ID:           c.ID.String(),
			Content:      c.Content,
			UserID:       c.UserID.String(),
			IssueID:      issueID,
			CreatedAt:    c.CreatedAt.Time.Format(time.RFC3339),
			UpdatedAt:    c.UpdatedAt.Time.Format(time.RFC3339),
			UserName:     c.Name.String,
			UserEmail:    c.Email,
			UserUsername: c.Username.String,
			UserAvatar:   c.AvatarUrl.String,
		}
	}

	// Cache the result
	commentsJSON, err := json.Marshal(comments)
	if err == nil {
		if err := s.cache.Set(ctx, cacheKey, commentsJSON, 10*time.Minute).Err(); err != nil {
			log.Printf("Failed to cache issue comments: %v", err)
		}
	}

	return comments, nil
}

// GetTaskComments retrieves all comments for a task
func (s *CommentService) GetTaskComments(ctx context.Context, taskID string, userID string) ([]CommentInfo, error) {
	var taskUUID pgtype.UUID
	if err := taskUUID.Scan(taskID); err != nil {
		return nil, fmt.Errorf("invalid task ID: %w", err)
	}

	// Verify the user has access to the task
	task, err := s.queries.GetTaskByID(ctx, taskUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	// Check access to the project this task belongs to
	if err := s.projectService.verifyProjectAccess(ctx, &store.Project{ID: task.ProjectID}, userID); err != nil {
		return nil, err
	}

	// Try to get from cache
	cacheKey := fmt.Sprintf("task:%s:comments", taskID)
	cachedComments, err := s.cache.Get(ctx, cacheKey).Result()
	if err == nil {
		var comments []CommentInfo
		if err := json.Unmarshal([]byte(cachedComments), &comments); err == nil {
			return comments, nil
		}
	}

	// Get from database
	dbComments, err := s.queries.GetTaskComments(ctx, taskUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task comments: %w", err)
	}

	// Convert to our response format
	comments := make([]CommentInfo, len(dbComments))
	for i, c := range dbComments {
		comments[i] = CommentInfo{
			ID:           c.ID.String(),
			Content:      c.Content,
			UserID:       c.UserID.String(),
			TaskID:       taskID,
			CreatedAt:    c.CreatedAt.Time.Format(time.RFC3339),
			UpdatedAt:    c.UpdatedAt.Time.Format(time.RFC3339),
			UserName:     c.Name.String,
			UserEmail:    c.Email,
			UserUsername: c.Username.String,
			UserAvatar:   c.AvatarUrl.String,
		}
	}

	// Cache the result
	commentsJSON, err := json.Marshal(comments)
	if err == nil {
		if err := s.cache.Set(ctx, cacheKey, commentsJSON, 10*time.Minute).Err(); err != nil {
			log.Printf("Failed to cache task comments: %v", err)
		}
	}

	return comments, nil
}

// UpdateComment updates a comment
func (s *CommentService) UpdateComment(ctx context.Context, params store.UpdateCommentParams, userID string) error {
	// Validate comment content
	if params.Content == "" {
		return fmt.Errorf("%w: comment content is required", ErrInvalidCommentData)
	}

	// Get the comment to check ownership
	comment, err := s.queries.GetCommentByID(ctx, params.ID)
	if err != nil {
		return fmt.Errorf("failed to get comment: %w", err)
	}

	// Verify the user is the author of the comment
	var userUUID pgtype.UUID
	if err := userUUID.Scan(userID); err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	if comment.UserID != userUUID {
		return ErrNotCommentAuthor
	}

	// Update the comment
	if err := s.queries.UpdateComment(ctx, params); err != nil {
		return fmt.Errorf("failed to update comment: %w", err)
	}

	// Invalidate comments list cache
	if comment.IssueID.Valid {
		s.invalidateCommentsCache(ctx, "issue", comment.IssueID.String())
	} else if comment.TaskID.Valid {
		s.invalidateCommentsCache(ctx, "task", comment.TaskID.String())
	}

	return nil
}

// DeleteComment deletes a comment
func (s *CommentService) DeleteComment(ctx context.Context, commentID string, userID string) error {
	var commentUUID pgtype.UUID
	if err := commentUUID.Scan(commentID); err != nil {
		return fmt.Errorf("invalid comment ID: %w", err)
	}

	// Get the comment to check ownership and get the related issue/task ID
	comment, err := s.queries.GetCommentByID(ctx, commentUUID)
	if err != nil {
		return fmt.Errorf("failed to get comment: %w", err)
	}

	// Verify the user is the author of the comment
	var userUUID pgtype.UUID
	if err := userUUID.Scan(userID); err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	if comment.UserID != userUUID {
		// Allow project owners to delete any comment in their project
		var hasAccess bool
		if comment.IssueID.Valid {
			issue, err := s.queries.GetIssueByID(ctx, comment.IssueID)
			if err == nil {
				// Check if user is project owner
				project, err := s.queries.GetProjectByID(ctx, issue.ProjectID)
				if err == nil && project.OwnerID == userUUID {
					hasAccess = true
				}
			}
		} else if comment.TaskID.Valid {
			task, err := s.queries.GetTaskByID(ctx, comment.TaskID)
			if err == nil {
				// Check if user is project owner
				project, err := s.queries.GetProjectByID(ctx, task.ProjectID)
				if err == nil && project.OwnerID == userUUID {
					hasAccess = true
				}
			}
		}

		if !hasAccess {
			return ErrNotCommentAuthor
		}
	}
	// Delete the comment
	if err := s.queries.DeleteComment(ctx, commentUUID); err != nil {
		return fmt.Errorf("failed to delete comment: %w", err)
	}

	// Invalidate comments list cache
	if comment.IssueID.Valid {
		s.invalidateCommentsCache(ctx, "issue", comment.IssueID.String())
	} else if comment.TaskID.Valid {
		s.invalidateCommentsCache(ctx, "task", comment.TaskID.String())
	}

	return nil
}

// Helper method to invalidate comments cache
func (s *CommentService) invalidateCommentsCache(_ context.Context, entityType string, entityID string) {
	if s.cache == nil {
		return
	}

	cacheKey := fmt.Sprintf("%s:%s:comments", entityType, entityID)
	if err := s.cache.Del(context.Background(), cacheKey).Err(); err != nil {
		log.Printf("Failed to invalidate comments cache: %v", err)
	}
}

// Helper method to verify access to the entity being commented on
func (s *CommentService) verifyCommentableAccess(ctx context.Context, issueID, taskID pgtype.UUID, userID string) error {
	// Verify that exactly one of issueID or taskID is provided
	if (issueID.Valid && taskID.Valid) || (!issueID.Valid && !taskID.Valid) {
		return fmt.Errorf("%w: exactly one of issue ID or task ID must be provided", ErrInvalidCommentData)
	}

	if issueID.Valid {
		// Get the issue and verify access
		issue, err := s.queries.GetIssueByID(ctx, issueID)
		if err != nil {
			return fmt.Errorf("failed to get issue: %w", err)
		}

		// Check access to the project this issue belongs to
		return s.projectService.verifyProjectAccess(ctx, &store.Project{ID: issue.ProjectID}, userID)
	} else {
		// Get the task and verify access
		task, err := s.queries.GetTaskByID(ctx, taskID)
		if err != nil {
			return fmt.Errorf("failed to get task: %w", err)
		}

		// Check access to the project this task belongs to
		return s.projectService.verifyProjectAccess(ctx, &store.Project{ID: task.ProjectID}, userID)
	}
}
