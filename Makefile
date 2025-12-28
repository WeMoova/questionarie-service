.PHONY: build run test clean docker-build docker-run

# Build the application
build:
	go build -o main .

# Run the application
run:
	go run .

# Run tests
test:
	go test -v -race -coverprofile=coverage.out ./...

# Clean build artifacts
clean:
	rm -f main
	rm -f coverage.out

# Build Docker image
docker-build:
	docker build -t questionarie-service:latest .

# Run Docker container
docker-run:
	docker run -p 8080:8080 --env-file .env questionarie-service:latest

# Install dependencies
deps:
	go mod download
	go mod tidy

# Create a new migration
migrate-create:
	@read -p "Enter migration name: " name; \
	goose -dir migrations create $$name sql

# Run migrations
migrate-up:
	goose -dir migrations postgres "postgresql://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=require&search_path=$(DB_SCHEMA)" up

# Rollback last migration
migrate-down:
	goose -dir migrations postgres "postgresql://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=require&search_path=$(DB_SCHEMA)" down
