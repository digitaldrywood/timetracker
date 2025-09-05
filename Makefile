.PHONY: build run test clean auth summary week add suggest install lint vet fmt

BINARY_NAME=timetracker
MAIN_PATH=cmd/timetracker/main.go

build:
	@echo "Building $(BINARY_NAME)..."
	@go build -o bin/$(BINARY_NAME) $(MAIN_PATH)

run: build
	@./bin/$(BINARY_NAME)

test:
	@echo "Running tests..."
	@go test -v ./...

clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@go clean

auth:
	@echo "Building authentication tool..."
	@go build -o bin/auth cmd/auth/main.go
	@echo "Authenticating with Google Sheets..."
	@./bin/auth

summary: build
	@echo "Getting today's summary..."
	@./bin/$(BINARY_NAME) -summary

week: build
	@echo "Getting weekly summary..."
	@./bin/$(BINARY_NAME) -week

add: build
	@echo "Adding time entry..."
	@./bin/$(BINARY_NAME) -add

suggest: build
	@echo "Getting suggested entries from GitHub activity..."
	@./bin/$(BINARY_NAME) -suggest

install:
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	@go build -o /usr/local/bin/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Installed successfully!"

lint:
	@echo "Running linters..."
	@golangci-lint run ./...

vet:
	@echo "Running go vet..."
	@go vet ./...

fmt:
	@echo "Formatting code..."
	@go fmt ./...

deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

help:
	@echo "Available commands:"
	@echo "  make build    - Build the binary"
	@echo "  make run      - Build and run the application"
	@echo "  make test     - Run tests"
	@echo "  make clean    - Clean build artifacts"
	@echo "  make auth     - Authenticate with Google Sheets"
	@echo "  make summary  - Show today's time tracking summary"
	@echo "  make week     - Show weekly summary"
	@echo "  make add      - Add a time entry interactively"
	@echo "  make suggest  - Get suggested entries from GitHub activity"
	@echo "  make install  - Install binary to /usr/local/bin"
	@echo "  make lint     - Run linters"
	@echo "  make vet      - Run go vet"
	@echo "  make fmt      - Format code"
	@echo "  make deps     - Download dependencies"
	@echo "  make help     - Show this help message"