# ── Stage 1: Build Go binary ──────────────────────────────────────────────────
FROM golang:1.24-bookworm AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY pkg/ ./pkg/
COPY main.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o image-service .

# ── Stage 2: Runtime ──────────────────────────────────────────────────────────
FROM debian:bookworm-slim

# Install system dependencies (Node.js + Chrome + FFmpeg)
RUN apt-get update && apt-get install -y \
    wget \
    gnupg \
    ca-certificates \
    ffmpeg \
    curl \
    python3 \
    --no-install-recommends

# Install Node.js
RUN curl -fsSL https://deb.nodesource.com/setup_20.x | bash - \
    && apt-get install -y nodejs

# Install real Google Chrome (avoids snap trap)
RUN wget -q -O - https://dl-ssl.google.com/linux/linux_signing_key.pub | apt-key add - \
    && echo "deb [arch=amd64] http://dl.google.com/linux/chrome/deb/ stable main" \
       > /etc/apt/sources.list.d/google-chrome.list \
    && apt-get update \
    && apt-get install -y google-chrome-stable --no-install-recommends \
    && rm -rf /var/lib/apt/lists/*

# Symlink so browser.go finds it
RUN ln -sf /usr/bin/google-chrome-stable /usr/bin/chromium-browser

# Install yt-dlp
RUN curl -L https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp \
    -o /usr/local/bin/yt-dlp && chmod a+rx /usr/local/bin/yt-dlp

WORKDIR /app

# Copy Go binary
COPY --from=builder /build/image-service .
# Copy Node app
COPY package.json scraper-api.js ./
RUN npm install --production

COPY assets ./assets
RUN mkdir -p downloads && chmod 777 downloads

# Environment variables
ENV MODE=hf
ENV PORT=7860
ENV SCRAPER_PORT=7861
ENV GIN_MODE=release
ENV CHROME_PATH=/usr/bin/google-chrome-stable

EXPOSE 7860
EXPOSE 7861

# Start script to run both
RUN echo '#!/bin/bash\nnpm start &\n./image-service' > start.sh && chmod +x start.sh

CMD ["./start.sh"]
