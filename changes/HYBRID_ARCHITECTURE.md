# 🚀 Go Image Service — Hybrid Architecture

## Overview

This service runs in two modes across two platforms:

| Mode | Platform | Handles |
|------|----------|---------|
| `render` | Render.com (24/7, free tier) | Combat images, game boards, card burn/convert, stickers, rule34 (Gelbooru API) |
| `codespace` | GitHub Codespaces (on-demand) | Pinterest scrape, PornPics, Audio, Powerscale, Anikai, Anime News, Card GIFs |

Your bot talks to the **Orchestrator** (also on Render), which routes requests automatically and caches results in Upstash Redis.

```
Bot → Orchestrator (Render, 24/7)
           │
           ├── Lightweight? → Go Service on Render (milliseconds)
           │
           └── Heavy? → Check Redis cache
                             │
                             ├── Cache HIT  → Return instantly
                             └── Cache MISS → Wake Codespace (30-45s)
                                                  → Process
                                                  → Cache result
                                                  → Kill Codespace
```

---

## Part 1 — GitHub Codespace Setup

### Step 1: First-time setup (runs once automatically)

When you first open the Codespace, `.devcontainer/setup.sh` runs automatically and:
- Installs Google Chrome
- Installs `yt-dlp`
- Installs `ffmpeg`
- Builds the Go binary

You don't need to do anything — just open the Codespace and wait.

### Step 2: Every wake-up (runs automatically)

`.devcontainer/start.sh` runs every time the Codespace starts. It:
- Starts `./image-service` in `MODE=codespace`
- Starts a Cloudflare tunnel
- Prints the tunnel URL in the terminal

**After it starts, copy the tunnel URL from the terminal:**
```
==============================================
  SERVICE URL: https://random-words.trycloudflare.com
  Set this in Render env: CODESPACE_SERVICE_URL=https://...
==============================================
```

### Step 3: Set the tunnel URL in Render

Every time the Codespace wakes up, the Cloudflare tunnel URL changes.
Go to your Render dashboard → Orchestrator service → Environment → update:
```
CODESPACE_SERVICE_URL=https://your-new-tunnel-url.trycloudflare.com
```

> ⚠️ This is the only manual step required after each Codespace wake. The Orchestrator won't know where the Codespace is without this URL.

### Step 4: Get your GitHub Codespace name

Run this in the Codespace terminal:
```bash
echo $CODESPACE_NAME
```
Copy that value — you'll need it for Render env vars.

### Step 5: Create a GitHub Personal Access Token

1. Go to github.com → Settings → Developer Settings → Personal Access Tokens → Fine-grained
2. Create a token with **Codespaces: read and write** permission
3. Copy the token

---

## Part 2 — Render Setup (Go Service — Render mode)

### Step 1: Deploy the Go service to Render

1. Push this repo to GitHub
2. Go to dashboard.render.com → New → Web Service
3. Connect your repo
4. Set **Runtime** to `Docker`
5. Set environment variables:
   ```
   MODE=render
   PORT=7860
   RAPIDAPI_KEY=your_key (optional)
   SCRAPE_CREATORS_KEY=your_key (optional)
   GELBOORU_USER_ID=your_id (optional)
   GELBOORU_API_KEY=your_key (optional)
   KLIPY_API_KEY=your_key (optional)
   ```
6. Deploy and copy the service URL (e.g. `https://goten-image-service.onrender.com`)

---

## Part 3 — Render Setup (Orchestrator)

### Step 1: Deploy the Orchestrator

1. In Render → New → Web Service
2. Connect the same repo but set **Root Directory** to `orchestrator`
3. Set **Runtime** to `Node`
4. Set environment variables:
   ```
   RENDER_SERVICE_URL=https://your-go-service.onrender.com
   CODESPACE_NAME=your-codespace-name-from-step-4-above
   GITHUB_TOKEN=your-github-pat
   CODESPACE_SERVICE_URL=https://your-tunnel-url.trycloudflare.com
   UPSTASH_REDIS_URL=https://your-redis.upstash.io
   UPSTASH_REDIS_TOKEN=your-upstash-token
   PORT=3000
   ```
5. Deploy and copy the orchestrator URL

### Step 2: Set up Upstash Redis (free, no credit card)

1. Go to console.upstash.com
2. Sign up with GitHub (no card needed)
3. Create a new Redis database (free tier)
4. Copy the **REST URL** and **REST Token**
5. Paste them into Render env vars above

---

## Part 4 — Connect Your Bot

In your bot's `.env`:
```env
GO_IMAGE_SERVICE_URL=https://your-orchestrator.onrender.com
```

Copy `node-client.js` to your bot's `core/` folder:
```javascript
const GoImageService = require('./core/goImageService');
const goService = new GoImageService();

// Lightweight (instant):
const image = await goService.generateCombatImage({ players, enemies, ... });

// Heavy (wakes Codespace if not cached, ~45s first time):
const results = await goService.searchPinterest('goku', 10);
```

---

## Part 5 — Keeping Render Alive (Free Tier)

Render's free tier sleeps after 15 minutes of inactivity. To prevent this, set up UptimeRobot:

1. Go to uptimerobot.com (free)
2. Add New Monitor → HTTP(s)
3. URL: `https://your-orchestrator.onrender.com/health`
4. Interval: every 5 minutes

This keeps the orchestrator awake all month within Render's free hour limits.

---

## Codespace Hour Budget

GitHub gives you **120 core-hours/month** free. The default Codespace uses 2 cores = 60 real hours.

With this architecture, the Codespace only runs for **10–20 seconds per heavy request**. That means:

```
60 hours × 3600 seconds = 216,000 seconds budget
20 seconds per request  = ~10,800 heavy requests per month for free
```

That's effectively unlimited for normal bot usage.

---

## Environment Variables — Full Reference

### Go Service (Render)
| Variable | Required | Description |
|----------|----------|-------------|
| `MODE` | Yes | Set to `render` |
| `PORT` | No | Default `7860` |
| `RAPIDAPI_KEY` | No | For Pinterest RapidAPI fallback |
| `SCRAPE_CREATORS_KEY` | No | For Pinterest ScrapeCreators fallback |
| `GELBOORU_USER_ID` | No | Gelbooru authenticated API |
| `GELBOORU_API_KEY` | No | Gelbooru authenticated API |
| `KLIPY_API_KEY` | No | For sticker search |

### Orchestrator (Render)
| Variable | Required | Description |
|----------|----------|-------------|
| `RENDER_SERVICE_URL` | Yes | URL of the Go service on Render |
| `CODESPACE_NAME` | Yes | From `echo $CODESPACE_NAME` in Codespace |
| `GITHUB_TOKEN` | Yes | PAT with Codespaces read/write scope |
| `CODESPACE_SERVICE_URL` | Yes | Cloudflare tunnel URL — update after each wake |
| `UPSTASH_REDIS_URL` | Yes | From Upstash console |
| `UPSTASH_REDIS_TOKEN` | Yes | From Upstash console |

### Go Service (Codespace)
| Variable | Set by | Description |
|----------|--------|-------------|
| `MODE` | `start.sh` | Automatically set to `codespace` |
| `PORT` | `start.sh` | Automatically set to `7860` |
| `CHROME_PATH` | Optional | Default `/usr/bin/chromium-browser` |

---

## Troubleshooting

**Codespace not waking up?**
- Check that `CODESPACE_NAME` is correct (`echo $CODESPACE_NAME` in the terminal)
- Check that your GitHub PAT has `codespace` read/write permissions
- Check Render logs for the wake error message

**Tunnel URL stopped working?**
- The tunnel URL changes every time the Codespace restarts
- Update `CODESPACE_SERVICE_URL` in Render env after each wake

**Cache not working?**
- Check Upstash console to see if keys are being created
- Make sure `UPSTASH_REDIS_URL` includes `https://`

**Chrome not launching in Codespace?**
- Run `ls /usr/bin/chromium-browser` — if missing, re-run setup: `bash .devcontainer/setup.sh`

**Binary missing after Codespace restart?**
- The binary is in `/workspaces/Bot_genaration/image-service`
- If missing, `start.sh` auto-rebuilds it

---

## File Structure

```
/
├── main.go                        ← MODE-aware route registration
├── .devcontainer/
│   ├── devcontainer.json          ← Codespace config
│   ├── setup.sh                   ← Runs ONCE on first create
│   └── start.sh                   ← Runs on every wake
├── orchestrator/
│   ├── index.js                   ← 24/7 Render middleman
│   └── package.json
├── node-client.js                 ← Drop into your bot's core/ folder
├── pkg/
│   ├── scraper/                   ← scraper.go = Render-safe HTTP scrapers
│   │                                 Others = Chrome (Codespace only)
│   ├── combat/
│   ├── cards/
│   ├── ludo/
│   ├── ttt/
│   ├── chess/
│   └── utils/
└── HYBRID_ARCHITECTURE.md         ← This file
```
