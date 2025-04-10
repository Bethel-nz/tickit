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

reference [sample env](./.env.sample)

### 3. Docker Setup

The project includes two Dockerfiles for different environments:

- `Dockerfile` - Optimized for production, uses the pre-built binary in `bin/`.
- `Dockerfile.dev` - Development version with Air for hot reload.

#### Running the Application

To run the application in development mode with hot reload:

```bash
# Use docker-compose.dev.yml
docker compose up -d
```

To run in production mode:

Change Dockerfile in docker-compose.yml to just Dockerfile

```bash
# Use docker-compose.yml
docker compose up -d
```

### 4. Available Services

- API: `http://localhost:8080`
- PostgreSQL: `localhost:5432`
- Redis: `localhost:6379`

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
