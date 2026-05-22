#!/bin/bash
set -e

echo "🚀 Starting VDAP services..."

# Install Node deps if node_modules doesn't exist yet
if [ ! -d "node_modules" ]; then
  echo "📦 Installing Node.js dependencies..."
  npm install --omit=dev
fi

# Start the Puppeteer scraper sidecar in the background
echo "🌐 Starting Puppeteer scraper on port ${SCRAPER_PORT:-7861}..."
node scraper-api.js &
SCRAPER_PID=$!

# Give Node a moment to bind to its port before Go starts proxying
sleep 2

# Start the Go server (foreground — keeps container alive)
echo "⚙️  Starting Go image service..."
./image-service

# If Go exits, kill the scraper too
kill $SCRAPER_PID 2>/dev/null || true