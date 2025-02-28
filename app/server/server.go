package server

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/Bethel-nz/tickit/app/router"
	"github.com/Bethel-nz/tickit/internal/database/store"
	"github.com/Bethel-nz/tickit/internal/types"
	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Application holds application-wide dependencies and configuration.
type Application struct {
	Config           *types.AppConfig
	Mux              *http.ServeMux
	DB               *pgxpool.Pool
	Store            *store.Queries
	Cache            *redis.Client
	GlobalMiddleware []func(http.Handler) http.Handler
}

// NewApplication creates a new instance of Application with default middleware.
func NewApplication() *Application {
	return &Application{
		GlobalMiddleware: make([]func(http.Handler) http.Handler, 0),
	}
}

// WithConfig initializes the application with the unified configuration.
// It creates the PGX pool, instantiates the store, and sets up Redis.
func (app *Application) WithConfig(cfg *types.AppConfig) *Application {
	app.Config = cfg

	// Create PGX pool using the DSN and configuration from AppConfig
	ctx := context.Background()
	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Unable to parse DSN: %v", err)
	}

	poolConfig.MaxConns = int32(cfg.MaxOpenConns)
	poolConfig.MaxConnIdleTime = cfg.MaxIdleTime

	pgxPool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		log.Fatalf("Unable to create PGX pool: %v", err)
	}

	app.DB = pgxPool
	app.Store = store.New(pgxPool)

	return app
}

// WithCache initializes the Redis client using the RedisURL from AppConfig.
func (app *Application) WithCache() *Application {
	app.Cache = redis.NewClient(&redis.Options{
		Addr: app.Config.RedisURL,
	})
	return app
}

// Use appends global middleware to the application.
func (app *Application) Use(middleware ...func(http.Handler) http.Handler) *Application {
	app.GlobalMiddleware = append(app.GlobalMiddleware, middleware...)
	return app
}

// WithMux registers application routes defined in a RouterGroup.
func (app *Application) WithMux(routes *router.RouterGroup) *Application {
	// Use the ServeMux from the router package
	app.Mux = router.ServeMux(routes)

	// Wrap the ServeMux with global middleware
	handler := http.Handler(app.Mux)
	for i := len(app.GlobalMiddleware) - 1; i >= 0; i-- {
		handler = app.GlobalMiddleware[i](handler)
	}

	// Update the Mux to the wrapped handler
	app.Mux = http.NewServeMux()
	app.Mux.Handle("/", handler)

	return app
}

// healthCheckHandler responds with a simple "OK" message.
func (app *Application) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// Serve starts the HTTP server and gracefully shuts it down on interrupt signals.
func (app *Application) Serve() error {
	server := &http.Server{
		Addr:         ":" + strconv.Itoa(app.Config.AppPort),
		Handler:      app.Mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	errChan := make(chan error, 1)
	go func() {
		log.Printf("Server starting on http://localhost:%d", app.Config.AppPort)
		errChan <- server.ListenAndServe()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	select {
	case err := <-errChan:
		return err
	case sig := <-quit:
		log.Printf("Received signal %v. Initiating graceful shutdown...", sig)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := server.Shutdown(ctx)
	if err != nil {
		log.Printf("Graceful shutdown failed: %v", err)
	} else {
		log.Println("Shutdown completed")
	}

	if app.DB != nil {
		app.DB.Close()
	}

	if app.Cache != nil {
		if err := app.Cache.Close(); err != nil {
			log.Printf("Error closing cache: %v", err)
		}
	}

	return nil
}
