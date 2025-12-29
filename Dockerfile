# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Allow Go to auto-download the required toolchain version from go.mod
ENV GOTOOLCHAIN=auto

# Install build deps (git often needed for go modules)
RUN apk add --no-cache git

# Copy go.mod/go.sum first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source
COPY . .

# Build the cmd/app binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/bin/app ./cmd/app

# Runtime stage
FROM alpine:3.20

WORKDIR /app

# Set non-root user for safety
RUN addgroup -S app && adduser -S app -G app
USER app

# Copy binary and migrations
COPY --from=builder /app/bin/app /app/app
COPY --from=builder /app/migrations /app/migrations

# Expose HTTP port
EXPOSE 8080

# Default command runs the API server
ENTRYPOINT ["/app/app"]
