.PHONY: all build run test clean up down migrate migrate-down sqlc swag lint

# Variables
APP_NAME=go-stable
API_BINARY=cmd/api/main.go
WORKER_BINARY=cmd/worker/main.go
DB_DSN=mysql://app:apppassword@tcp(localhost:3306)/go_stable
MIGRATIONS_PATH=db/migrations

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
	migrate -path $(MIGRATIONS_PATH) -database "$(DB_DSN)" up

migrate-down:
	@echo "Rolling back migrations..."
	migrate -path $(MIGRATIONS_PATH) -database "$(DB_DSN)" down 1

migrate-force:
	@echo "Force setting migration version..."
	migrate -path $(MIGRATIONS_PATH) -database "$(DB_DSN)" force $(VERSION)

migrate-create:
	@echo "Creating new migration..."
	migrate create -ext sql -dir $(MIGRATIONS_PATH) -seq $(NAME)

# sqlc
sqlc:
	sqlc generate

sqlc-verify:
	sqlc verify

# Swagger
swag:
	swag init -g cmd/api/main.go -o docs
	@echo "Swagger docs generated at docs/"

# Lint
lint:
	golangci-lint run ./...

# Clean
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Development helpers
dev: up migrate run

# Generate all (sqlc + swagger)
generate: sqlc swag
	@echo "All code generated"

# Install tools
tools:
	go install -tags 'mysql' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/swaggo/swag/cmd/swag@v1.16.3
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Health check
health:
	curl -s http://localhost:8080/health | jq .

ready:
	curl -s http://localhost:8080/ready | jq .

swagger:
	@echo "Swagger UI: http://localhost:8080/swagger/index.html"

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
	@echo "  swag          - Generate Swagger docs"
	@echo "  generate      - Generate all (sqlc + swagger)"
	@echo "  lint          - Run linter"
	@echo "  dev           - Start all services and run API"
	@echo "  tools         - Install development tools"
	@echo "  health        - Check health endpoint"
	@echo "  ready         - Check ready endpoint"
	@echo "  swagger       - Show Swagger UI URL"
