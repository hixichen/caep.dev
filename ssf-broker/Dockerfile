# Multi-stage build for SSF Broker Service
FROM golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

# Install dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the SSF broker service
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ssf-broker ./cmd/server/

# Final stage
FROM alpine:latest

# Install ca-certificates and wget for health checks
RUN apk --no-cache add ca-certificates wget

WORKDIR /root/

# Create directories for configs and secrets
RUN mkdir -p /etc/config /var/secrets/google /var/lib/ssf-broker

# Copy the binary from builder stage
COPY --from=builder /app/ssf-broker .

# Copy configuration files
COPY --from=builder /app/configs/ /etc/config/

# Expose the HTTP server port
EXPOSE 8080

# Add health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Create non-root user for security
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Change ownership of the working directory
RUN chown -R appuser:appgroup /root /var/lib/ssf-broker

# Switch to non-root user
USER appuser

# Set default environment variables
ENV SERVER_PORT=8080
ENV LOG_LEVEL=info
ENV LOG_FORMAT=json

# Run the SSF broker service
CMD ["./ssf-broker"]