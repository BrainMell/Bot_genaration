#!/bin/bash
# ============================================================
# start.sh — runs every time the Codespace wakes up.
# Starts the Go service in CODESPACE mode, then starts
# a Cloudflare tunnel so the outside world can reach it.
# The tunnel URL is written to /tmp/tunnel_url.txt so the
# orchestrator on Render can read it (if needed).
# ============================================================

set -e

WORKDIR="/workspaces/Bot_genaration"
cd "$WORKDIR"

# Rebuild binary if it's missing (safety net)
if [ ! -f "./image-service" ]; then
  echo "⚠️  [start] Binary missing — rebuilding..."
  go build -o image-service .
fi

# Install cloudflared if not present
if ! command -v cloudflared &> /dev/null; then
  echo "🌐 [start] Installing cloudflared..."
  curl -sL https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64 \
    -o /usr/local/bin/cloudflared
  chmod +x /usr/local/bin/cloudflared
fi

# Start the Go service (codespace mode) in the background
echo "🚀 [start] Starting Go service in CODESPACE mode..."
MODE=codespace PORT=7860 ./image-service > /tmp/image-service.log 2>&1 &
SERVICE_PID=$!
echo "$SERVICE_PID" > /tmp/image-service.pid

# Wait for the service to be healthy before starting tunnel
echo "⏳ [start] Waiting for service to be ready..."
for i in $(seq 1 30); do
  if curl -s http://localhost:7860/health > /dev/null 2>&1; then
    echo "✅ [start] Service is healthy."
    break
  fi
  sleep 1
done

# Start Cloudflare tunnel — output goes to log, URL extracted
echo "🌐 [start] Starting Cloudflare tunnel..."
cloudflared tunnel --url http://localhost:7860 > /tmp/cloudflared.log 2>&1 &
TUNNEL_PID=$!
echo "$TUNNEL_PID" > /tmp/cloudflared.pid

# Wait for the tunnel URL to appear in the log
echo "⏳ [start] Waiting for tunnel URL..."
for i in $(seq 1 30); do
  TUNNEL_URL=$(grep -oP 'https://[a-z0-9\-]+\.trycloudflare\.com' /tmp/cloudflared.log 2>/dev/null | head -1)
  if [ -n "$TUNNEL_URL" ]; then
    echo "$TUNNEL_URL" > /tmp/tunnel_url.txt
    echo "✅ [start] Tunnel URL: $TUNNEL_URL"
    echo ""
    echo "=============================================="
    echo "  SERVICE URL: $TUNNEL_URL"
    echo "  Set this in Render env: CODESPACE_SERVICE_URL=$TUNNEL_URL"
    echo "=============================================="
    break
  fi
  sleep 2
done

echo "🟢 [start] Everything running. Logs:"
echo "  Service:  tail -f /tmp/image-service.log"
echo "  Tunnel:   tail -f /tmp/cloudflared.log"
