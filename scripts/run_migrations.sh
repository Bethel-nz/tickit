#!/bin/bash
set -e

# Run migrations with sensible defaults
# The script will use environment variables or defaults from env package
echo "Running database migrations..."
go run scripts/migrations/init_db.go

echo "Migrations completed!" 