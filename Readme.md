# Tickit

A modern task and project management API built with Go, PostgreSQL, and Redis. Features user authentication, team collaboration.

## Features

- User authentication and authorization
- Project and task management
- Team collaboration
- Real-time updates via Redis
- PostgreSQL for persistent storage
- Docker containerization
- Hot reload development environment

## Prerequisites

- Go 1.24+
- Docker and Docker Compose
- SQLC
- Make (optional)

## Getting Started

### 1. Generate Database Code

First, generate the Go code from SQL definitions:

```bash
# Install sqlc if not already installed
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# Generate code
sqlc generate
```

### 2. Environment Setup

Create a `.envrc` file in the project root using the sample provided below:

```env
# PostgreSQL database connection URL
export DATABASE_URL="postgres://user:password@localhost:5432/yourdb?sslmode=disable"

# Port the application listens on
export APP_PORT="8080"

# Enable or disable debug mode
export DEBUG_MODE="false"

# Request timeout duration (e.g., 5 seconds)
export REQUEST_TIMEOUT="5s"

# Threshold value (e.g., 0.75)
export THRESHOLD="0.75"

# Redis connection URL
export REDIS_URL="localhost:6379"

# Maximum open database connections
export MAX_OPEN_CONNS="25"

# Maximum idle time for database connections (e.g., 5 minutes)
export MAX_IDLE_TIME="5m"
```

To automatically load environment variables when entering the project directory, install(https://direnv.net/).

### 3. Docker Setup

The project includes two Dockerfiles for different environments:

- `Dockerfile` - Optimized for production, uses the pre-built binary in `bin/`.
- `Dockerfile.dev` - Development version with Air for hot reload.


#### Running the Application

To run the application in development mode with hot reload:

```bash
# Use docker-compose.dev.yml
docker compose -f docker-compose.dev.yml up -d
```

To run in production mode:

```bash
# Use docker-compose.yml
docker compose -f docker-compose.yml up -d
```

### 4. Available Services

- API: `http://localhost:8080`
- PostgreSQL: `localhost:5432`
- Redis: `localhost:6379`
- Redis Commander: `http://localhost:8081`

## Development

For local development with hot reload:

```bash
# Install Air for hot reload
go install github.com/cosmtrek/air@v1.16.1

# Run with Air
air
```

## License

MIT
