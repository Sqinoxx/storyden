// Logged-in admin performance check against production.
// Usage (PowerShell):
//   $env:STORYDEN_USER="..."; $env:STORYDEN_PASS="..."
//   node .claude\skills\release-ubuntu\perf-check.mjs F:\Zahnmedizin\Forum\storyden\web
//   Remove-Item Env:STORYDEN_USER, Env:STORYDEN_PASS
import { createRequire } from "module";

const webDir = process.argv[2];
const base = process.argv[3] || "https://unidentists.de";
const user = process.env.STORYDEN_USER;
const pass = process.env.STORYDEN_PASS;
if (!webDir || !user || !pass) {
  console.error("usage: STORYDEN_USER=.. STORYDEN_PASS=.. node perf-check.mjs <web-dir> [base-url]");
  process.exit(2);
}

const require = createRequire(webDir + "/");
const { chromium } = require("@playwright/test");

const browser = await chromium.launch();
const ctx = await browser.newContext({ viewport: { width: 1366, height: 900 } });
const page = await ctx.newPage();

await page.goto(base + "/login");
await page.locator('input[name="identifier"]').first().fill(user);
const pw = page.locator('input[name="password"], input[name="token"]').first();
await pw.fill(pass);
await pw.press("Enter");
await page.waitForURL((u) => !u.pathname.startsWith("/login"), { timeout: 30000 });

await page.addInitScript(() => {
  window.__lt = [];
  new PerformanceObserver((l) => l.getEntries().forEach((e) => window.__lt.push(Math.round(e.duration)))).observe({
    type: "longtask",
    buffered: true,
  });
});
const cdp = await ctx.newCDPSession(page);
await cdp.send("Performance.enable");

let bad = false;
for (const path of ["/", "/d", "/"]) {
  const t0 = Date.now();
  await page.goto(base + path, { waitUntil: "load", timeout: 90000 });
  await page.waitForLoadState("networkidle", { timeout: 90000 }).catch(() => {});
  const idle = Date.now() - t0;
  await page.waitForTimeout(2000);
  const r = await page.evaluate(() => {
    const lt = window.__lt || [];
    const portals = [...document.body.children].filter((c) => c.getAttribute("data-scope")).length;
    const groups = {};
    document.querySelectorAll("[data-part=trigger-item]").forEach((e) => {
      const k = e.textContent.trim().slice(0, 30);
      groups[k] = (groups[k] || 0) + 1;
    });
    return {
      dom: document.getElementsByTagName("*").length,
      portals,
      submenuTriggers: groups,
      ltSum: lt.reduce((a, b) => a + b, 0),
      ltMax: Math.max(0, ...lt),
    };
  });
  const heap = (await cdp.send("Performance.getMetrics")).metrics.find((m) => m.name === "JSHeapUsedSize").value;
  const flag = r.dom > 10000 || r.ltMax > 1000 || r.portals > 300 || heap > 150e6;
  bad ||= flag;
  console.log(
    `${flag ? "BAD " : "ok  "}${path}: networkidle ${idle}ms, DOM ${r.dom}, body portals ${r.portals}, ` +
      `longtasks sum ${r.ltSum}ms max ${r.ltMax}ms, heap ${(heap / 1e6).toFixed(0)}MB, ` +
      `pre-rendered submenus ${JSON.stringify(r.submenuTriggers)}`,
  );
}

await browser.close();
console.log(bad ? "RESULT: SLOW - investigate before declaring the release done" : "RESULT: OK");
process.exit(bad ? 1 : 0);
