package server

import (
	"context"
	"crypto/tls"
	"fmt"
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
	tlsConfig        *tls.Config // New field for TLS configuration
}

// NewApplication creates a new instance of Application with default middleware.
func NewApplication() *Application {
	return &Application{
		GlobalMiddleware: make([]func(http.Handler) http.Handler, 0),
	}
}

// WithConfig initializes the application with the unified configuration.
func (app *Application) WithConfig(cfg *types.AppConfig) *Application {
	app.Config = cfg

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
	app.Mux = router.ServeMux(routes)

	handler := http.Handler(app.Mux)
	for i := len(app.GlobalMiddleware) - 1; i >= 0; i-- {
		handler = app.GlobalMiddleware[i](handler)
	}

	app.Mux = http.NewServeMux()
	app.Mux.Handle("/", handler)

	return app
}

// TLSServer represents an Application configured with TLS, restricting chaining to Serve.
type TLSServer struct {
	app *Application
}

// WithTLS configures the application to use TLS with the provided tls.Config.
// It returns a TLSServer, which can only chain with Serve.
func (app *Application) WithTLS(cfg *tls.Config) *TLSServer {
	if cfg == nil || len(cfg.Certificates) == 0 {
		log.Fatal("TLS configuration must include at least one certificate")
	}
	app.tlsConfig = cfg
	return &TLSServer{app: app}
}

// Serve starts the HTTP server and gracefully shuts it down on interrupt signals.
// When called on Application, it starts an HTTP server.
// When called on TLSServer, it starts an HTTPS server with TLS.
func (app *Application) Serve() error {
	server := &http.Server{
		Addr:         ":" + strconv.Itoa(app.Config.AppPort),
		Handler:      app.Mux,
		ReadTimeout:  app.Config.ServerReadTimeout,
		WriteTimeout: app.Config.ServerWriteTimeout,
	}

	// If tlsConfig is set, use it; otherwise, default to HTTP
	if app.tlsConfig != nil {
		server.TLSConfig = app.tlsConfig
	}

	errChan := make(chan error, 1)
	go func() {
		if app.tlsConfig != nil {
			log.Printf("Server starting with TLS on https://localhost:%d", app.Config.AppPort)
			// Since tlsConfig is provided, use ListenAndServeTLS with empty cert/key files
			// (assumes certificates are loaded in tlsConfig)
			errChan <- server.ListenAndServeTLS("", "")
		} else {
			log.Printf("Server starting on http://localhost:%d", app.Config.AppPort)
			errChan <- server.ListenAndServe()
		}
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

	var shutdownErr error

	if app.DB != nil {
		app.DB.Close()
	}

	if app.Cache != nil {
		if err := app.Cache.Close(); err != nil {
			shutdownErr = fmt.Errorf("cache close error: %v", err)
		}
	}

	return shutdownErr
}

// Serve for TLSServer ensures TLS is used (reuses Application's Serve logic).
func (ts *TLSServer) Serve() error {

	return ts.app.Serve()
}
