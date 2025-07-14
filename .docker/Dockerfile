# Build stage - for building custom golangci-lint
FROM golang:1.23-alpine AS builder

# Install required build tools
RUN apk add --no-cache git make curl bash

# Install golangci-lint
ARG GOLANGCI_LINT_VERSION=v2.1.5
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b /usr/local/bin ${GOLANGCI_LINT_VERSION}

# Create work directory
WORKDIR /build

# Copy Go module files first for better layer caching
COPY go.mod go.sum ./

# Download Go modules
RUN go mod download

# Copy the entire source code (needed for custom plugins)
COPY . .

# Build custom golangci-lint with our plugin
RUN echo "Building custom golangci-lint with custom plugins..." && \
    golangci-lint custom && \
    echo "Custom golangci-lint built successfully"

# Runtime stage - final image with Go environment
FROM golang:1.23-alpine

# Install runtime dependencies
RUN apk add --no-cache bash sed grep git

# Git security fix - mark mounted directories as safe
RUN git config --global --add safe.directory /app

# Copy the custom golangci-lint binary from builder stage
COPY --from=builder /build/custom-gcl /usr/local/bin/qor5-linter

# Create working directory
WORKDIR /app

# Add entrypoint script
COPY .docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"] 
