# Simple Makefile for a Go project
include .env
# connection string
POSTGRES_URL=postgresql://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable
MIGRATE_PATH=internal/server/database/migrations

# Build the application
all: build

server-build:
	@echo "Building..."
	
	@go build -o main cmd/server/main.go

peer-build:
	@echo "Building..."
	
	@go build -o main cmd/peer/main.go

# Run the application
server-run:
	@go run cmd/server/main.go

peer-run:
	@go run cmd/peer/main.go

# Create DB container
docker-run:
	@if docker compose up 2>/dev/null; then \
		: ; \
	else \
		echo "Falling back to Docker Compose V1"; \
		docker-compose up; \
	fi

# Shutdown DB container
docker-down:
	@if docker compose down 2>/dev/null; then \
		: ; \
	else \
		echo "Falling back to Docker Compose V1"; \
		docker-compose down; \
	fi

# Test the application
test:
	@echo "Testing..."
	@go test ./tests -v

# Clean the binary
clean:
	@echo "Cleaning..."
	@rm -f main

migrate-up:
	@if command -v migrate > /dev/null; then \
		echo "Migrating up..."; \
		migrate -path $(MIGRATE_PATH) -database $(POSTGRES_URL) -verbose up;\
	else \
		read -p "migrate is not installed. Do you want to install it? [Y/n] " choice; \
	    if [ "$$choice" != "n" ] && [ "$$choice" != "N" ]; then \
	        go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest; \
	        migrate -path $(MIGRATE_PATH) -database $(POSTGRES_URL) up; \
	    else \
	        echo "You chose not to install migrate. Exiting..."; \
	        exit 1; \
	    fi; \
	fi

migrate-down:
	@if command -v migrate > /dev/null; then \
		echo "Migrating down..."; \
		migrate -path $(MIGRATE_PATH) -database $(POSTGRES_URL) -verbose down; \
	else \
		read -p "migrate is not installed. Do you want to install it? [Y/n] " choice; \
	    if [ "$$choice" != "n" ] && [ "$$choice" != "N" ]; then \
	        go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest; \
	        migrate -path $(MIGRATE_PATH) -database $(POSTGRES_URL) down; \
	    else \
	        echo "You chose not to install migrate. Exiting..."; \
	        exit 1; \
	    fi; \
	fi

# Live Reload
watch:
	@if command -v air > /dev/null; then \
	    air; \
	    echo "Watching...";\
	else \
	    read -p "Go's 'air' is not installed on your machine. Do you want to install it? [Y/n] " choice; \
	    if [ "$$choice" != "n" ] && [ "$$choice" != "N" ]; then \
	        go install github.com/cosmtrek/air@latest; \
	        air; \
	        echo "Watching...";\
	    else \
	        echo "You chose not to install air. Exiting..."; \
	        exit 1; \
	    fi; \
	fi

.PHONY: all build run test clean
