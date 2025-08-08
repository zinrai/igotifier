# Build stage
FROM golang:1.21-bookworm AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY main.go ./

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o igotifier \
    main.go

# Runtime stage
FROM debian:bookworm-slim

# Install ca-certificates for HTTPS support and basic shell utilities
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Create non-root user
RUN useradd -r -u 1001 -s /bin/false igotifier

# Copy binary from builder
COPY --from=builder /build/igotifier /usr/local/bin/igotifier

# Make binary executable
RUN chmod +x /usr/local/bin/igotifier

# Switch to non-root user
USER igotifier

# Set entrypoint
ENTRYPOINT ["igotifier"]

# Default command shows help
CMD ["-h"]
