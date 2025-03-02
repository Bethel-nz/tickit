.PHONY: migrate migrate-create migrate-up migrate-down

# Run migrations
migrate:
	go run scripts/migrations/init_db.go

# Create a new migration
migrate-create:
	@read -p "Enter migration name: " name; \
	migrate create -ext sql -dir internal/database/migrations -seq $$name

# Apply migrations up
migrate-up:
	go run scripts/migrations/init_db.go

# Revert migrations
migrate-down:
	go run scripts/migrations/init_db.go -down 