const puppeteer = require('puppeteer-core');

async function run() {
    console.log("Launching Chromium in HEADFUL mode...");
    const browser = await puppeteer.launch({
        executablePath: '/usr/bin/google-chrome-stable',
        headless: false,
        args: ['--no-sandbox', '--disable-setuid-sandbox']
    });

    try {
        const page = await browser.newPage();
        await page.setUserAgent(
            'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36'
        );
        
        console.log("Navigating to onlymp3.to...");
        await page.goto('https://en.onlymp3.to/', { waitUntil: 'networkidle2', timeout: 60000 });
        
        console.log("Waiting up to 120 seconds for input#txtUrl to appear. Please solve the Cloudflare captcha in the browser window if prompted...");
        await page.waitForSelector('input#txtUrl', { timeout: 120000 });
        
        console.log("Typing YouTube link...");
        await page.type('input#txtUrl', 'https://www.youtube.com/watch?v=JGwWNGJdvx8');
        
        console.log("Clicking Convert...");
        await page.click('button#btnSubmit');
        
        console.log("Waiting for results...");
        await new Promise(r => setTimeout(r, 12000));
        
        console.log("Dumping result HTML section...");
        const html = await page.evaluate(() => {
            const anchors = Array.from(document.querySelectorAll('a'));
            const downloadAnchors = anchors.map(a => {
                const text = a.innerText || a.textContent || "";
                const href = a.getAttribute('href') || "";
                const id = a.id || "";
                const className = a.className || "";
                return { text: text.trim(), href, id, className };
            }).filter(a => {
                const t = a.text.toLowerCase();
                const h = a.href.toLowerCase();
                const i = a.id.toLowerCase();
                return t.includes('download') || h.includes('download') || i.includes('download') || h.includes('mp3') || t.includes('convert');
            });
            
            const showres = document.querySelector('#showres, .showres, #result, .result, #download, .download') || document.body;
            return {
                downloadAnchors,
                bodyText: document.body.innerText.substring(0, 1000),
                containerHTML: showres ? showres.outerHTML.substring(0, 4000) : 'not found'
            };
        });
        
        console.log("Download Anchors found:", html.downloadAnchors);
        console.log("\nContainer HTML:", html.containerHTML);
        
    } catch (err) {
        console.error("Error during execution:", err.message);
    } finally {
        await browser.close();
        console.log("Browser closed.");
    }
}

run();
