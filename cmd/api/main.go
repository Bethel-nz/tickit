package main

import (
	"log"

	"github.com/Bethel-nz/tickit/app/middleware"
	"github.com/Bethel-nz/tickit/app/router"
	"github.com/Bethel-nz/tickit/app/server"
	"github.com/Bethel-nz/tickit/handlers"
	"github.com/Bethel-nz/tickit/internal/config"
	"github.com/Bethel-nz/tickit/internal/services"
)

func main() {
	// Load the unified configuration
	appConfig := config.LoadConfig()

	// Initialize the application with config, cache, and global middleware
	app := server.NewApplication().
		WithConfig(appConfig).
		WithCache().
		Use(middleware.LoggerMiddleware, middleware.RecovererMiddleware, middleware.CorsMiddleware)

	// Initialize services and capture the result
	svcs := services.InitServices(app.Store, app.Cache, nil) // Email service is nil for now

	// Initialize handlers with the services struct
	handlers.Init(svcs)

	// Create router group and set up routes
	routes := router.NewRouter()
	setupMainRoutes(routes, app.Store)

	// Register routes with the application
	app.WithMux(routes)

	// Start the server
	if err := app.Serve(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
