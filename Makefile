.PHONY: build run clean test lint install coverage coverage-html

# Binary name
BINARY_NAME=simtool
MAIN_PATH=./cmd/simtool

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
BUILT_BY := $(shell whoami)

# Build flags
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE) -X main.builtBy=$(BUILT_BY)"

# Build the application
build:
	go build $(LDFLAGS) -o $(BINARY_NAME) $(MAIN_PATH)

# Run the application
run:
	go run $(MAIN_PATH)

# Clean build artifacts
clean:
	go clean
	rm -f $(BINARY_NAME)
	rm -f debug.log

# Run tests
test:
	go test -v ./...

# Run linter (requires golangci-lint)
lint:
	golangci-lint run

# Install the binary to GOPATH/bin
install:
	go install $(LDFLAGS) $(MAIN_PATH)

# Format code
fmt:
	go fmt ./...

# Download dependencies
deps:
	go mod download
	go mod tidy

# Build for multiple platforms
build-all:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)

# Run tests with coverage
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
	@echo ""
	@echo "To update the README badge, run: ./scripts/coverage-badge.sh"

# Generate HTML coverage report
coverage-html: coverage
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"