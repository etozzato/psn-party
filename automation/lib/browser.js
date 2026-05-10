const fs = require("fs");
const path = require("path");

const chromeCandidates = [
  "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
  "/Applications/Chromium.app/Contents/MacOS/Chromium",
  "/Applications/Google Chrome Canary.app/Contents/MacOS/Google Chrome Canary",
  "/usr/bin/google-chrome",
  "/usr/bin/google-chrome-stable",
  "/usr/bin/chromium",
  "/usr/bin/chromium-browser",
  "/app/.chrome-for-testing/chrome-linux64/chrome"
];

async function loadPuppeteer() {
  try {
    return require("puppeteer");
  } catch (_) {
    try {
      return require("puppeteer-core");
    } catch (error) {
      throw new Error(
        "Puppeteer is not installed. Run `cd automation && npm install`, or install puppeteer in the project."
      );
    }
  }
}

function chromeExecutablePath() {
  if (process.env.PUPPETEER_EXECUTABLE_PATH) {
    return process.env.PUPPETEER_EXECUTABLE_PATH;
  }
  return chromeCandidates.find((candidate) => fs.existsSync(candidate));
}

async function getBrowser() {
  const puppeteer = await loadPuppeteer();
  const executablePath = chromeExecutablePath();
  const userDataDir =
    process.env.PUPPETEER_USER_DATA_DIR ||
    path.resolve(__dirname, "..", "output", "browser-data");

  return puppeteer.launch({
    headless: process.env.HEADLESS === "false" ? false : "new",
    executablePath,
    userDataDir,
    pipe: true,
    dumpio: process.env.PUPPETEER_DUMPIO === "1",
    args: [
      "--disable-dev-shm-usage",
      "--disable-setuid-sandbox",
      "--no-first-run",
      "--no-default-browser-check",
      "--no-sandbox"
    ]
  });
}

async function getPage(browser) {
  const page = await browser.newPage();
  page.setDefaultNavigationTimeout(Number(process.env.NAVIGATION_TIMEOUT || 15000));
  await page.setViewport({ width: 1440, height: 920 });
  return page;
}

module.exports = { getBrowser, getPage };
