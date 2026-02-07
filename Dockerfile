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

# Pre-download Chromium during build (so it's cached in the image)
# Note: We do this in the final stage instead if we want it in the final image, 
# but Rod usually downloads to a local cache folder.
# For Alpine, it's better to install it via apk which we already do.
# However, running --prepare ensures all Rod-specific setups are done.

# =============================================================================
# Final Stage - Runtime with Chromium for Rod
# =============================================================================
FROM alpine:latest

# Install runtime dependencies for Rod
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

# Environment for Rod
ENV CHROME_BIN=/usr/bin/chromium-browser \
    CHROME_PATH=/usr/bin/chromium-browser

# Run prepare step to ensure everything is ready
RUN ./image-service --prepare

# Expose port
EXPOSE 8080

# Run
CMD ["./image-service"]