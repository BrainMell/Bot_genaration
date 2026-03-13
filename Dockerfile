# Go Image Service - Ultra-Stable Build
# Version: 3.0.2 (Build-Safe)
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Copy and verify dependencies
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy ALL source code (crucial for local imports)
COPY pkg/ ./pkg/
COPY main.go ./

# Build with RAM safety (GOMAXPROCS=1) and verbose output
# We remove -trimpath to see if it helps with memory
RUN GOMAXPROCS=1 CGO_ENABLED=0 GOOS=linux go build -v -ldflags="-s -w" -o image-service .

# =============================================================================
# Final Stage - Runtime
# =============================================================================
FROM alpine:latest

# Install dependencies including Chromium for Rod
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

# Copy binary
COPY --from=builder /build/image-service .

# Copy assets exactly as structured
COPY assets ./assets

# Environment
ENV GIN_MODE=release
ENV CHROME_PATH=/usr/bin/chromium-browser

EXPOSE 7860

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:7860/health || exit 1

CMD ["./image-service"]
