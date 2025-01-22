BINARY_NAME=liam-bot

ENV_FILE=.env

check-env:
	@if [ ! -f $(ENV_FILE) ]; then \
		echo "Error: $(ENV_FILE) not found. Please create it."; \
		exit 1; \
	fi

DB_URL := $(shell grep '^DB_URL' $(ENV_FILE) | cut -d '=' -f2)

build:
	@echo "Building $(BINARY_NAME)..."
	go build -o $(BINARY_NAME) main.go
	@echo "Build complete."

run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_NAME)

migration-create: check-env
	@if [ -z "$(name)" ]; then \
		echo "Error: Please provide a migration name, e.g. 'make migration-create name=add_users_table'"; \
		exit 1; \
	fi
	@echo "Creating migration '$(name)'..."
	goose -dir db/migrations create $(name) sql
	@echo "Migration '$(name)' created."

migrate-up: check-env
	@echo "Applying migrations..."
	goose -dir db/migrations postgres "$(DB_URL)" up
	@echo "Migrations applied."

migrate-down: check-env
	@echo "Reverting migrations..."
	goose -dir db/migrations postgres "$(DB_URL)" down
	@echo "Migrations reverted."

migrate-status: check-env
	@echo "Checking migration status..."
	goose -dir db/migrations postgres "$(DB_URL)" status

clean:
	@echo "Cleaning up..."
	rm -f $(BINARY_NAME)
	@echo "Cleanup complete."

update-deps:
	@echo "Updating dependencies..."
	go mod tidy
	@echo "Dependencies updated."

help:
	@echo "Available commands:"
	@echo "  make build            - Build the project"
	@echo "  make run              - Run the project"
	@echo "  make migration-create name=<name> - Create a new migration"
	@echo "  make migrate-up       - Apply all migrations"
	@echo "  make migrate-down     - Revert migrations"
	@echo "  make migrate-status   - Check migration status"
	@echo "  make clean            - Clean up build files"
	@echo "  make update-deps      - Update Go dependencies"

.PHONY: build run migration-create migrate-up migrate-down migrate-status clean update-deps help check-env
