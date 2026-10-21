// Screenshots of every admin page and a few overlays (temporary, not committed).
import { chromium } from "@playwright/test";
const out = process.argv[2];
const base = "http://localhost:4173";
const browser = await chromium.launch({ executablePath: "/opt/pw-browsers/chromium-1194/chrome-linux/chrome" });
const page = await browser.newPage({ viewport: { width: 1440, height: 900 } });
const errors = [];
page.on("pageerror", (e) => errors.push(e.message));
page.on("console", (m) => m.type() === "error" && errors.push("console: " + m.text().slice(0, 200)));
await page.goto(`${base}/login`);
await page.waitForTimeout(800);
await page.screenshot({ path: `${out}/00-login.png` });
await page.getByPlaceholder("Username").fill("e2e-admin");
await page.getByPlaceholder("Password").fill("e2e-password-123");
await page.getByRole("button", { name: "Login" }).click();
await page.waitForURL(/dashboard/);
const routes = ["dashboard", "contrib", "contrib/edit/articles", "media", "approvals", "model", "model/collection/articles", "model/collection/new", "workspace", "agents", "agents/new", "agents/e2e-tagger/edit", "activity", "settings/users", "settings/roles", "settings/tokens", "settings/webhooks", "settings/locales"];
let i = 1;
for (const r of routes) {
  await page.goto(`${base}/${r}`);
  await page.waitForLoadState("networkidle").catch(() => {});
  await page.waitForTimeout(900);
  await page.screenshot({ path: `${out}/${String(i++).padStart(2, "0")}-${r.replaceAll("/", "_")}.png`, fullPage: true });
}
// Overlays
const shot = async (name) => { await page.waitForTimeout(700); await page.screenshot({ path: `${out}/${name}.png` }); await page.keyboard.press("Escape"); await page.waitForTimeout(400); };
const tryClick = async (loc) => { try { await loc.first().click({ timeout: 4000 }); return true; } catch { return false; } };
await page.goto(`${base}/contrib`); await page.waitForTimeout(1200);
await tryClick(page.getByText(/^articles$/i)); await page.waitForTimeout(800);
await page.screenshot({ path: `${out}/30-contrib-articles.png`, fullPage: true });
if (await tryClick(page.getByRole("button", { name: "Quick Add" }))) await shot("31-quick-add-modal");
if (await tryClick(page.getByRole("button", { name: "Item history" }))) await shot("32-history-drawer");
await page.goto(`${base}/settings/users`); await page.waitForTimeout(1000);
if (await tryClick(page.getByRole("button", { name: "New user" }))) await shot("33-new-user-modal");
await page.goto(`${base}/settings/locales`); await page.waitForTimeout(1000);
if (await tryClick(page.getByRole("button", { name: "Add locale" }))) await shot("34-add-locale");
await page.goto(`${base}/dashboard`); await page.waitForTimeout(1000);
if (await tryClick(page.getByRole("button", { name: "Account menu" }))) await shot("35-account-menu");
await page.goto(`${base}/agents`); await page.waitForTimeout(1500);
await page.screenshot({ path: `${out}/36-agents-selected.png`, fullPage: true });
console.log(JSON.stringify([...new Set(errors)].slice(0, 30), null, 1));
await browser.close();
