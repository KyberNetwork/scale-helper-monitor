FROM golang:1.24-alpine

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go build -o scale-helper-monitor ./cmd/monitor

# Command to run the monitoring service
CMD ["./scale-helper-monitor"] 