#!/bin/bash
# ============================================================
# start.sh — runs every time the Codespace wakes up.
# ============================================================

set -e

WORKDIR="/workspaces/Bot_genaration"
cd "$WORKDIR"

# 1. Kill any existing instances to prevent port conflicts
echo "🧹 [start] Cleaning up old processes..."
pkill image-service || true
pkill cloudflared || true

# 2. Rebuild binary if it's missing
if [ ! -f "./image-service" ]; then
  echo "⚠️  [start] Binary missing — rebuilding..."
  go build -o image-service .
fi

# 3. Install cloudflared if not present
if ! command -v cloudflared &> /dev/null; then
  echo "🌐 [start] Installing cloudflared..."
  curl -sL https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64 \
    -o /usr/local/bin/cloudflared
  chmod +x /usr/local/bin/cloudflared
fi

# 4. Start the Go service
echo "🚀 [start] Starting Go service in CODESPACE mode..."
# Use 127.0.0.1 to be explicit
nohup MODE=codespace PORT=7860 ./image-service > /tmp/image-service.log 2>&1 &
SERVICE_PID=$!
echo "$SERVICE_PID" > /tmp/image-service.pid

# 5. Wait for the service to be healthy (using 127.0.0.1)
echo "⏳ [start] Waiting for service to be ready..."
HEALTHY=false
for i in $(seq 1 30); do
  if curl -s http://127.0.0.1:7860/health > /dev/null 2>&1; then
    echo "✅ [start] Service is healthy."
    HEALTHY=true
    break
  fi
  sleep 1
done

if [ "$HEALTHY" = false ]; then
  echo "❌ [start] Service failed to start. Check /tmp/image-service.log"
  exit 1
fi

# 6. Start Cloudflare tunnel
echo "🌐 [start] Starting Cloudflare tunnel..."
nohup cloudflared tunnel --url http://127.0.0.1:7860 > /tmp/cloudflared.log 2>&1 &
TUNNEL_PID=$!
echo "$TUNNEL_PID" > /tmp/cloudflared.pid

# 7. Wait for the tunnel URL
echo "⏳ [start] Waiting for tunnel URL..."
for i in $(seq 1 30); do
  TUNNEL_URL=$(grep -oP 'https://[a-z0-9\-]+\.trycloudflare\.com' /tmp/cloudflared.log 2>/dev/null | head -1)
  if [ -n "$TUNNEL_URL" ]; then
    echo "$TUNNEL_URL" > /tmp/tunnel_url.txt
    echo "✅ [start] Tunnel URL: $TUNNEL_URL"

    # --- AUTOMATION: Update Render Env Var ---
    if [ -n "$RENDER_API_KEY" ] && [ -n "$RENDER_SERVICE_ID" ]; then
      echo "🚀 [start] Automating Render update..."
      curl -s --request PATCH \
        --url "https://api.render.com/v1/services/$RENDER_SERVICE_ID/env-vars" \
        --header "Authorization: Bearer $RENDER_API_KEY" \
        --header "Content-Type: application/json" \
        --data "[
          {
            \"key\": \"CODESPACE_SERVICE_URL\",
            \"value\": \"$TUNNEL_URL\"
          }
        ]"
      echo "✅ [start] Render environment updated. Orchestrator will restart shortly."
    else
      echo "⚠️  [start] RENDER_API_KEY or RENDER_SERVICE_ID missing. Skipping automation."
    fi
    # -----------------------------------------

    echo ""
    echo "=============================================="
    echo "  SERVICE URL: $TUNNEL_URL"
    echo "=============================================="
    break
  fi
  sleep 2
done

echo "🟢 [start] Everything running. Logs:"
echo "  Service:  tail -f /tmp/image-service.log"
echo "  Tunnel:   tail -f /tmp/cloudflared.log"
