#!/bin/bash
set -e

echo "Starting initialization process..."
echo "Checking PostgreSQL connection..."

# Function to check PostgreSQL connection
check_postgres() {
    echo "Attempting to connect to PostgreSQL..."
    PGPASSWORD=adminpassword psql -h db -U admin -d tickit -c '\q' 2>&1
    return $?
}

# Try to connect to PostgreSQL with a timeout
timeout=60
start_time=$(date +%s)
while true; do
    if check_postgres; then
        echo "PostgreSQL is ready!"
        break
    else
        current_time=$(date +%s)
        elapsed=$((current_time - start_time))
        if [ $elapsed -ge $timeout ]; then
            echo "Timeout waiting for PostgreSQL after $timeout seconds"
            exit 1
        fi
        echo "PostgreSQL is not ready yet. Waiting... (${elapsed}s elapsed)"
        sleep 2
    fi
done

echo "Running database migrations..."
cd /app && go run scripts/migrations/init_db.go
echo "Database initialized!" 