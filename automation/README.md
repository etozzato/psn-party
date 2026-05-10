# Automation Sandbox

Small, portable browser/debug harness for server-rendered apps.

## Route Discovery

```bash
node automation/routes.js
node automation/routes.js --json
```

This scans Gin route registrations in `internal/handlers/handlers.go`.

## Browser Form Debug

Install Puppeteer once:

```bash
cd automation
npm install
```

Inspect a page without mutating data:

```bash
BASE_URL=http://localhost:8890 npm run inspect:add
```

Submit through a real browser:

```bash
GROUP_URL='http://localhost:8890/g/<slug>?admin=<token>' \
TEST_NAME='Browser Smoke' \
TEST_PSN='BrowserSmoke123' \
npm run smoke:add
```

The script logs:

- rendered form method/action/enctype
- inputs in the form
- values immediately before submit
- actual browser POST request body
- rendered response state

Use a throwaway group or a unique PSN-like test ID when `MUTATE=1`.
