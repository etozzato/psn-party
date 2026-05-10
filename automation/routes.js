const fs = require("fs");
const path = require("path");

const repoRoot = path.resolve(__dirname, "..");
const target = process.argv[2] || "internal/handlers/handlers.go";
const file = path.resolve(repoRoot, target);
const source = fs.readFileSync(file, "utf8");

const routePattern = /\br\.(GET|POST|PUT|PATCH|DELETE)\("([^"]+)"/g;
const routes = [];
let match;
while ((match = routePattern.exec(source))) {
  routes.push({ method: match[1], path: match[2] });
}

if (process.argv.includes("--json")) {
  console.log(JSON.stringify(routes, null, 2));
} else {
  const width = Math.max(...routes.map((route) => route.method.length), 6);
  for (const route of routes) {
    console.log(`${route.method.padEnd(width)} ${route.path}`);
  }
}
