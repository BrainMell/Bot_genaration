# Go Image Service - Chrome-Powered
# Optimized build for Alpine
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY pkg/ ./pkg/
COPY main.go ./

# Build binary with modern optimizations (removed deprecated -installsuffix)
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o image-service .

# =============================================================================
# Final Stage - Runtime with Chrome + ffmpeg + yt-dlp
# =============================================================================
FROM alpine:latest

# Install runtime dependencies
# We need:
# - ca-certificates for HTTPS
# - ffmpeg for media processing
# - chromium for Rod (Chrome automation)
# - python3 + yt-dlp for audio extraction
RUN apk add --no-cache \
    ca-certificates \
    ffmpeg \
    wget \
    python3 \
    py3-pip \
    chromium \
    && pip3 install --no-cache-dir --break-system-packages -U yt-dlp \
    && rm -rf /var/cache/apk/* /root/.cache

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/image-service .

# Copy assets
COPY assets ./assets

# Environment variables
ENV GIN_MODE=release
# Rod path for Alpine
ENV CHROME_PATH=/usr/bin/chromium-browser

# Expose port
EXPOSE 7860

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:7860/health || exit 1

# Run the service
CMD ["./image-service"]
