.PHONY: proto test lint clean run

# Generate protobuf code
proto:
	@echo "Generating protobuf code..."
	buf generate

# Run tests
test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...

# Run tests with coverage report
coverage:
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Lint code
lint:
	@echo "Linting code..."
	golangci-lint run ./...

# Format code
fmt:
	@echo "Formatting code..."
	gofmt -s -w .
	goimports -w .

# Clean generated files
clean:
	@echo "Cleaning generated files..."
	rm -rf gen/
	rm -f coverage.out coverage.html

# Build the service
build:
	@echo "Building service..."
	go build -o bin/ledger cmd/server/main.go

# Run the service
run:
	@echo "Running service..."
	go run cmd/server/main.go

# Install development tools
install-tools:
	@echo "Installing development tools..."
	go install github.com/bufbuild/buf/cmd/buf@latest
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest

# Tidy dependencies
tidy:
	go mod tidy
