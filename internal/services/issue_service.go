package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Bethel-nz/tickit/internal/database/store"
	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v5/pgtype"
)

// Issue service errors
var (
	ErrIssueNotFound    = errors.New("issue not found")
	ErrInvalidIssueData = errors.New("invalid issue data")
)

// IssueInfo represents issue information returned to clients
type IssueInfo struct {
	ID          string     `json:"id"`
	ProjectID   string     `json:"project_id"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	Status      string     `json:"status"`
	ReporterID  string     `json:"reporter_id"`
	AssigneeID  string     `json:"assignee_id,omitempty"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	CreatedAt   string     `json:"created_at"`
	UpdatedAt   string     `json:"updated_at,omitempty"`
}

// IssueUpdates contains fields that can be updated for an issue
type IssueUpdates struct {
	Title       string
	Description string
	Status      string
	AssigneeID  string
	DueDate     *time.Time
}

type IssueService struct {
	queries        *store.Queries
	cache          *redis.Client
	projectService *ProjectService
}

func NewIssueService(queries *store.Queries, cache *redis.Client, projectService *ProjectService) *IssueService {
	return &IssueService{
		queries:        queries,
		cache:          cache,
		projectService: projectService,
	}
}

// GetProjectIssues retrieves all issues for a project
func (s *IssueService) GetProjectIssues(ctx context.Context, projectID string, userID string) ([]IssueInfo, error) {
	// Verify project access
	_, err := s.projectService.GetProjectByID(ctx, projectID, userID)
	if err != nil {
		return nil, err
	}

	var projectUUID pgtype.UUID
	if err := projectUUID.Scan(projectID); err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	issues, err := s.queries.GetProjectIssues(ctx, projectUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project issues: %w", err)
	}

	result := make([]IssueInfo, 0, len(issues))
	for _, issue := range issues {
		info := IssueInfo{
			ID:          issue.ID.String(),
			ProjectID:   issue.ProjectID.String(),
			Title:       issue.Title,
			Description: issue.Description.String,
			Status:      issue.Status.String,
			ReporterID:  issue.ReporterID.String(),
			CreatedAt:   issue.CreatedAt.Time.Format(time.RFC3339),
			UpdatedAt:   issue.UpdatedAt.Time.Format(time.RFC3339),
		}

		if issue.AssigneeID.Valid {
			info.AssigneeID = issue.AssigneeID.String()
		}

		if issue.DueDate.Valid {
			dueDate := issue.DueDate.Time
			info.DueDate = &dueDate
		}

		result = append(result, info)
	}

	return result, nil
}

// GetIssuesByStatus retrieves issues with a specific status for a project
func (s *IssueService) GetIssuesByStatus(ctx context.Context, projectID, status, userID string) ([]IssueInfo, error) {
	// Verify project access
	_, err := s.projectService.GetProjectByID(ctx, projectID, userID)
	if err != nil {
		return nil, err
	}

	var projectUUID pgtype.UUID
	if err := projectUUID.Scan(projectID); err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	var statusText pgtype.Text
	if err := statusText.Scan(status); err != nil {
		return nil, fmt.Errorf("invalid status: %w", err)
	}

	issues, err := s.queries.GetIssuesByStatus(ctx, store.GetIssuesByStatusParams{
		ProjectID: projectUUID,
		Status:    statusText,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get issues by status: %w", err)
	}

	result := make([]IssueInfo, 0, len(issues))
	for _, issue := range issues {
		info := IssueInfo{
			ID:          issue.ID.String(),
			ProjectID:   issue.ProjectID.String(),
			Title:       issue.Title,
			Description: issue.Description.String,
			Status:      status,
			ReporterID:  issue.ReporterID.String(),
			CreatedAt:   issue.CreatedAt.Time.Format(time.RFC3339),
			UpdatedAt:   issue.UpdatedAt.Time.Format(time.RFC3339),
		}

		if issue.AssigneeID.Valid {
			info.AssigneeID = issue.AssigneeID.String()
		}

		if issue.DueDate.Valid {
			dueDate := issue.DueDate.Time
			info.DueDate = &dueDate
		}

		result = append(result, info)
	}

	return result, nil
}

// CreateIssue creates a new issue
func (s *IssueService) CreateIssue(ctx context.Context, params store.CreateIssueParams, userID string) (*IssueInfo, error) {
	// Verify project access
	_, err := s.projectService.GetProjectByID(ctx, params.ProjectID.String(), userID)
	if err != nil {
		return nil, err
	}

	issue, err := s.queries.CreateIssue(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create issue: %w", err)
	}

	info := issueToInfo(issue)
	return &info, nil
}

// GetIssueByID retrieves a specific issue
func (s *IssueService) GetIssueByID(ctx context.Context, issueID, userID string) (*IssueInfo, error) {
	var issueUUID pgtype.UUID
	if err := issueUUID.Scan(issueID); err != nil {
		return nil, fmt.Errorf("invalid issue ID: %w", err)
	}

	issue, err := s.queries.GetIssueByID(ctx, issueUUID)
	if err != nil {
		return nil, ErrIssueNotFound
	}

	// Verify project access
	_, err = s.projectService.GetProjectByID(ctx, issue.ProjectID.String(), userID)
	if err != nil {
		return nil, err
	}

	info := issueToInfo(issue)
	return &info, nil
}

// UpdateIssue updates an issue
func (s *IssueService) UpdateIssue(ctx context.Context, issueID string, updates IssueUpdates, userID string) error {
	var issueUUID pgtype.UUID
	if err := issueUUID.Scan(issueID); err != nil {
		return fmt.Errorf("invalid issue ID: %w", err)
	}

	// Get the issue to verify project access
	issue, err := s.queries.GetIssueByID(ctx, issueUUID)
	if err != nil {
		return ErrIssueNotFound
	}

	// Verify project access
	_, err = s.projectService.GetProjectByID(ctx, issue.ProjectID.String(), userID)
	if err != nil {
		return err
	}

	// Prepare update parameters
	params := store.UpdateIssueDetailsParams{
		ID: issueUUID,
	}

	if updates.Title != "" {
		params.Title = updates.Title
	}

	if updates.Description != "" {
		params.Description = pgtype.Text{String: updates.Description, Valid: true}
	}

	if updates.Status != "" {
		params.Status = pgtype.Text{String: updates.Status, Valid: true}
	}

	if updates.AssigneeID != "" {
		var assigneeUUID pgtype.UUID
		if err := assigneeUUID.Scan(updates.AssigneeID); err != nil {
			return fmt.Errorf("invalid assignee ID: %w", err)
		}
		params.AssigneeID = assigneeUUID
	}

	if updates.DueDate != nil {
		params.DueDate = pgtype.Timestamp{Time: *updates.DueDate, Valid: true}
	}

	if err := s.queries.UpdateIssueDetails(ctx, params); err != nil {
		return fmt.Errorf("failed to update issue: %w", err)
	}

	return nil
}

// DeleteIssue deletes an issue
func (s *IssueService) DeleteIssue(ctx context.Context, issueID, userID string) error {
	var issueUUID pgtype.UUID
	if err := issueUUID.Scan(issueID); err != nil {
		return fmt.Errorf("invalid issue ID: %w", err)
	}

	// Get the issue to verify project access
	issue, err := s.queries.GetIssueByID(ctx, issueUUID)
	if err != nil {
		return ErrIssueNotFound
	}

	// Verify project access
	_, err = s.projectService.GetProjectByID(ctx, issue.ProjectID.String(), userID)
	if err != nil {
		return err
	}

	if err := s.queries.DeleteIssue(ctx, issueUUID); err != nil {
		return fmt.Errorf("failed to delete issue: %w", err)
	}

	return nil
}

// Helper function to convert issue to info
func issueToInfo(issue store.Issue) IssueInfo {
	info := IssueInfo{
		ID:          issue.ID.String(),
		ProjectID:   issue.ProjectID.String(),
		Title:       issue.Title,
		Description: issue.Description.String,
		Status:      issue.Status.String,
		ReporterID:  issue.ReporterID.String(),
		CreatedAt:   issue.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:   issue.UpdatedAt.Time.Format(time.RFC3339),
	}

	if issue.AssigneeID.Valid {
		info.AssigneeID = issue.AssigneeID.String()
	}

	if issue.DueDate.Valid {
		dueDate := issue.DueDate.Time
		info.DueDate = &dueDate
	}

	return info
}
