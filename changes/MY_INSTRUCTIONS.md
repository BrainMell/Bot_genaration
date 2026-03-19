# My Project Instructions & Decisions

A compiled record of every instruction, decision, and requirement made throughout this project.

---

## 1. The Server Problem

- GitHub Codespaces was being used to host the Go image service
- Port visibility was set to Public but external monitors (UptimeRobot) and bots still could not reach it
- The `chromium-browser` snap trap was hit — fixed by installing real Google Chrome and symlinking it
- Cloudflare tunnel was identified as the solution to bypass GitHub's authentication wall

---

## 2. Core Requirements

- **No credit card** for any hosting service
- **Good RAM** (minimum 4GB) for running headless Chrome (Rod browser engine)
- **Not 24/7** — only needs to be on at intervals, not constantly
- **REST API** — needs a stable public HTTPS URL
- **Scraper-friendly** — must not flag or ban for running headless Chrome or scraping

---

## 3. Architecture Decision

Two servers, one repository:

### Render Server (24/7, lightweight)
- Kept alive by UptimeRobot pinging `/health` every 5 minutes
- Handles all lightweight requests directly with millisecond response
- Acts as the orchestrator/middleman for heavy requests
- Uses Render's free monthly hours (kept in one specific Render account)

### GitHub Codespace (on-demand, heavy)
- Only wakes up when a heavy request comes in
- Runs for approximately 10–20 seconds per job then shuts down immediately
- Stays within the 400 core-hours/month free limit

### Cache Layer (Upstash Redis)
- Both servers share a Redis cache
- After Codespace processes something, result is sent to Render and stored in cache
- If the same request comes in again, Render serves it from cache instantly — Codespace never wakes

---

## 4. Route Split Decision

### Render handles (no Chrome needed):
- `POST /api/combat` — combat image generation
- `POST /api/combat/endscreen`
- `POST /api/ludo` — game board rendering
- `POST /api/ttt`
- `POST /api/ttt/leaderboard`
- `POST /api/chess`
- `POST /api/cards/burn` — FFmpeg only
- `POST /api/cards/convert` — FFmpeg only
- `GET /api/scrape/stickers` — API-based, no Chrome
- `GET /api/scrape/rule34` — Gelbooru DAPI, no Chrome

### Codespace handles (Chrome required):
- `GET /api/scrape/pinterest` — Rod/Chrome scrape
- `GET /api/scrape/pornpics` — Rod/Chrome scrape
- `GET /api/scrape/audio` — yt-dlp download
- `GET /api/scrape/powerscale` — Rod/Chrome VS Battles search
- `GET /api/scrape/powerscale/fetch` — Rod/Chrome VS Battles page
- `GET /api/scrape/anikai` — Rod/Chrome watch link
- `GET /api/scrape/news` — Rod/Chrome anime news
- `GET /api/scrape/rule34/deep` — Rod/Chrome deep scrape
- `POST /api/cards/gif` — heavy FFmpeg multi-input video pipeline

---

## 5. Codespace Boot Speed Instructions

- Enable **Prebuilds** in GitHub repo Settings → Codespaces → Prebuild configurations
- Use `postStartCommand` in `.devcontainer/devcontainer.json` to auto-run `./image-service` on every wake
- Go binary is already compiled — boot time is purely VM wake time (~25–35 seconds)
- `setup.sh` runs once on first create (installs Chrome, yt-dlp, ffmpeg, builds binary)
- `start.sh` runs on every wake (starts service + Cloudflare tunnel, prints URL)

---

## 6. Files Delivered

| File | Purpose |
|------|---------|
| `main.go` | Updated with `MODE=render/codespace/full` route splitting |
| `pkg/scraper/scraper.go` | Cleaned up, Render-safe (no Chrome imports) |
| `.devcontainer/devcontainer.json` | Codespace config, auto port forwarding |
| `.devcontainer/setup.sh` | First-time setup (Chrome, yt-dlp, ffmpeg, build) |
| `.devcontainer/start.sh` | Every-wake startup (service + Cloudflare tunnel) |
| `orchestrator/index.js` | 24/7 Render middleman with Redis cache + Codespace wake/kill |
| `orchestrator/package.json` | Orchestrator dependencies |
| `node-client.js` | Updated client for your bot |
| `HYBRID_ARCHITECTURE.md` | Full setup instructions |

---

## 7. Files Placement in Repo

```
Bot_genaration/
├── .devcontainer/
│   ├── devcontainer.json     ← NEW
│   ├── setup.sh              ← NEW
│   └── start.sh              ← NEW
├── orchestrator/
│   ├── index.js              ← NEW
│   └── package.json          ← NEW
├── pkg/
│   └── scraper/
│       └── scraper.go        ← REPLACE existing
├── main.go                   ← REPLACE existing
├── node-client.js            ← REPLACE existing
└── HYBRID_ARCHITECTURE.md    ← NEW (setup instructions)
```

All other files in `pkg/` (combat, cards, ludo, ttt, chess, utils) are **untouched**.

---

## 8. Environment Variables Needed

### Go Service on Render
```
MODE=render
PORT=7860
RAPIDAPI_KEY=optional
SCRAPE_CREATORS_KEY=optional
GELBOORU_USER_ID=optional
GELBOORU_API_KEY=optional
KLIPY_API_KEY=optional
```

### Orchestrator on Render
```
RENDER_SERVICE_URL=https://your-go-service.onrender.com
CODESPACE_NAME=your-codespace-name
GITHUB_TOKEN=your-github-pat
CODESPACE_SERVICE_URL=https://your-tunnel-url.trycloudflare.com
UPSTASH_REDIS_URL=https://your-redis.upstash.io
UPSTASH_REDIS_TOKEN=your-upstash-token
PORT=3000
```

### Bot .env
```
GO_IMAGE_SERVICE_URL=https://your-orchestrator.onrender.com
```

---

## 9. One Manual Step After Every Codespace Wake

Every time the Codespace restarts, the Cloudflare tunnel URL changes.
The terminal will print:
```
SERVICE URL: https://random-words.trycloudflare.com
```
Go to Render → Orchestrator → Environment → update `CODESPACE_SERVICE_URL` with the new URL.

---

## 10. Keep Render Alive

- Set up UptimeRobot (free, no card)
- Monitor URL: `https://your-orchestrator.onrender.com/health`
- Interval: every 5 minutes
- This keeps Render within its free monthly hours without sleeping
