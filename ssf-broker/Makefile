# SSF Broker Service Makefile

.PHONY: help build test clean run docker-build docker-run lint fmt vet deps

# Default target
help: ## Show this help message
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Build
build: ## Build the SSF broker binary
	@echo "Building SSF broker..."
	@go build -o bin/ssf-broker ./cmd/server

build-linux: ## Build for Linux
	@echo "Building SSF broker for Linux..."
	@GOOS=linux GOARCH=amd64 go build -o bin/ssf-broker-linux ./cmd/server

# Test
test: ## Run all tests
	@echo "Running tests..."
	@go test -v ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-race: ## Run tests with race detection
	@echo "Running tests with race detection..."
	@go test -v -race ./...

test-integration: ## Run integration tests (requires GCP setup)
	@echo "Running integration tests..."
	@go test -v -tags=integration ./...

# Code quality
lint: ## Run golangci-lint
	@echo "Running linter..."
	@golangci-lint run

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...

vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

# Run
run: ## Run the SSF broker locally
	@echo "Starting SSF broker..."
	@GCP_PROJECT_ID=test-project PUBSUB_TOPIC_PREFIX=ssf-events-test go run ./cmd/server

run-dev: ## Run with development settings
	@echo "Starting SSF broker in development mode..."
	@LOG_LEVEL=debug GCP_PROJECT_ID=test-project PUBSUB_TOPIC_PREFIX=ssf-events-dev go run ./cmd/server

# Docker
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t ssf-broker:latest .

docker-run: ## Run Docker container
	@echo "Running Docker container..."
	@docker run -p 8080:8080 \
		-e GCP_PROJECT_ID=test-project \
		-e PUBSUB_TOPIC_PREFIX=ssf-events-test \
		-e LOG_LEVEL=debug \
		ssf-broker:latest

docker-push: ## Push Docker image to registry
	@echo "Pushing Docker image..."
	@docker tag ssf-broker:latest $(REGISTRY)/ssf-broker:$(VERSION)
	@docker push $(REGISTRY)/ssf-broker:$(VERSION)

# Kubernetes
k8s-deploy: ## Deploy to Kubernetes
	@echo "Deploying to Kubernetes..."
	@kubectl apply -f deployments/kubernetes/

k8s-delete: ## Delete from Kubernetes
	@echo "Deleting from Kubernetes..."
	@kubectl delete -f deployments/kubernetes/

k8s-logs: ## View Kubernetes logs
	@kubectl logs -l app=ssf-broker --tail=100 -f

k8s-status: ## Check Kubernetes deployment status
	@kubectl get pods -l app=ssf-broker
	@kubectl get svc ssf-broker-service

# Development
dev-setup: ## Set up development environment
	@echo "Setting up development environment..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go mod download

generate: ## Generate mocks and other generated code
	@echo "Generating mocks..."
	@go generate ./...

# Clean
clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@go clean

# Security
security: ## Run security checks
	@echo "Running security checks..."
	@gosec ./...

# Configuration validation
validate-config: ## Validate configuration files
	@echo "Validating configuration files..."
	@yamllint configs/
	@kubeval deployments/kubernetes/*.yaml

# Release
release: clean build-linux test lint ## Build release version
	@echo "Creating release..."
	@mkdir -p release
	@tar -czf release/ssf-broker-$(VERSION)-linux-amd64.tar.gz -C bin ssf-broker-linux
	@echo "Release created: release/ssf-broker-$(VERSION)-linux-amd64.tar.gz"

# Variables (can be overridden)
REGISTRY ?= your-registry.com
VERSION ?= latest

.DEFAULT_GOAL := help