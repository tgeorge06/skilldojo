// Renders every roster creature into a static HTML sheet and screenshots it,
// so the art can be judged as a set. Dev-only: node scripts/preview-kata.mjs
import { readFileSync, writeFileSync, mkdtempSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { createRequire } from "node:module";
import { chromium } from "playwright";

const require = createRequire(import.meta.url);
const KataSVG = require("../static/kata-svg.js");
const roster = JSON.parse(readFileSync(new URL("../internal/kata/kata.json", import.meta.url), "utf8")).creatures;
const mode = process.argv[2] || "caught"; // caught | half | unknown | evolved
const out = process.argv[3] || join(mkdtempSync(join(tmpdir(), "kata-")), `kata-${mode}.png`);

let cards = "";
for (const c of roster) {
  const fills = mode === "unknown" ? 0 : mode === "half" ? Math.floor(c.regions / 2) : c.regions;
  const svg = KataSVG.creature(c.seed, { palette: c.palette, regions: c.regions, fills, evolved: mode === "evolved", name: c.name, size: 150, silhouette: mode === "unknown" });
  cards += `<div class="card">${svg}<p>${c.name}<br><small>${c.id}</small></p></div>`;
}
const html = `<!doctype html><meta charset="utf-8"><style>
body{margin:0;background:#f1f5f9;font:12px ui-sans-serif,system-ui}
.grid{display:grid;grid-template-columns:repeat(9,170px);gap:8px;padding:12px}
.card{background:#fff;border-radius:12px;padding:6px;text-align:center}
.card p{margin:2px 0 0;font-weight:700}.card small{font-weight:400;color:#64748b}
</style><div class="grid">${cards}</div>`;
const page = await (await chromium.launch()).newPage({ viewport: { width: 1620, height: 1400 }, deviceScaleFactor: 1 });
await page.setContent(html);
await page.screenshot({ path: out, fullPage: true });
console.log(out);
process.exit(0);
