#!/bin/bash
set -e

echo "Waiting for PostgreSQL to be ready..."
while ! pg_isready -h db -U admin -d tickit; do
  sleep 1
done
echo "PostgreSQL is ready!"

echo "Running database migrations..."
cd /app && go run scripts/migrations/init_db.go
echo "Database initialized!" 