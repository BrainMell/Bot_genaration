/**
 * node-client.js
 * 
 * Drop this in your bot's core/ folder.
 * Point GO_IMAGE_SERVICE_URL at your Render ORCHESTRATOR URL — not the
 * raw Go service. The orchestrator handles routing automatically.
 * 
 * Example .env:
 *   GO_IMAGE_SERVICE_URL=https://your-orchestrator.onrender.com
 */

const axios = require('axios');

class GoImageService {
  constructor(serviceUrl) {
    this.baseUrl = serviceUrl || process.env.GO_IMAGE_SERVICE_URL || 'http://localhost:3000';
    this.client = axios.create({
      baseURL: this.baseUrl,
      timeout: 120000, // 2 min — heavy ops (Codespace wake + process)
      maxBodyLength: Infinity,
      maxContentLength: Infinity,
    });
  }

  async healthCheck() {
    try {
      const res = await this.client.get('/health');
      return res.data;
    } catch (_) {
      return null;
    }
  }

  // ── Lightweight (handled by Render Go service directly) ──────────────────

  async generateCombatImage(data) {
    const res = await this.client.post('/api/combat', data, { responseType: 'arraybuffer' });
    return Buffer.from(res.data);
  }

  async generateEndScreen(data) {
    const res = await this.client.post('/api/combat/endscreen', data, { responseType: 'arraybuffer' });
    return Buffer.from(res.data);
  }

  async renderLudo(data) {
    const res = await this.client.post('/api/ludo', data, { responseType: 'arraybuffer' });
    return Buffer.from(res.data);
  }

  async renderTTT(data) {
    const res = await this.client.post('/api/ttt', data, { responseType: 'arraybuffer' });
    return Buffer.from(res.data);
  }

  async renderTTTLeaderboard(data) {
    const res = await this.client.post('/api/ttt/leaderboard', data, { responseType: 'arraybuffer' });
    return Buffer.from(res.data);
  }

  async renderChess(data) {
    const res = await this.client.post('/api/chess', data, { responseType: 'arraybuffer' });
    return Buffer.from(res.data);
  }

  async burnCard(imageUrl) {
    const res = await this.client.post('/api/cards/burn', { imageUrl }, { responseType: 'arraybuffer' });
    return Buffer.from(res.data);
  }

  async convertCard(imageUrl) {
    const res = await this.client.post('/api/cards/convert', { imageUrl }, { responseType: 'arraybuffer' });
    return Buffer.from(res.data);
  }

  async searchStickers(query) {
    const res = await this.client.get('/api/scrape/stickers', { params: { query } });
    return res.data;
  }

  async searchRule34(query, count = 10) {
    const res = await this.client.get('/api/scrape/rule34', { params: { query, count } });
    return res.data;
  }

  // ── Heavy (wakes Codespace, cached) ──────────────────────────────────────

  /**
   * Pinterest — Chrome scrape, cached 1hr
   * @param {string} query
   * @param {number} count
   */
  async searchPinterest(query, count = 10) {
    const res = await this.client.get('/api/scrape/pinterest', { params: { query, count } });
    return res.data;
  }

  /**
   * PornPics — Chrome scrape, cached 1hr
   */
  async searchPornPics(query, count = 10) {
    const res = await this.client.get('/api/scrape/pornpics', { params: { query, count } });
    return res.data;
  }

  /**
   * Audio — yt-dlp download, cached 24hr
   */
  async searchAudio(query) {
    const res = await this.client.get('/api/scrape/audio', { params: { query } });
    return res.data;
  }

  /**
   * VS Battles powerscale search — Chrome, cached 24hr
   */
  async searchPowerscale(query) {
    const res = await this.client.get('/api/scrape/powerscale', { params: { query } });
    return res.data;
  }

  /**
   * VS Battles powerscale page fetch — Chrome, cached 24hr
   */
  async getPowerscalePage(url) {
    const res = await this.client.get('/api/scrape/powerscale/fetch', { params: { url } });
    return res.data;
  }

  /**
   * Anikai watch link — Chrome, cached 24hr
   */
  async searchAnikai(title) {
    const res = await this.client.get('/api/scrape/anikai', { params: { title } });
    return res.data;
  }

  /**
   * Anime Corner news — Chrome, cached 30min
   */
  async getAnimeNews() {
    const res = await this.client.get('/api/scrape/news');
    return res.data;
  }

  /**
   * Card GIF — heavy FFmpeg pipeline, cached 24hr
   * @param {string[]} images
   * @param {string} title
   */
  async generateCardGif(images, title = '') {
    const res = await this.client.post('/api/cards/gif', { images, title }, { responseType: 'arraybuffer' });
    return Buffer.from(res.data);
  }
}

module.exports = GoImageService;
