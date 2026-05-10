const { getBrowser, getPage } = require("./lib/browser");

const baseURL = process.env.BASE_URL || "http://localhost:8890";
const groupURL = process.env.GROUP_URL || "";
const testName = process.env.TEST_NAME || `Automation ${Date.now()}`;
const testPSN = process.env.TEST_PSN || `Auto${String(Date.now()).slice(-8)}`;
const mutate = process.env.MUTATE === "1";
const isPrivate = process.env.PRIVATE === "1";
const csvTextEnv = process.env.CSV_TEXT || "";

function log(event, data = {}) {
  console.log(
    JSON.stringify({ event, time: new Date().toISOString(), ...data }),
  );
}

async function createGroup(page) {
  await page.goto(`${baseURL}/new`, { waitUntil: "networkidle0" });
  // Fill group name
  const groupName = `Automation ${Date.now()}`;
  await page.focus('input[name="name"]');
  await page.keyboard.type(groupName);
  // Set visibility to private if requested
  if (isPrivate) {
    await page.click('input[name="visibility"][value="private"]');
  }
  // Submit
  await Promise.all([
    page.waitForNavigation({ waitUntil: "networkidle0" }),
    page.click('button[type="submit"]'),
  ]);
  // Extract admin URL from the "OPEN ADMIN" link on the created page
  const adminHref = await page.$eval("a.button", (node) => node.href);
  log("created_group", { adminHref, groupName, private: isPrivate });
  return adminHref;
}

async function inspectAddForm(page, targetURL) {
  await page.goto(targetURL, { waitUntil: "networkidle0" });
  const form = await page.$eval("#new-entry form", (node) => ({
    method: node.method,
    action: node.action,
    enctype: node.enctype,
    inputNames: Array.from(
      node.querySelectorAll("input, textarea, button"),
    ).map((el) => ({
      tag: el.tagName.toLowerCase(),
      type: el.getAttribute("type") || "",
      name: el.getAttribute("name") || "",
      value: el.value || "",
      required: el.required || false,
    })),
  }));
  log("add_form", form);
}

async function submitAddForm(page) {
  const posts = [];
  page.on("request", (request) => {
    if (request.method() === "POST") {
      posts.push({
        url: request.url(),
        method: request.method(),
        postData: request.postData() || "",
        headers: request.headers(),
      });
    }
  });

  await page.focus('input[name="display_name"]');
  await page.keyboard.type(testName);
  await page.focus('input[name="online_id"]');
  await page.keyboard.type(testPSN);

  const beforeSubmit = await page.$eval("#new-entry form", (form) => ({
    displayName: form.querySelector('[name="display_name"]')?.value || "",
    onlineID: form.querySelector('[name="online_id"]')?.value || "",
    valid: form.checkValidity(),
  }));
  log("before_submit", beforeSubmit);

  await Promise.all([
    page.waitForNavigation({ waitUntil: "networkidle0" }).catch((error) => {
      log("navigation_error", { message: error.message });
    }),
    page.click("#new-entry button[type='submit']"),
  ]);

  const pageText = await page.evaluate(() => document.body.innerText);
  const hasOpenAdminButton = await page
    .$eval(".action-row a.button", (node) =>
      node.textContent.includes("OPEN ADMIN"),
    )
    .catch(() => false);
  log("posted_requests", { posts });
  log("after_submit", {
    url: page.url(),
    hasEntryAdmin: pageText.includes("ENTRY ADMIN"),
    hasOpenAdminButton,
    errorLine:
      pageText
        .split("\n")
        .find((line) => line.includes("PSN ID") || line.includes("Received")) ||
      "",
  });
}

// new function to upload CSV
async function uploadCSV(page, adminHref) {
  if (!csvTextEnv) {
    log("csv_skip", { message: "No CSV_TEXT provided." });
    return;
  }
  try {
    const urlObj = new URL(adminHref);
    const uploadUrl = `${urlObj.origin}${urlObj.pathname}/upload${urlObj.search}`;
    await page.goto(uploadUrl, { waitUntil: "networkidle0" });
    await page.focus('textarea[name="csv_text"]');
    await page.keyboard.type(csvTextEnv);
    log("csv_before_submit", { lines: csvTextEnv.split(/\r?\n/).length });
    await Promise.all([
      page.waitForNavigation({ waitUntil: "networkidle0" }).catch((error) => {
        log("csv_navigation_error", { message: error.message });
      }),
      page.click('form button[type="submit"]'),
    ]);
    const pageText = await page.evaluate(() => document.body.innerText);
    log("csv_after_submit", {
      hasResult: pageText.includes("RESULT") || pageText.includes("RESULTS"),
      errorLine:
        pageText
          .split("\n")
          .find(
            (line) =>
              line.toUpperCase().includes("CSV") && line.includes("error"),
          ) || "",
    });
  } catch (err) {
    log("csv_error", { message: err.message });
  }
}

async function main() {
  if (!mutate) {
    log("dry_run", {
      message:
        "Set MUTATE=1 to submit forms. This mode only inspects the add form.",
    });
  }

  const browser = await getBrowser();
  const page = await getPage(browser);
  let adminHref = groupURL || "";
  try {
    if (!groupURL && mutate) {
      adminHref = await createGroup(page);
    }
    const targetURL = adminHref || `${baseURL}/new`;
    if (!adminHref && !mutate) {
      await page.goto(targetURL, { waitUntil: "networkidle0" });
      log("new_page_loaded", { url: page.url(), title: await page.title() });
      return;
    }

    await inspectAddForm(page, targetURL);
    if (mutate) {
      await submitAddForm(page);
      if (adminHref) {
        await uploadCSV(page, adminHref);
        await page.goto(targetURL, { waitUntil: "networkidle0" });
      }
    }
  } finally {
    if (process.env.CLOSE === "true") {
      await browser.close();
    }
  }
}

main().catch((error) => {
  log("fatal", { message: error.message, stack: error.stack });
  process.exit(1);
});
