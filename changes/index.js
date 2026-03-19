/**
 * orchestrator/index.js
 * 
 * 24/7 Render server — the middleman between your bot and the Codespace.
 * 
 * Responsibilities:
 *  1. Handle all lightweight requests directly (combat, boards, cards/burn, stickers, rule34)
 *  2. For heavy requests (scrape/pinterest, scrape/pornpics, scrape/audio,
 *     scrape/powerscale, scrape/anikai, scrape/news, cards/gif):
 *     a. Check Redis cache — if hit, return instantly
 *     b. Wake the Codespace via GitHub API
 *     c. Wait until it's healthy
 *     d. Forward the request
 *     e. Cache the result
 *     f. Immediately shut down the Codespace
 */

const express = require('express');
const axios = require('axios');

const app = express();
app.use(express.json({ limit: '50mb' }));

// ── Config ────────────────────────────────────────────────────────────────────
const RENDER_SERVICE_URL   = process.env.RENDER_SERVICE_URL   || 'http://localhost:7860';
const CODESPACE_NAME       = process.env.CODESPACE_NAME;       // e.g. "brainmell-bot-genaration-xxxx"
const GITHUB_TOKEN         = process.env.GITHUB_TOKEN;         // Personal Access Token with codespace scope
const UPSTASH_REDIS_URL    = process.env.UPSTASH_REDIS_URL;    // https://xxx.upstash.io
const UPSTASH_REDIS_TOKEN  = process.env.UPSTASH_REDIS_TOKEN;
const PORT                 = process.env.PORT || 3000;

// How long to cache results (seconds)
const CACHE_TTL = {
  pinterest:  3600,   // 1 hour
  pornpics:   3600,
  rule34:     3600,
  powerscale: 86400,  // 24 hours — character stats rarely change
  anikai:     86400,
  news:       1800,   // 30 minutes — news changes often
  audio:      86400,  // audio URLs are stable
  gif:        86400,
};

// Routes that must go to the Codespace
const HEAVY_ROUTES = [
  '/api/scrape/pinterest',
  '/api/scrape/pornpics',
  '/api/scrape/audio',
  '/api/scrape/powerscale',
  '/api/scrape/anikai',
  '/api/scrape/news',
  '/api/scrape/rule34/deep', // Chrome deep scrape (Codespace)
  '/api/cards/gif',
];

// ── Redis helpers (Upstash REST API) ─────────────────────────────────────────
async function cacheGet(key) {
  if (!UPSTASH_REDIS_URL) return null;
  try {
    const res = await axios.get(`${UPSTASH_REDIS_URL}/get/${encodeURIComponent(key)}`, {
      headers: { Authorization: `Bearer ${UPSTASH_REDIS_TOKEN}` },
    });
    if (res.data.result) {
      return JSON.parse(res.data.result);
    }
  } catch (e) {
    console.error('[Cache] GET error:', e.message);
  }
  return null;
}

async function cacheSet(key, value, ttlSeconds) {
  if (!UPSTASH_REDIS_URL) return;
  try {
    await axios.post(
      `${UPSTASH_REDIS_URL}/set/${encodeURIComponent(key)}`,
      JSON.stringify(value),
      {
        headers: {
          Authorization: `Bearer ${UPSTASH_REDIS_TOKEN}`,
          'Content-Type': 'text/plain',
        },
        params: { ex: ttlSeconds },
      }
    );
  } catch (e) {
    console.error('[Cache] SET error:', e.message);
  }
}

function getCacheKey(path, query, body) {
  const queryStr = new URLSearchParams(query).toString();
  const bodyStr  = body ? JSON.stringify(body) : '';
  return `goservice:${path}:${queryStr}:${bodyStr}`.substring(0, 512);
}

function getTTL(path) {
  for (const [route, ttl] of Object.entries(CACHE_TTL)) {
    if (path.includes(route)) return ttl;
  }
  return 3600;
}

// ── GitHub Codespace API helpers ──────────────────────────────────────────────
async function getCodespaceStatus() {
  const res = await axios.get(
    `https://api.github.com/user/codespaces/${CODESPACE_NAME}`,
    { headers: { Authorization: `token ${GITHUB_TOKEN}`, Accept: 'application/vnd.github.v3+json' } }
  );
  return res.data.state; // 'Available', 'Stopped', 'Starting', 'Stopping'
}

async function startCodespace() {
  console.log('[Codespace] Sending start signal...');
  await axios.post(
    `https://api.github.com/user/codespaces/${CODESPACE_NAME}/start`,
    {},
    { headers: { Authorization: `token ${GITHUB_TOKEN}`, Accept: 'application/vnd.github.v3+json' } }
  );
}

async function stopCodespace() {
  console.log('[Codespace] Sending stop signal...');
  try {
    await axios.post(
      `https://api.github.com/user/codespaces/${CODESPACE_NAME}/stop`,
      {},
      { headers: { Authorization: `token ${GITHUB_TOKEN}`, Accept: 'application/vnd.github.v3+json' } }
    );
    console.log('[Codespace] Stop signal sent.');
  } catch (e) {
    console.error('[Codespace] Stop failed (non-fatal):', e.message);
  }
}

/**
 * Wakes the Codespace, waits until the /health endpoint responds OK.
 * Returns the base URL of the live Codespace service.
 */
async function wakeAndWait() {
  const CODESPACE_SERVICE_URL = process.env.CODESPACE_SERVICE_URL;
  if (!CODESPACE_SERVICE_URL) {
    throw new Error('CODESPACE_SERVICE_URL not set in environment. Update it after each Codespace wake.');
  }

  const status = await getCodespaceStatus();
  console.log(`[Codespace] Current status: ${status}`);

  if (status !== 'Available') {
    await startCodespace();

    // Wait up to 90 seconds for the VM to be 'Available'
    for (let i = 0; i < 45; i++) {
      await sleep(2000);
      const s = await getCodespaceStatus();
      console.log(`[Codespace] Waiting for Available... (${s})`);
      if (s === 'Available') break;
    }
  }

  // Now wait for the Go service inside the Codespace to be healthy
  console.log('[Codespace] Waiting for Go service health...');
  for (let i = 0; i < 30; i++) {
    try {
      const r = await axios.get(`${CODESPACE_SERVICE_URL}/health`, { timeout: 3000 });
      if (r.data.status === 'ok') {
        console.log('[Codespace] Go service is healthy!');
        return CODESPACE_SERVICE_URL;
      }
    } catch (_) {}
    await sleep(2000);
  }

  throw new Error('Codespace Go service did not become healthy in time.');
}

// ── Proxy helper ──────────────────────────────────────────────────────────────
async function proxyRequest(baseURL, path, method, query, body, res) {
  const url = `${baseURL}${path}`;
  const config = {
    method,
    url,
    params: query,
    data: body,
    responseType: 'arraybuffer', // handles both JSON and binary (images/video)
    timeout: 120000,
  };

  const response = await axios(config);
  const contentType = response.headers['content-type'] || 'application/json';

  res.set('Content-Type', contentType);
  res.status(response.status).send(Buffer.from(response.data));
  return { contentType, data: response.data };
}

// ── Main proxy handler ────────────────────────────────────────────────────────
app.all('/api/*', async (req, res) => {
  const path   = req.path;
  const method = req.method.toLowerCase();
  const query  = req.query;
  const body   = req.body;

  const isHeavy = HEAVY_ROUTES.some(r => path.startsWith(r));

  // ── Lightweight: proxy straight to Render's own Go service ──────────────
  if (!isHeavy) {
    try {
      await proxyRequest(RENDER_SERVICE_URL, path, method, query, body, res);
    } catch (e) {
      console.error('[Render] Proxy error:', e.message);
      if (!res.headersSent) res.status(502).json({ error: 'Render service unavailable', details: e.message });
    }
    return;
  }

  // ── Heavy: check cache first ──────────────────────────────────────────────
  const cacheKey = getCacheKey(path, query, body);
  const cached   = await cacheGet(cacheKey);

  if (cached) {
    console.log(`[Cache] HIT for ${path}`);
    const contentType = cached.contentType || 'application/json';
    res.set('Content-Type', contentType);
    // If it was binary (image/video), it was base64-encoded before caching
    if (contentType.startsWith('image/') || contentType.startsWith('video/')) {
      res.send(Buffer.from(cached.data, 'base64'));
    } else {
      res.json(cached.data);
    }
    return;
  }

  // ── Heavy: wake Codespace, forward, cache, then shut it down ─────────────
  console.log(`[Heavy] Cache MISS for ${path} — waking Codespace...`);
  let codespaceURL;

  try {
    codespaceURL = await wakeAndWait();
  } catch (e) {
    console.error('[Codespace] Wake failed:', e.message);
    return res.status(503).json({ error: 'Codespace failed to start', details: e.message });
  }

  try {
    const axiosConfig = {
      method,
      url: `${codespaceURL}${path}`,
      params: query,
      data: body,
      responseType: 'arraybuffer',
      timeout: 120000,
    };

    const response = await axios(axiosConfig);
    const contentType = response.headers['content-type'] || 'application/json';
    const rawData = Buffer.from(response.data);

    // Send to client immediately
    res.set('Content-Type', contentType);
    res.status(response.status).send(rawData);

    // Cache in background (don't block the response)
    const ttl = getTTL(path);
    if (contentType.startsWith('image/') || contentType.startsWith('video/')) {
      cacheSet(cacheKey, { contentType, data: rawData.toString('base64') }, ttl);
    } else {
      try {
        const parsed = JSON.parse(rawData.toString());
        cacheSet(cacheKey, { contentType, data: parsed }, ttl);
      } catch (_) {}
    }

  } catch (e) {
    console.error('[Codespace] Forward error:', e.message);
    if (!res.headersSent) res.status(502).json({ error: 'Codespace request failed', details: e.message });
  } finally {
    // Always shut the Codespace down after the job is done
    if (CODESPACE_NAME && GITHUB_TOKEN) {
      stopCodespace();
    }
  }
});

// ── Health ────────────────────────────────────────────────────────────────────
app.get('/health', (req, res) => res.json({ status: 'ok', service: 'orchestrator' }));

app.get('/', (req, res) => res.json({
  status: 'online',
  service: 'Orchestrator — Go Image Service Middleman',
  routes: {
    lightweight: 'proxied to Render Go service directly',
    heavy: 'wakes Codespace → forwards → caches → shuts down',
  },
}));

// ── Helpers ───────────────────────────────────────────────────────────────────
function sleep(ms) { return new Promise(r => setTimeout(r, ms)); }

app.listen(PORT, () => {
  console.log(`🚀 Orchestrator running on port ${PORT}`);
  console.log(`   Render service:    ${RENDER_SERVICE_URL}`);
  console.log(`   Codespace name:    ${CODESPACE_NAME || '(not set)'}`);
  console.log(`   Redis cache:       ${UPSTASH_REDIS_URL ? 'enabled' : 'disabled'}`);
});
