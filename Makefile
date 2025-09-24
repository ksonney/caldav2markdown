# Makefile for caldav2markdown

# Variables
BINARY_NAME=caldav2markdown
BIN_DIR=bin
BUILD_DIR=./cmd/caldav2markdown
GO_FILES=$(shell find . -type f -name '*.go')

# Default target
.PHONY: all
all: build

# Build the application
.PHONY: build
build: $(BIN_DIR)/$(BINARY_NAME)

$(BIN_DIR)/$(BINARY_NAME): $(GO_FILES)
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(BINARY_NAME) $(BUILD_DIR)

# Run the application
.PHONY: run
run: build
	./$(BIN_DIR)/$(BINARY_NAME)

# Test connection to CalDAV server
.PHONY: test-connection
test-connection: build
	./$(BIN_DIR)/$(BINARY_NAME) -test

# Run with custom configuration
.PHONY: run-config
run-config: build
	./$(BIN_DIR)/$(BINARY_NAME) -config $(CONFIG_FILE)

# Run tests
.PHONY: test
test:
	go test ./...

# Run tests with verbose output
.PHONY: test-verbose
test-verbose:
	go test -v ./...

# Clean build artifacts
.PHONY: clean
clean:
	rm -rf $(BIN_DIR)
	go clean

# Install dependencies
.PHONY: deps
deps:
	go mod tidy
	go mod download

# Build for multiple platforms
.PHONY: build-all
build-all: build-linux build-darwin build-windows

.PHONY: build-linux
build-linux:
	@mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 go build -o $(BIN_DIR)/$(BINARY_NAME)-linux-amd64 $(BUILD_DIR)

.PHONY: build-darwin
build-darwin:
	@mkdir -p $(BIN_DIR)
	GOOS=darwin GOARCH=amd64 go build -o $(BIN_DIR)/$(BINARY_NAME)-darwin-amd64 $(BUILD_DIR)
	GOOS=darwin GOARCH=arm64 go build -o $(BIN_DIR)/$(BINARY_NAME)-darwin-arm64 $(BUILD_DIR)

.PHONY: build-windows
build-windows:
	@mkdir -p $(BIN_DIR)
	GOOS=windows GOARCH=amd64 go build -o $(BIN_DIR)/$(BINARY_NAME)-windows-amd64.exe $(BUILD_DIR)

# Development build with debug info
.PHONY: build-dev
build-dev:
	@mkdir -p $(BIN_DIR)
	go build -gcflags="all=-N -l" -o $(BIN_DIR)/$(BINARY_NAME)-dev $(BUILD_DIR)

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build          Build the application (default)"
	@echo "  run            Build and run the application"
	@echo "  test-connection Build and test CalDAV connection"
	@echo "  run-config     Build and run with custom config (set CONFIG_FILE)"
	@echo "  test           Run all tests"
	@echo "  test-verbose   Run tests with verbose output"
	@echo "  clean          Remove build artifacts"
	@echo "  deps           Install and tidy dependencies"
	@echo "  build-all      Build for all platforms"
	@echo "  build-linux    Build for Linux"
	@echo "  build-darwin   Build for macOS"
	@echo "  build-windows  Build for Windows"
	@echo "  build-dev      Build with debug information"
	@echo "  help           Show this help message"