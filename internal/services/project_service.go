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

// Project service errors
var (
	ErrProjectNotFound    = errors.New("project not found")
	ErrInvalidProjectData = errors.New("invalid project data")
	ErrNotProjectOwner    = errors.New("user is not the project owner")
	ErrNotTeamProject     = errors.New("project is not associated with this team")
)

// ProjectStats represents project statistics
type ProjectStats struct {
	TotalIssues      int `json:"total_issues"`
	OpenIssues       int `json:"open_issues"`
	InProgressIssues int `json:"in_progress_issues"`
	ClosedIssues     int `json:"closed_issues"`
	TotalTasks       int `json:"total_tasks"`
	TodoTasks        int `json:"todo_tasks"`
	InProgressTasks  int `json:"in_progress_tasks"`
	DoneTasks        int `json:"done_tasks"`
}

// ProjectInfo represents project information returned to clients
type ProjectInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	OwnerID     string `json:"owner_id"`
	TeamID      string `json:"team_id,omitempty"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at,omitempty"`
}

type ProjectService struct {
	queries     *store.Queries
	cache       *redis.Client
	teamService *TeamService
}

func NewProjectService(queries *store.Queries, cache *redis.Client, teamService *TeamService) *ProjectService {
	return &ProjectService{
		queries:     queries,
		cache:       cache,
		teamService: teamService,
	}
}

// CreateProject creates a new project with the provided information
func (s *ProjectService) CreateProject(ctx context.Context, params store.CreateProjectParams) (*store.Project, error) {
	if params.Name == "" {
		return nil, fmt.Errorf("%w: project name is required", ErrInvalidProjectData)
	}

	if params.TeamID.Valid {
		isMember, err := s.teamService.CheckTeamMembership(ctx, params.TeamID.String(), params.OwnerID.String())
		if err != nil {
			return nil, fmt.Errorf("failed to check team membership: %w", err)
		}
		if !isMember {
			return nil, fmt.Errorf("%w: user is not a member of the specified team", ErrInvalidProjectData)
		}
	}

	// Create project in database
	project, err := s.queries.CreateProject(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	// Cache project data
	s.cacheProject(ctx, &project)

	return &project, nil
}

func (s *ProjectService) GetProjectByID(ctx context.Context, projectID string, userID string) (*store.Project, error) {
	var projectUUID pgtype.UUID
	if err := projectUUID.Scan(projectID); err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	cacheKey := fmt.Sprintf("project:%s", projectID)
	cachedProject, err := s.cache.Get(ctx, cacheKey).Result()
	if err == nil {
		var project store.Project
		if err := json.Unmarshal([]byte(cachedProject), &project); err == nil {

			if err := s.verifyProjectAccess(ctx, &project, userID); err != nil {
				return nil, err
			}
			return &project, nil
		}
	}

	project, err := s.queries.GetProjectByID(ctx, projectUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	if err := s.verifyProjectAccess(ctx, &project, userID); err != nil {
		return nil, err
	}

	s.cacheProject(ctx, &project)

	return &project, nil
}

// GetUserProjects retrieves all projects owned by or accessible to a user
func (s *ProjectService) GetUserProjects(ctx context.Context, userID string) ([]ProjectInfo, error) {
	var userUUID pgtype.UUID
	if err := userUUID.Scan(userID); err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	cacheKey := fmt.Sprintf("user:%s:projects", userID)
	cachedProjects, err := s.cache.Get(ctx, cacheKey).Result()
	if err == nil {
		var projects []ProjectInfo
		if err := json.Unmarshal([]byte(cachedProjects), &projects); err == nil {
			return projects, nil
		}
	}

	// Get from database
	dbProjects, err := s.queries.GetUserProjects(ctx, userUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user projects: %w", err)
	}

	projects := make([]ProjectInfo, len(dbProjects))
	for i, p := range dbProjects {
		projects[i] = ProjectInfo{
			ID:          p.ID.String(),
			Name:        p.Name,
			Description: p.Description.String,
			OwnerID:     p.OwnerID.String(),
			TeamID:      p.TeamID.String(),
			Status:      p.Status.String,
			CreatedAt:   p.CreatedAt.Time.Format(time.RFC3339),
			UpdatedAt:   p.UpdatedAt.Time.Format(time.RFC3339),
		}
	}

	projectsJSON, err := json.Marshal(projects)
	if err == nil {
		if err := s.cache.Set(ctx, cacheKey, projectsJSON, 10*time.Minute).Err(); err != nil {
			log.Printf("Failed to cache user projects: %v", err)
		}
	}

	return projects, nil
}

// GetTeamProjects retrieves all projects associated with a team
func (s *ProjectService) GetTeamProjects(ctx context.Context, teamID string, userID string) ([]ProjectInfo, error) {
	var teamUUID pgtype.UUID
	if err := teamUUID.Scan(teamID); err != nil {
		return nil, fmt.Errorf("invalid team ID: %w", err)
	}

	isMember, err := s.teamService.CheckTeamMembership(ctx, teamID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check team membership: %w", err)
	}
	if !isMember {
		return nil, ErrNotTeamMember
	}

	// Try to get from cache
	cacheKey := fmt.Sprintf("team:%s:projects", teamID)
	cachedProjects, err := s.cache.Get(ctx, cacheKey).Result()
	if err == nil {
		var projects []ProjectInfo
		if err := json.Unmarshal([]byte(cachedProjects), &projects); err == nil {
			return projects, nil
		}
	}

	// Get from database using a custom query
	// Note: You might need to add this query to your SQL file and regenerate the code
	dbProjects, err := s.queries.GetTeamProjects(ctx, teamUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get team projects: %w", err)
	}

	// Convert to our response format
	projects := make([]ProjectInfo, len(dbProjects))
	for i, p := range dbProjects {
		projects[i] = ProjectInfo{
			ID:          p.ID.String(),
			Name:        p.Name,
			Description: p.Description.String,
			OwnerID:     p.OwnerID.String(),
			TeamID:      p.TeamID.String(),
			Status:      p.Status.String,
			CreatedAt:   p.CreatedAt.Time.Format(time.RFC3339),
			UpdatedAt:   p.UpdatedAt.Time.Format(time.RFC3339),
		}
	}

	// Cache the result
	projectsJSON, err := json.Marshal(projects)
	if err == nil {
		if err := s.cache.Set(ctx, cacheKey, projectsJSON, 10*time.Minute).Err(); err != nil {
			log.Printf("Failed to cache team projects: %v", err)
		}
	}

	return projects, nil
}

// UpdateProject updates project information
func (s *ProjectService) UpdateProject(ctx context.Context, params store.UpdateProjectDetailsParams, userID string) error {
	// Get the current project to check ownership
	var projectUUID pgtype.UUID
	if err := projectUUID.Scan(params.ID); err != nil {
		return fmt.Errorf("invalid project ID: %w", err)
	}

	project, err := s.queries.GetProjectByID(ctx, projectUUID)
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}

	// Check if user has permission to update
	if err := s.verifyProjectOwnership(&project, userID); err != nil {
		return err
	}

	// If team is being changed, check membership
	if params.TeamID.Valid {
		isMember, err := s.teamService.CheckTeamMembership(ctx, params.TeamID.String(), userID)
		if err != nil {
			return fmt.Errorf("failed to check team membership: %w", err)
		}
		if !isMember {
			return fmt.Errorf("%w: user is not a member of the specified team", ErrInvalidProjectData)
		}
	}

	// Update project in database
	if err := s.queries.UpdateProjectDetails(ctx, params); err != nil {
		return fmt.Errorf("failed to update project: %w", err)
	}

	// Invalidate cache
	cacheKey := fmt.Sprintf("project:%s", params.ID)
	if err := s.cache.Del(ctx, cacheKey).Err(); err != nil {
		log.Printf("Failed to invalidate project cache: %v", err)
	}

	// Also invalidate user/team projects caches
	var userUUID pgtype.UUID
	if err := userUUID.Scan(userID); err == nil {
		userCacheKey := fmt.Sprintf("user:%s:projects", userID)
		s.cache.Del(ctx, userCacheKey)
	}

	if project.TeamID.Valid {
		teamCacheKey := fmt.Sprintf("team:%s:projects", project.TeamID.String())
		s.cache.Del(ctx, teamCacheKey)
	}

	return nil
}

// DeleteProject deletes a project
func (s *ProjectService) DeleteProject(ctx context.Context, projectID string, userID string) error {
	var projectUUID pgtype.UUID
	if err := projectUUID.Scan(projectID); err != nil {
		return fmt.Errorf("invalid project ID: %w", err)
	}

	// Get the project to check ownership
	project, err := s.queries.GetProjectByID(ctx, projectUUID)
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}

	// Check if user has permission to delete
	if err := s.verifyProjectOwnership(&project, userID); err != nil {
		return err
	}

	// Delete project from database
	if err := s.queries.DeleteProject(ctx, projectUUID); err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	// Invalidate cache
	cacheKey := fmt.Sprintf("project:%s", projectID)
	if err := s.cache.Del(ctx, cacheKey).Err(); err != nil {
		log.Printf("Failed to invalidate project cache: %v", err)
	}

	// Also invalidate user/team projects caches
	userCacheKey := fmt.Sprintf("user:%s:projects", userID)
	s.cache.Del(ctx, userCacheKey)

	if project.TeamID.Valid {
		teamCacheKey := fmt.Sprintf("team:%s:projects", project.TeamID.String())
		s.cache.Del(ctx, teamCacheKey)
	}

	return nil
}

// GetProjectStats retrieves statistics for a project
func (s *ProjectService) GetProjectStats(ctx context.Context, projectID string, userID string) (*ProjectStats, error) {
	var projectUUID pgtype.UUID
	if err := projectUUID.Scan(projectID); err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	// Get the project to check access
	project, err := s.queries.GetProjectByID(ctx, projectUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	// Verify access
	if err := s.verifyProjectAccess(ctx, &project, userID); err != nil {
		return nil, err
	}

	// Try to get from cache first
	cacheKey := fmt.Sprintf("project:%s:stats", projectID)
	cachedStats, err := s.cache.Get(ctx, cacheKey).Result()
	if err == nil {
		var stats ProjectStats
		if err := json.Unmarshal([]byte(cachedStats), &stats); err == nil {
			return &stats, nil
		}
	}

	// Get stats from database
	dbStats, err := s.queries.GetProjectStats(ctx, projectUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project stats: %w", err)
	}

	// Convert to our response type
	stats := &ProjectStats{
		TotalIssues:      int(dbStats.TotalIssues),
		OpenIssues:       int(dbStats.OpenIssues),
		InProgressIssues: int(dbStats.InProgressIssues),
		ClosedIssues:     int(dbStats.ClosedIssues),
		TotalTasks:       int(dbStats.TotalTasks),
		TodoTasks:        int(dbStats.TodoTasks),
		InProgressTasks:  int(dbStats.InProgressTasks),
		DoneTasks:        int(dbStats.DoneTasks),
	}

	// Cache for 5 minutes since stats change frequently
	statsJSON, err := json.Marshal(stats)
	if err == nil {
		if err := s.cache.Set(ctx, cacheKey, statsJSON, 5*time.Minute).Err(); err != nil {
			log.Printf("Failed to cache project stats: %v", err)
		}
	}

	return stats, nil
}

// Helper method to cache a project
func (s *ProjectService) cacheProject(ctx context.Context, project *store.Project) {
	if s.cache == nil {
		return
	}

	projectJSON, err := json.Marshal(project)
	if err != nil {
		log.Printf("Failed to marshal project: %v", err)
		return
	}

	cacheKey := fmt.Sprintf("project:%s", project.ID.String())
	if err := s.cache.Set(ctx, cacheKey, projectJSON, time.Hour).Err(); err != nil {
		log.Printf("Failed to cache project: %v", err)
	}
}

// verifyProjectOwnership checks if a user is the owner of a project
func (s *ProjectService) verifyProjectOwnership(project *store.Project, userID string) error {
	var userUUID pgtype.UUID
	if err := userUUID.Scan(userID); err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	if project.OwnerID != userUUID {
		return ErrNotProjectOwner
	}

	return nil
}

// verifyProjectAccess checks if a user has access to a project (owner or team member)
func (s *ProjectService) verifyProjectAccess(ctx context.Context, project *store.Project, userID string) error {
	var userUUID pgtype.UUID
	if err := userUUID.Scan(userID); err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	if project.OwnerID == userUUID {
		return nil
	}

	if !project.TeamID.Valid {
		return ErrNotProjectOwner
	}

	isMember, err := s.teamService.CheckTeamMembership(ctx, project.TeamID.String(), userID)
	if err != nil {
		return fmt.Errorf("failed to check team membership: %w", err)
	}
	if !isMember {
		return ErrNotTeamMember
	}

	return nil
}
