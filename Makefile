.PHONY: build run clean test lint install

# Binary name
BINARY_NAME=simtool
MAIN_PATH=./cmd/simtool

# Build the application
build:
	go build -o $(BINARY_NAME) $(MAIN_PATH)

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
	go install $(MAIN_PATH)

# Format code
fmt:
	go fmt ./...

# Download dependencies
deps:
	go mod download
	go mod tidy

# Build for multiple platforms
build-all:
	GOOS=darwin GOARCH=amd64 go build -o $(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=arm64 go build -o $(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)