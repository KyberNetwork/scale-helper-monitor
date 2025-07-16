# Build stage
FROM golang:1.24-alpine AS builder

# Install git and ca-certificates (needed for some dependencies)
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the distributor monitor
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o distributor-monitor ./cmd/distributor

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

# Create app directory
WORKDIR /app

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Copy binary from builder stage
COPY --from=builder /app/distributor-monitor .

# Copy configuration files
COPY --chown=appuser:appgroup distributor-config.yaml .
COPY --chown=appuser:appgroup config/ ./config/

# Create directory for state file and set permissions
RUN mkdir -p config/distributor && \
    touch config/distributor/State.json && \
    chown -R appuser:appgroup config/

# Switch to non-root user
USER appuser

# Create volumes for configuration and state persistence
VOLUME ["/app/config"]

# Set environment variables
ENV GO_ENV=production

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD ps aux | grep '[d]istributor-monitor' || exit 1

# Run the distributor monitor
CMD ["./distributor-monitor"] 