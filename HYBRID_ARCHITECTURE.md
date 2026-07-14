# 🚀 Go Image Service — Hugging Face Hybrid Architecture

## Overview

This service runs in two modes across two platforms:

| Mode | Platform | Handles |
|------|----------|---------|
| `render` | Render.com (24/7) | Combat images, game boards, stickers, rule34 (API), and **Orchestration** |
| `codespace` | Hugging Face Spaces (On-Demand) | Chrome Scrapes (Pinterest, PornPics, etc), yt-dlp, Card GIFs |

The **Render instance** acts as the brain. It handles lightweight requests instantly and automatically wakes/proxies heavy requests to the **Hugging Face Space**.

```
Bot → Render (24/7)
           │
           ├── Lightweight? → Process Locally (ms)
           │
           └── Heavy? → Check Redis cache
                             │
                             ├── Cache HIT  → Return instantly
                             └── Cache MISS → Wake HF Space via API
                                                  → Process
                                                  → Cache result
                                                  → Auto-pause after 5m idle
```

---

## Part 1 — Hugging Face Setup

### Step 1: Create the Space
1. Go to [huggingface.co/new-space](https://huggingface.co/new-space)
2. Select **Docker** as the SDK.
3. Set visibility to **Public** (required for the bot to reach it).
4. Push this repository to the Space.

### Step 2: Get your API Token
1. Go to HF Settings -> [Access Tokens](https://huggingface.co/settings/tokens).
2. Create a token with **Write** permission (needed for auto-pause/resume).

---

## Part 2 — Render Setup

### Step 1: Deploy to Render
1. Create a new **Web Service** on Render.
2. Connect your repo.
3. Set **Runtime** to `Docker` (Render will use the same Dockerfile).
4. Set Environment Variables:
   ```env
   MODE=render
   HF_SPACE_ID=your-username/your-space-name
   HF_SPACE_URL=https://your-username-your-space-name.hf.space
   HF_TOKEN=your-hf-token
   UPSTASH_REDIS_URL=https://your-db.upstash.io
   UPSTASH_REDIS_TOKEN=your-token
   ```

---

## Part 3 — How Automation Works

- **Auto-Wake:** When a heavy request (like Pinterest) hits Render and the cache is empty, Render sends a `resume` signal to Hugging Face and waits for the Space to be healthy before forwarding the request.
- **Auto-Pause:** The HF Space starts a 5-minute timer after every heavy request. If no new requests come in within 5 minutes, it sends itself a `pause` signal to stay within free tier limits.
- **Permanent URL:** Unlike Codespaces, the HF URL never changes. You set it once and forget it.

---

## Part 4 — Connect Your Bot

In your bot's `.env`:
```env
GO_IMAGE_SERVICE_URL=https://your-render-app.onrender.com
```

Use `node-client.js` in your bot's `core/` folder to make requests.
