const express = require('express');
const puppeteer = require('puppeteer-core');
require('dotenv').config();

const app = express();
const port = process.env.SCRAPER_PORT || 7861;

let browser;

async function getBrowser() {
    if (browser && browser.connected) return browser;

    const token = process.env.BROWSERLESS_TOKEN;
    if (token) {
        console.log('[SCRAPER] Connecting to Browserless.io...');
        browser = await puppeteer.connect({
            browserWSEndpoint: `wss://production-sfo.browserless.io/chromium?token=${token}`,
            defaultViewport: null
        });
    } else {
        console.log('[SCRAPER] Launching local Chromium...');
        browser = await puppeteer.launch({
            executablePath: process.env.CHROME_PATH || '/usr/bin/google-chrome-stable',
            headless: 'new',
            args: ['--no-sandbox', '--disable-setuid-sandbox', '--disable-dev-shm-usage']
        });
    }
    return browser;
}

// Sets a realistic user agent + headers on every page to avoid bot detection
async function setupPage(browser) {
    const page = await browser.newPage();
    await page.setUserAgent(
        'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36'
    );
    await page.setExtraHTTPHeaders({
        'Accept-Language': 'en-US,en;q=0.9',
        'Accept': 'text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8',
        'sec-ch-ua': '"Chromium";v="124", "Google Chrome";v="124"',
        'sec-ch-ua-mobile': '?0',
        'sec-ch-ua-platform': '"Windows"',
    });
    return page;
}

app.get('/health', (req, res) => {
    res.json({ status: 'ready', engine: 'puppeteer' });
});

// API Router for proxied requests
const api = express.Router();
app.use('/api/scrape', api);

// ── Pinterest Scraper ──────────────────────────────────────────────────
api.get('/pinterest', async (req, res) => {
    const { query, count = 10 } = req.query;
    if (!query) return res.status(400).json({ error: 'Query required' });

    try {
        const b = await getBrowser();
        const page = await setupPage(b);
        const searchURL = `https://www.pinterest.com/search/pins/?q=${encodeURIComponent(query)}`;
        
        console.log(`[Pinterest] Searching: ${searchURL}`);
        await page.goto(searchURL, { waitUntil: 'networkidle2', timeout: 60000 });

        // Scroll loop
        const scrolls = Math.min(Math.max(Math.floor(count / 10), 2), 5);
        for (let i = 0; i < scrolls; i++) {
            await page.evaluate(() => window.scrollBy(0, 1000));
            await new Promise(r => setTimeout(r, 1000));
        }

        const images = await page.evaluate((maxCount) => {
            const nodes = document.querySelectorAll('div[data-test-id="pinWrapper"] img');
            const seen = new Set();
            const urls = [];
            
            for (const img of nodes) {
                let src = img.src;
                if (src && src.includes('pinimg.com')) {
                    const hdURL = src.replace(/(236x|474x)/, '736x');
                    if (!seen.has(hdURL)) {
                        seen.add(hdURL);
                        urls.push(hdURL);
                    }
                }
                if (urls.length >= maxCount) break;
            }
            return urls;
        }, parseInt(count));

        await page.close();
        res.json({ images, count: images.length });
    } catch (err) {
        console.error('[Pinterest] Error:', err.message);
        res.status(500).json({ error: err.message });
    }
});

// ── PornPics Scraper ──────────────────────────────────────────────────
// Uses direct HTTP fetch — site blocks Chrome from datacenter IPs but
// accepts plain requests with proper headers.
api.get('/pornpics', async (req, res) => {
    const { query, count = 10 } = req.query;
    if (!query) return res.status(400).json({ error: 'Query required' });

    const maxCount = parseInt(count);
    const searchURL = `https://www.pornpics.com/?q=${encodeURIComponent(query)}`;
    console.log(`[PornPics] Fetching: ${searchURL}`);

    try {
        const response = await fetch(searchURL, {
            headers: {
                'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36',
                'Accept': 'text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8',
                'Accept-Language': 'en-US,en;q=0.9',
                'Cache-Control': 'no-cache',
            },
            signal: AbortSignal.timeout(30000),
        });

        if (!response.ok) throw new Error(`HTTP ${response.status}`);

        const html = await response.text();
        const images = [];
        const seen = new Set();

        // pornpics stores image URLs in data-src on lazy-loaded imgs
        const regex = /data-src="(https?:\/\/[^"]+\.(?:jpg|jpeg|png))"/gi;
        let match;
        while ((match = regex.exec(html)) !== null) {
            const url = match[1];
            if (!seen.has(url) && !url.includes('logo') && !url.includes('icon')) {
                seen.add(url);
                images.push(url);
            }
            if (images.length >= maxCount) break;
        }

        // Fallback: src attributes
        if (images.length === 0) {
            const fallback = /src="(https?:\/\/[^"]+cdn[^"]+\.(?:jpg|jpeg))"/gi;
            while ((match = fallback.exec(html)) !== null) {
                const url = match[1];
                if (!seen.has(url)) { seen.add(url); images.push(url); }
                if (images.length >= maxCount) break;
            }
        }

        console.log(`[PornPics] Found ${images.length} images`);
        res.json({ images, count: images.length });
    } catch (err) {
        console.error('[PornPics] Error:', err.message);
        res.status(500).json({ error: err.message });
    }
});

// ── Rule34 Deep Scraper ──────────────────────────────────────────────────
api.get(['/rule34', '/rule34/deep'], async (req, res) => {
    const { query, count = 10 } = req.query;
    if (!query) return res.status(400).json({ error: 'Query required' });

    try {
        const b = await getBrowser();
        const page = await setupPage(b);
        const tag = query.trim().replace(/\s+/g, '_');
        const searchURL = `https://rule34.xxx/index.php?page=post&s=list&tags=${encodeURIComponent(tag)}`;
        
        console.log(`[Rule34] Deep Scrape: ${searchURL}`);
        await page.goto(searchURL, { waitUntil: 'networkidle2', timeout: 60000 });

        const postURLs = await page.evaluate((maxCount) => {
            const links = document.querySelectorAll('.thumb a');
            return Array.from(links)
                .slice(0, maxCount)
                .map(l => l.href);
        }, parseInt(count));

        const images = [];
        for (const postURL of postURLs) {
            await page.goto(postURL, { waitUntil: 'networkidle2', timeout: 30000 });
            const src = await page.evaluate(() => {
                const img = document.querySelector('#image');
                if (img) return img.src;
                const vid = document.querySelector('video source');
                if (vid) return vid.src;
                const meta = document.querySelector('meta[property="og:image"]');
                if (meta) return meta.content;
                return null;
            });
            if (src) images.push(src);
        }

        await page.close();
        res.json({ images, count: images.length });
    } catch (err) {
        console.error('[Rule34] Error:', err.message);
        res.status(500).json({ error: err.message });
    }
});

// ── Powerscale Search ──────────────────────────────────────────────────
// Uses Fandom's MediaWiki API instead of scraping the search page —
// much more reliable, no browser needed, returns clean JSON.
api.get('/powerscale', async (req, res) => {
    const { query } = req.query;
    if (!query) return res.status(400).json({ error: 'Query required' });

    console.log(`[Powerscale] Searching API for: ${query}`);

    try {
        const apiURL = `https://vsbattles.fandom.com/api.php?action=opensearch&search=${encodeURIComponent(query)}&limit=10&namespace=0&format=json`;
        const response = await fetch(apiURL, {
            headers: {
                'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36',
                'Accept': 'application/json',
            },
            signal: AbortSignal.timeout(15000),
        });

        if (!response.ok) throw new Error(`Fandom API returned HTTP ${response.status}`);

        // opensearch format: [query, [titles], [descriptions], [urls]]
        const data = await response.json();
        const titles = data[1] || [];
        const urls = data[3] || [];

        const junk = ['Special:', 'Category:', 'Talk:', 'User:', 'File:', 'Template:'];
        const characters = titles
            .map((name, i) => ({ id: i + 1, name, url: urls[i] }))
            .filter(c => c.url && c.url.includes('/wiki/') && !junk.some(j => c.url.includes(j)));

        console.log(`[Powerscale] Found ${characters.length} results`);

        if (characters.length === 0) {
            return res.status(404).json({ error: `No results found for '${query}'` });
        }

        res.json({ characters });
    } catch (err) {
        console.error('[Powerscale] Error:', err.message);
        res.status(500).json({ error: err.message });
    }
});

// ── Powerscale Fetch ──────────────────────────────────────────────────
api.get('/powerscale/fetch', async (req, res) => {
    const { url } = req.query;
    if (!url) return res.status(400).json({ error: 'URL required' });

    try {
        const b = await getBrowser();
        const page = await setupPage(b);
        await page.goto(url, { waitUntil: 'networkidle2', timeout: 60000 });
        
        await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight / 2));
        await new Promise(r => setTimeout(r, 2000));

        const data = await page.evaluate((pageURL) => {
            // Image
            let imageURL = '';
            const imgEl = document.querySelector('img.pi-image-thumbnail');
            if (imgEl) {
                imageURL = imgEl.getAttribute('data-src') || imgEl.src;
                if (imageURL && imageURL.includes('/revision/')) {
                    imageURL = imageURL.split('/revision/')[0];
                }
            }
            if (!imageURL) {
                const articleImgs = document.querySelectorAll('#mw-content-text img');
                for (const img of articleImgs) {
                    const src = img.src;
                    if (src && src.includes('static.wikia.nocookie.net')) {
                        imageURL = src.split('/revision/')[0];
                        break;
                    }
                }
            }

            // Summary
            let summary = '';
            const firstP = document.querySelector('#mw-content-text p');
            if (firstP) {
                summary = firstP.innerText.trim();
                if (summary.length > 400) summary = summary.substring(0, 400) + '...';
            }

            // Stats
            const stats = {};
            const statFields = ["Tier", "Attack Potency", "Speed", "Durability", "Stamina", "Range", "Striking Strength", "Lifting Strength", "Intelligence", "Standard Equipment"];
            const pageText = document.querySelector('#mw-content-text').innerText;
            
            for (const field of statFields) {
                const re = new RegExp(`${field}\\s*:\\s*(.+)`, 'i');
                const match = pageText.match(re);
                if (match) {
                    let val = match[1].split('\n')[0].trim();
                    val = val.replace(/\[[^\]]+\]/g, '').replace(/\([^)]+\)/g, '').split('|').pop().trim();
                    if (val && val !== 'N/A' && val.length < 300) {
                        stats[field] = val;
                    }
                }
            }

            // Name
            const h1 = document.querySelector('h1.page-header__title') || document.querySelector('#firstHeading');
            const name = h1 ? h1.innerText.trim() : '';

            return { name, imageUrl: imageURL, summary, stats, pageUrl: pageURL };
        }, url);

        await page.close();
        res.json(data);
    } catch (err) {
        console.error('[PowerscaleFetch] Error:', err.message);
        res.status(500).json({ error: err.message });
    }
});

// ── Anikai Scraper ──────────────────────────────────────────────────
api.get('/anikai', async (req, res) => {
    const { title } = req.query;
    if (!title) return res.status(400).json({ error: 'Title required' });

    try {
        const b = await getBrowser();
        const page = await setupPage(b);
        const searchURL = `https://anikai.to/browser?keyword=${encodeURIComponent(title)}`;
        
        await page.goto(searchURL, { waitUntil: 'networkidle2', timeout: 60000 });

        const watchLink = await page.evaluate((fallback) => {
            const link = document.querySelector('a[href*="/watch/"]');
            if (!link) return fallback;
            let href = link.getAttribute('href');
            if (href.startsWith('/')) href = 'https://anikai.to' + href;
            return href + '#ep=1';
        }, searchURL);

        await page.close();
        res.json({ watchLink });
    } catch (err) {
        console.error('[Anikai] Error:', err.message);
        res.status(500).json({ error: err.message });
    }
});

// ── Anime News Scraper ──────────────────────────────────────────────────
api.get('/news', async (req, res) => {
    try {
        const b = await getBrowser();
        const page = await setupPage(b);
        const newsURL = 'https://animecorner.me/category/anime-news/';
        
        await page.goto(newsURL, { waitUntil: 'networkidle2', timeout: 60000 });

        const articles = await page.evaluate(() => {
            const cards = document.querySelectorAll('article');
            const results = [];
            const limit = 5;

            for (let i = 0; i < Math.min(cards.length, limit); i++) {
                const card = cards[i];
                const titleEl = card.querySelector('h2, h3');
                const linkEl = card.querySelector('a');
                const imgEl = card.querySelector('img');

                if (titleEl && linkEl) {
                    results.push({
                        title: titleEl.innerText.trim(),
                        link: linkEl.href,
                        img: imgEl ? (imgEl.getAttribute('data-src') || imgEl.src) : ''
                    });
                }
            }
            return results;
        });

        await page.close();
        res.json({ articles });
    } catch (err) {
        console.error('[AnimeNews] Error:', err.message);
        res.status(500).json({ error: err.message });
    }
});

app.listen(port, () => {
    console.log(`🚀 Puppeteer Scraper API listening on port ${port}`);
});