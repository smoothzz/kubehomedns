# Build stage
FROM golang:1.24-alpine AS builder

# Set environment variables
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64

# Set working directory
WORKDIR /app

# Copy go mod and sum files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the Go app statically
RUN go build -o app .

# Final stage - minimal image
FROM alpine:latest

# Install ca-certificates package for TLS verification
RUN apk --no-cache add ca-certificates

# Copy the compiled binary from the builder stage
COPY --from=builder /app/app /app/app

# Command to run the executable
ENTRYPOINT ["/app/app"]