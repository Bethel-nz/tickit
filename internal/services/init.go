package services

import (
	"github.com/Bethel-nz/tickit/internal/database/store"
	"github.com/Bethel-nz/tickit/internal/email"
	"github.com/go-redis/redis/v8"
)

// Services holds all the service instances
type Services struct {
	UserService    *UserService
	ProjectService *ProjectService
	IssueService   *IssueService
	CommentService *CommentService
	SearchService  *SearchService
	TeamService    *TeamService
}

// InitServices initializes all services with their dependencies
func InitServices(queries *store.Queries, cache *redis.Client, emailService *email.EmailService) *Services {
	// Initialize team service first as it's a dependency for project service
	teamService := NewTeamService(queries, cache)

	// Initialize project service with team service dependency
	projectService := NewProjectService(queries, cache, teamService)

	// Initialize issue service with project service dependency
	issueService := NewIssueService(queries, cache, projectService)

	// Initialize comment service with project service dependency
	commentService := NewCommentService(queries, cache, projectService)

	// Initialize search service
	searchService := NewSearchService(queries, cache)

	// Initialize user service
	userService := NewUserService(queries, cache, emailService)

	return &Services{
		UserService:    userService,
		ProjectService: projectService,
		IssueService:   issueService,
		CommentService: commentService,
		SearchService:  searchService,
		TeamService:    teamService,
	}
}
