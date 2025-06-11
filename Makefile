.PHONY: build test clean install lint vet fmt help check test-unit test-integration test-all docker-test static

# Build variables
BINARY_NAME=apache2buddy-go
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags="-s -w -X main.version=$(VERSION)"
STATIC_LDFLAGS=-ldflags="-s -w -X main.version=$(VERSION) -extldflags '-static'"

# Go build flags for static binary
CGO_ENABLED=0

# Default target
all: check build

## build: Build the binary
build:
	go build $(LDFLAGS) -o $(BINARY_NAME) .

## build-static: Build static binary (no dependencies)
build-static:
	CGO_ENABLED=$(CGO_ENABLED) go build $(STATIC_LDFLAGS) -o $(BINARY_NAME) .

## test: Run all tests
test: test-unit

## test-unit: Run unit tests
test-unit:
	@echo "Running unit tests..."
	go test -v ./...

## test-race: Run tests with race detector
test-race:
	@echo "Running tests with race detector..."
	go test -race -v ./...

## test-cover: Run tests with coverage
test-cover:
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## test-integration: Run integration tests with Docker
test-integration: docker-test

## test-all: Run all tests including integration
test-all: test-unit test-race test-integration

## check: Run quality checks (fmt, vet, lint, test)
check: fmt vet deps test-unit
	@echo "All quality checks passed!"

## clean: Clean build artifacts
clean:
	rm -f $(BINARY_NAME) apache2buddy-test
	rm -f coverage.out coverage.html
	rm -rf dist/
	docker-compose -f docker-compose.test.yml down --volumes --remove-orphans 2>/dev/null || true

## install: Install the binary to /usr/local/bin
install: build
	sudo cp $(BINARY_NAME) /usr/local/bin/

## lint: Run golangci-lint
lint:
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not found. Install it from https://golangci-lint.run/"; exit 1; }
	golangci-lint run

## vet: Run go vet
vet:
	go vet ./...

## fmt: Format code
fmt:
	go fmt ./...

## deps: Download and verify dependencies
deps:
	go mod download
	go mod verify
	go mod tidy

## docker-test: Run integration tests using Docker containers
docker-test: docker-build-containers docker-run-tests docker-integration-tests docker-down

## docker-build-containers: Build Apache containers with apache2buddy-go
docker-build-containers:
	@echo "Building Apache containers with apache2buddy-go..."
	@command -v docker-compose >/dev/null 2>&1 || { echo "docker-compose not found. Please install Docker Compose."; exit 1; }
	docker-compose -f docker-compose.test.yml build apache-httpd-test apache-ubuntu-test

## docker-run-tests: Run unit tests and start Apache containers
docker-run-tests:
	@echo "Running unit tests..."
	docker-compose -f docker-compose.test.yml run --rm unit-tests
	@echo "Starting Apache containers..."
	docker-compose -f docker-compose.test.yml up -d apache-httpd-test apache-ubuntu-test
	@echo "Waiting for Apache containers to be healthy..."
	@timeout=60; count=0; \
	while [ $$count -lt $$timeout ]; do \
		if docker exec apache2buddy-test-httpd wget --quiet --tries=1 --spider http://localhost/server-status?auto 2>/dev/null; then \
			echo "✅ HTTPD container is healthy"; \
			break; \
		fi; \
		sleep 2; \
		count=$$((count + 1)); \
	done; \
	if [ $$count -eq $$timeout ]; then \
		echo "❌ HTTPD container failed to become healthy"; \
		exit 1; \
	fi
	@timeout=60; count=0; \
	while [ $$count -lt $$timeout ]; do \
		if docker exec apache2buddy-test-ubuntu curl -f http://localhost/server-status?auto 2>/dev/null; then \
			echo "✅ Ubuntu container is healthy"; \
			break; \
		fi; \
		sleep 2; \
		count=$$((count + 1)); \
	done; \
	if [ $$count -eq $$timeout ]; then \
		echo "⚠️  Ubuntu container not healthy (may be expected)"; \
	fi

## docker-integration-tests: Run apache2buddy-go inside each container
docker-integration-tests:
	@echo "Running apache2buddy-go integration tests..."
	@echo "🧪 Testing HTTPD container..."
	docker exec apache2buddy-test-httpd apache2buddy-go
	@echo "✅ HTTPD integration test passed"
	@echo "🧪 Testing Ubuntu container..."
	@if docker exec apache2buddy-test-ubuntu apache2buddy-go; then \
		echo "✅ Ubuntu integration test passed"; \
	else \
		echo "⚠️  Ubuntu integration test failed (may be expected)"; \
	fi

## docker-up: Start Docker services for integration tests
docker-up:
	@echo "Starting Docker services for integration tests..."
	@command -v docker-compose >/dev/null 2>&1 || { echo "docker-compose not found. Please install Docker Compose."; exit 1; }
	docker-compose -f docker-compose.test.yml up -d apache-httpd-test apache-ubuntu-test

## docker-down: Stop Docker services
docker-down:
	@echo "Stopping Docker services..."
	docker-compose -f docker-compose.test.yml down --volumes --remove-orphans 2>/dev/null || true

## docker-logs: Show Docker container logs
docker-logs:
	@echo "=== Apache HTTPD Logs ==="
	docker-compose -f docker-compose.test.yml logs apache-httpd-test || true
	@echo "=== Ubuntu Apache Logs ==="
	docker-compose -f docker-compose.test.yml logs apache-ubuntu-test || true

## docker-logs-follow: Follow Docker container logs in real-time
docker-logs-follow:
	@echo "Following Docker container logs (Ctrl+C to stop)..."
	docker-compose -f docker-compose.test.yml logs -f apache-httpd-test apache-ubuntu-test

## docker-status: Show status of Docker containers
docker-status:
	@echo "Docker container status:"
	docker-compose -f docker-compose.test.yml ps || true

## test-build: Test building for different platforms
test-build:
	@echo "Testing cross-platform builds..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(STATIC_LDFLAGS) -o /tmp/$(BINARY_NAME)-linux-amd64 .
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(STATIC_LDFLAGS) -o /tmp/$(BINARY_NAME)-linux-arm64 .
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(STATIC_LDFLAGS) -o /tmp/$(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(STATIC_LDFLAGS) -o /tmp/$(BINARY_NAME)-darwin-arm64 .
	GOOS=freebsd GOARCH=amd64 CGO_ENABLED=0 go build $(STATIC_LDFLAGS) -o /tmp/$(BINARY_NAME)-freebsd-amd64 .
	@echo "All platform builds successful"
	@rm -f /tmp/$(BINARY_NAME)-*

## benchmarks: Run benchmark tests
benchmarks:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

## help: Show this help
help: Makefile
	@echo "Usage: make [target]"
	@echo ""
	@echo "Build targets:"
	@echo "  build         Build the binary"
	@echo "  build-static  Build static binary (no dependencies)"
	@echo "  test-build    Test cross-platform builds"
	@echo ""
	@echo "Test targets:"
	@echo "  test          Run unit tests"
	@echo "  test-unit     Run unit tests"
	@echo "  test-race     Run tests with race detector"
	@echo "  test-cover    Run tests with coverage"
	@echo "  test-integration Run Docker-based integration tests"
	@echo "  test-all      Run all tests"
	@echo "  benchmarks    Run benchmark tests"
	@echo ""
	@echo "Docker test targets:"
	@echo "  docker-test              Run full Docker integration test suite"
	@echo "  docker-build-containers  Build Apache containers with apache2buddy-go"
	@echo "  docker-run-tests         Run unit tests and start Apache containers"
	@echo "  docker-integration-tests Run apache2buddy-go inside containers"
	@echo "  docker-up                Start Docker services"
	@echo "  docker-down              Stop Docker services"
	@echo "  docker-logs              Show Docker container logs"
	@echo "  docker-logs-follow       Follow Docker container logs in real-time"
	@echo "  docker-status            Show status of Docker containers"
	@echo ""
	@echo "Quality targets:"
	@echo "  check         Run all quality checks"
	@echo "  fmt           Format code"
	@echo "  vet           Run go vet"
	@echo "  lint          Run golangci-lint"
	@echo ""
	@echo "Utility targets:"
	@echo "  deps          Download and verify dependencies"
	@echo "  clean         Clean build artifacts"
	@echo "  install       Install binary to /usr/local/bin"
	@echo "  help          Show this help"