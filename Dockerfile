# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
# The binary will be statically linked
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o mein-idaas .

# Runtime stage
FROM alpine:latest

# Install runtime dependencies (ca-certificates for HTTPS, tzdata for timezones)
RUN apk --no-cache add ca-certificates tzdata

# Set working directory
WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/mein-idaas .

# Copy .env file (will be overridden at runtime via docker run -e or docker-compose)
# For development only - in production, use --build-arg or runtime environment variables
COPY .env .

# Expose the port (matches PORT in .env)
EXPOSE 4000

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:4000/api/v1/health || exit 1

# Run the application
# Environment variables can be passed via:
# docker run -e PORT=4000 -e DB_HOST=postgres ...
# or docker-compose with env_file directive
CMD ["./mein-idaas"]

