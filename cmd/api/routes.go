package main

import (
	"github.com/Bethel-nz/tickit/app/middleware"
	"github.com/Bethel-nz/tickit/app/router"
	"github.com/Bethel-nz/tickit/handlers"
	"github.com/Bethel-nz/tickit/internal/database/store"
)

// setupRoutes configures all application routes
func setupRoutes(r *router.RouterGroup, queries *store.Queries) {
	ownershipMiddleware := middleware.NewOwnershipMiddleware(queries)

	// User routes
	users := r.Group("/users")

	// Public endpoints
	users.POST("/register", handlers.RegisterUser)
	users.POST("/login", handlers.LoginUser)
	users.POST("/forgot-password", handlers.ForgotPassword)
	users.POST("/reset-password/{token}", handlers.ResetPassword)

	// Protected endpoints requiring authentication
	authenticated := users.Group("", middleware.AuthMiddleware)
	authenticated.GET("/me", handlers.GetUserProfile)
	authenticated.PUT("/me", handlers.UpdateUserProfile)
	authenticated.POST("/change-password", handlers.ChangePassword)
	authenticated.DELETE("/me", handlers.DeleteAccount)

	// Search route - accessible to authenticated users
	r.GET("/search", handlers.SearchEntities, middleware.AuthMiddleware)

	// Project routes
	projects := r.Group("/projects", middleware.AuthMiddleware)
	projects.GET("/", handlers.ListProjects)
	projects.POST("/", handlers.CreateProject)
	projects.GET("/{id}", handlers.GetProject)
	projects.PUT("/{id}", handlers.UpdateProject, ownershipMiddleware)
	projects.DELETE("/{id}", handlers.DeleteProject, ownershipMiddleware)

	// Ticket routes
	tickets := projects.Group("/{project_id}/tickets")
	tickets.GET("/", handlers.ListTickets)
	tickets.POST("/", handlers.CreateTicket)
	tickets.GET("/{id}", handlers.GetTicket)
	tickets.PUT("/{id}", handlers.UpdateTicket)
	tickets.DELETE("/{id}", handlers.DeleteTicket)
	tickets.POST("/{id}/assign", handlers.AssignTicket)

	// Comments under tickets (issues)
	comments := tickets.Group("/{ticket_id}/comments")
	comments.GET("/", handlers.ListComments)
	comments.POST("/", handlers.CreateComment)
	comments.PUT("/{id}", handlers.UpdateComment)    // Ownership handled by service
	comments.DELETE("/{id}", handlers.DeleteComment) // Ownership handled by service

	// Optional: If you have a separate tasks endpoint
	tasks := projects.Group("/{project_id}/tasks")
	tasks.GET("/{task_id}/comments", handlers.ListComments)
	tasks.POST("/{task_id}/comments", handlers.CreateComment)
}

// setupMainRoutes configures main application routes
func setupMainRoutes(r *router.RouterGroup, queries *store.Queries) {
	setupRoutes(r, queries)

	// Add health check endpoint
	r.GET("/health", handlers.HealthCheck)
}
