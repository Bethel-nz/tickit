package types

import "time"

// AppConfig holds application configuration values.
type AppConfig struct {
	DatabaseURL    string        // PostgreSQL connection string
	AppPort        int           // Port to listen on
	DebugMode      bool          // Enable debug mode
	RequestTimeout time.Duration // Timeout for requests
	Threshold      float64       // Threshold value
	RedisURL       string        // Redis connection URL
	MaxOpenConns   int           // Maximum open database connections
	MaxIdleTime    time.Duration // Maximum idle time for database connections
}
