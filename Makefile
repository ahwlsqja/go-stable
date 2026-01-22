.PHONY: all build run test clean up down migrate migrate-down sqlc lint

# Variables
APP_NAME=go-stable
API_BINARY=cmd/api/main.go
WORKER_BINARY=cmd/worker/main.go
DB_DSN=mysql://app:apppassword@tcp(localhost:3306)/go_stable

# Build
all: build

build:
	go build -o bin/api $(API_BINARY)
	@echo "Build complete: bin/api"

build-worker:
	go build -o bin/worker $(WORKER_BINARY)
	@echo "Build complete: bin/worker"

# Run
run:
	go run $(API_BINARY)

run-worker:
	go run $(WORKER_BINARY)

# Test
test:
	go test -v -race ./...

test-coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Docker Compose
up:
	docker-compose up -d
	@echo "Waiting for services to be ready..."
	@sleep 5

down:
	docker-compose down

down-v:
	docker-compose down -v

logs:
	docker-compose logs -f

# Database
migrate:
	@echo "Running migrations..."
	migrate -path migrations -database "$(DB_DSN)" up

migrate-down:
	@echo "Rolling back migrations..."
	migrate -path migrations -database "$(DB_DSN)" down 1

migrate-force:
	@echo "Force setting migration version..."
	migrate -path migrations -database "$(DB_DSN)" force $(VERSION)

migrate-create:
	@echo "Creating new migration..."
	migrate create -ext sql -dir migrations -seq $(NAME)

# sqlc
sqlc:
	sqlc generate

sqlc-verify:
	sqlc verify

# Lint
lint:
	golangci-lint run ./...

# Clean
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Development helpers
dev: up migrate run

# Install tools
tools:
	go install -tags 'mysql' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Health check
health:
	curl -s http://localhost:8080/health | jq .

ready:
	curl -s http://localhost:8080/ready | jq .

# Help
help:
	@echo "Available targets:"
	@echo "  build         - Build API binary"
	@echo "  run           - Run API server"
	@echo "  test          - Run tests"
	@echo "  up            - Start docker-compose services"
	@echo "  down          - Stop docker-compose services"
	@echo "  migrate       - Run database migrations"
	@echo "  migrate-down  - Rollback last migration"
	@echo "  sqlc          - Generate sqlc code"
	@echo "  lint          - Run linter"
	@echo "  dev           - Start all services and run API"
	@echo "  tools         - Install development tools"
	@echo "  health        - Check health endpoint"
	@echo "  ready         - Check ready endpoint"
