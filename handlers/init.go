package handlers

import "github.com/Bethel-nz/tickit/internal/services"

// Init initializes all handler services from the provided Services struct
func Init(s *services.Services) {
	SetUserService(s.UserService)
	SetProjectService(s.ProjectService)
	SetIssueService(s.IssueService)
	SetCommentService(s.CommentService)
	SetSearchService(s.SearchService)
	SetTeamService(s.TeamService)
}
