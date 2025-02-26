package main

import (
	"log"

	"github.com/Bethel-nz/tickit/app/server"
	"github.com/Bethel-nz/tickit/internal/config"
)

func main() {
	// Load the unified configuration
	appConfig := config.LoadConfig()

	app := server.NewApplication().
		WithConfig(appConfig).
		WithCache()

	// routes := NewRouterGroup()
	// app.WithMux(routes)

	// Start the server
	if err := app.Serve(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
