#!/bin/bash
# ============================================================
# setup.sh — runs ONCE when the Codespace is first created.
# Installs Chrome + yt-dlp, builds the Go binary.
# Does NOT run on every start (that's start.sh).
# ============================================================

set -e

echo "📦 [setup] Installing system dependencies..."
sudo apt-get update -qq
sudo apt-get install -y -qq wget gnupg ca-certificates ffmpeg

# Install real Google Chrome (avoids the snap trap)
echo "🌐 [setup] Installing Google Chrome..."
wget -q https://dl.google.com/linux/direct/google-chrome-stable_current_amd64.deb
sudo apt-get install -y -qq ./google-chrome-stable_current_amd64.deb
rm google-chrome-stable_current_amd64.deb

# Symlink so the Go binary finds it at /usr/bin/chromium-browser
sudo ln -sf /usr/bin/google-chrome /usr/bin/chromium-browser

# Install yt-dlp
echo "🎵 [setup] Installing yt-dlp..."
sudo wget -q -O /usr/local/bin/yt-dlp https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp
sudo chmod +x /usr/local/bin/yt-dlp

# Build the Go binary
echo "🔨 [setup] Building Go binary..."
cd /workspaces/Bot_genaration
go mod tidy
go build -o image-service .

echo "✅ [setup] Setup complete. Binary ready at ./image-service"