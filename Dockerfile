# Build stage
FROM golang:1.25.5-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make protobuf

# Install buf and protoc plugins
RUN go install github.com/bufbuild/buf/cmd/buf@latest && \
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Set working directory
WORKDIR /app

# Copy proto files and buf config first
COPY buf.yaml buf.gen.yaml ./
COPY proto/ ./proto/

# Generate protobuf code
RUN buf generate

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ledger cmd/server/main.go

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/ledger .

# Expose gRPC port
EXPOSE 9090

# Run the service
CMD ["./ledger"]
