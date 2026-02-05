# 🚀 Go Image & Scraper Microservice

This service offloads heavy image processing and scraping from your Node.js bot to a high-performance Go application.

## 📦 Features

*   **Combat Image Generation:** Uses `fogleman/gg` and `disintegration/imaging` for fast, non-blocking rendering.
*   **Game Boards:** Renders Ludo and Tic-Tac-Toe boards instantly.
*   **Web Scraping:** Includes fast scrapers for Pinterest and VS Battles Wiki using `gocolly`.

## 🛠️ Setup

### 1. Deploy
This service is designed to run on **Render** (as a Web Service) or any Docker-compatible host.

**Docker:**
```bash
docker build -t image-service .
docker run -p 8080:8080 image-service
```

### 2. Environment Variables
Set these in your Node.js bot's `.env`:
```env
GO_IMAGE_SERVICE_URL=https://your-deployed-service.onrender.com
```

### 3. Assets
The service expects an `assets` folder in the working directory containing:
*   `rpgasset/characters`
*   `rpgasset/enemies`
*   `rpgasset/environment`
*   `rpgasset/ui`

*(Ensure you copy your bot's `rpgasset` folder to the service's `assets` folder during deployment).*

## 🔌 API Endpoints

### Images
*   `POST /api/combat` - Generate combat scene
*   `POST /api/ludo` - Render Ludo board
*   `POST /api/ttt` - Render Tic-Tac-Toe board

### Scrapers
*   `GET /api/scrape/pinterest?query=...`
*   `GET /api/scrape/vsbattles/search?query=...`
*   `GET /api/scrape/vsbattles/detail?url=...`

## 💻 Node.js Client
Copy `node-client.js` to your bot's `core` or `modules` folder to easily interact with this service.