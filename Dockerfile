# Go Image & Scraper Service - Optimized for chromedp
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o image-service .

# =============================================================================
# Final Stage - Runtime with Chromium for chromedp
# =============================================================================
FROM alpine:latest

# Install runtime dependencies for chromedp
RUN apk add --no-cache \
    ca-certificates \
    chromium \
    nss \
    freetype \
    harfbuzz \
    ttf-freefont \
    && rm -rf /var/cache/apk/*

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/image-service .

# Copy assets (IMPORTANT!)
COPY assets ./assets

# Environment for chromedp
ENV CHROME_BIN=/usr/bin/chromium-browser \
    CHROME_PATH=/usr/bin/chromium-browser

# Expose port
EXPOSE 8080

# Run
CMD ["./image-service"]