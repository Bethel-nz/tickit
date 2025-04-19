
# Tickit

A modern task and project management API built with Go, PostgreSQL, and Redis. Features user authentication, team collaboration.

## Features

-   User authentication and authorization
-   Project and task management
-   Team collaboration
-   Real-time updates via Redis
-   PostgreSQL for persistent storage
-   Docker containerization
-   Hot reload development environment
-   Custom HTTP Router

## Prerequisites

-   Go 1.24+
-   Docker and Docker Compose
-   SQLC
-   Make (optional)

## Getting Started

### 1. Generate Database Code

First, generate the Go code from SQL definitions:

```bash
# Install sqlc if not already installed
go install [github.com/sqlc-dev/sqlc/cmd/sqlc@latest](https://github.com/sqlc-dev/sqlc/cmd/sqlc@latest)

# Generate code
sqlc generate
````

### 2\. Environment Setup

reference [sample env](https://www.google.com/search?q=./.env.sample)

### 3\. Docker Setup

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

### 4\. Available Services

  - API: `http://localhost:8080`
  - PostgreSQL: `localhost:5432`
  - Redis: `localhost:6379`

## Development

For local development with hot reload:

```bash
# Install Air for hot reload
go install [github.com/cosmtrek/air@v1.16.1](https://github.com/cosmtrek/air@v1.16.1)

# Run with Air
air
```

## License

MIT

-----

## Further Reading / Deep Dives

  - **Building the HTTP Router:** A core component of the Tickit API is its custom-built HTTP router. This decision was part of a detailed exploration into Go's `net/http` and routing principles. I wrote a 3-part blog series detailing the motivation, implementation details, challenges faced, and lessons learned from building this component: [Building a Go HTTP Router (Part 1): Why & The Stdlib Showdown](https://not-bethel.vercel.app/logs/building-a-go-http-router-part-1-why-the-stdlib-showdown)
