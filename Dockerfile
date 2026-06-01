# ── Stage 1: Build Go binary ──────────────────────────────────────────────────
FROM golang:1.24-bookworm AS builder

# Install git-lfs
RUN apt-get update && apt-get install -y git-lfs && git lfs install

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN git lfs pull
RUN CGO_ENABLED=0 GOOS=linux go build -o image-service .

# ── Stage 2: Runtime ──────────────────────────────────────────────────────────
FROM debian:bookworm-slim

# System deps
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    wget \
    gnupg \
    python3 \
    python3-pip \
    ffmpeg \
    fonts-liberation \
    libasound2 \
    libatk-bridge2.0-0 \
    libatk1.0-0 \
    libcups2 \
    libdbus-1-3 \
    libdrm2 \
    libgbm1 \
    libglib2.0-0 \
    libgtk-3-0 \
    libnspr4 \
    libnss3 \
    libx11-6 \
    libxcomposite1 \
    libxdamage1 \
    libxext6 \
    libxfixes3 \
    libxrandr2 \
    libxshmfence1 \
    xdg-utils \
    && rm -rf /var/lib/apt/lists/*

# Google Chrome
RUN wget -q -O /tmp/chrome.deb https://dl.google.com/linux/direct/google-chrome-stable_current_amd64.deb \
    && apt-get install -y /tmp/chrome.deb \
    && rm /tmp/chrome.deb

# Node.js 20 LTS
RUN curl -fsSL https://deb.nodesource.com/setup_20.x | bash - \
    && apt-get install -y nodejs \
    && rm -rf /var/lib/apt/lists/*

# yt-dlp
RUN pip3 install --break-system-packages yt-dlp

WORKDIR /app

# Copy Go binary
COPY --from=builder /build/image-service .

# Copy assets (fonts, images, etc.) — required for card generation
COPY --from=builder /build/assets ./assets

# Copy Node.js scraper files
COPY scraper-api.js package.json ./
RUN npm install --omit=dev

# Copy startup script
COPY start.sh .
RUN chmod +x start.sh


ENV MODE=full
ENV PORT=7860
ENV SCRAPER_PORT=7861
ENV GIN_MODE=release

EXPOSE 7860

CMD ["./start.sh"]