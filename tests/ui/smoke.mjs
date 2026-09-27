// Render gate: boot the real binary in -dev mode, drive every page and view
// in headless Chromium, and fail on any page error or console error. Unit
// tests cannot see a template binding to a helper that does not exist; this
// can. Run with `make ui-test`.
import { spawn, execFileSync } from "node:child_process";
import { mkdtempSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import { chromium } from "playwright";

const root = join(dirname(fileURLToPath(import.meta.url)), "..", "..");
const port = 18080 + Math.floor(Math.random() * 1000);
const base = `http://127.0.0.1:${port}`;
const dir = mkdtempSync(join(tmpdir(), "skilldojo-ui-"));
const bin = join(dir, "skilldojo");

execFileSync("go", ["build", "-o", bin, "."], { cwd: root, stdio: "inherit" });
const server = spawn(bin, ["-dev", "-addr", `127.0.0.1:${port}`, "-db", join(dir, "ui.db")], { stdio: ["ignore", "pipe", "pipe"] });
let serverLog = "";
server.stdout.on("data", (d) => (serverLog += d));
server.stderr.on("data", (d) => (serverLog += d));

async function waitForServer() {
  for (let i = 0; i < 100; i += 1) {
    try {
      const res = await fetch(base + "/");
      if (res.ok) return;
    } catch {}
    await new Promise((r) => setTimeout(r, 100));
  }
  throw new Error("server did not start:\n" + serverLog);
}

const problems = [];
function watch(page, label) {
  page.on("pageerror", (err) => problems.push(`${label}: pageerror ${err.message}`));
  page.on("console", (msg) => {
    if (msg.type() === "error") problems.push(`${label}: console.error ${msg.text()}`);
  });
  page.on("response", (res) => {
    if (res.status() >= 500) problems.push(`${label}: ${res.status()} ${res.url()}`);
  });
}

async function expect(cond, message) {
  if (!cond) problems.push(message);
}

let browser;
try {
  await waitForServer();
  browser = await chromium.launch();
  const context = await browser.newContext();
  const page = await context.newPage();
  watch(page, "practice");

  // Anonymous practice: every view renders.
  await page.goto(base + "/");
  await expect(await page.locator("text=Parents").count() === 1, "anonymous header should link to /login");
  await page.getByRole("button", { name: /Math/ }).click();
  await page.getByRole("button", { name: /Start math training/ }).click();
  const answers = page.locator("input[inputmode=numeric]");
  await answers.first().waitFor({ timeout: 5000 });
  const answerCount = await answers.count();
  await expect(answerCount === 10, `math sheet should show 10 inputs, saw ${answerCount}`);
  for (let i = 0; i < answerCount; i += 1) await answers.nth(i).fill("1");
  await page.getByRole("button", { name: /Grade my sheet/ }).click();
  await page.locator("text=/out of/").first().waitFor({ timeout: 5000 });
  await page.getByRole("button", { name: /Choose new training/ }).click();

  await page.getByRole("button", { name: /Spelling/ }).click();
  await page.getByRole("button", { name: /Start word rescue/ }).click();
  await page.locator("#spelling-word-heading").waitFor({ timeout: 5000 });
  for (const letter of ["E", "A", "T"]) {
    await page.locator("button.letter-key", { hasText: new RegExp(`^${letter}$`) }).click();
  }
  await expect(await page.locator("button.letter-key[disabled]").count() === 3, "guessed letters should be disabled");
  const wholeWord = page.locator("#whole-word");
  if (await wholeWord.isVisible() && await wholeWord.isEnabled()) {
    await wholeWord.fill("zzzz");
    await page.getByRole("button", { name: /Rescue word/ }).click();
  }
  await expect(await page.locator("#next-spelling-button, button.letter-key").count() > 0, "spelling view should still be rendered");

  // Parent flow: sign in via the dev link, add a profile, see it on the practice page.
  const parent = await context.newPage();
  watch(parent, "parent");
  await parent.goto(base + "/login");
  await parent.fill("input[name=email]", "ui@example.com");
  await parent.click("button[type=submit]");
  await parent.locator("text=Check your email").waitFor({ timeout: 5000 });
  await parent.click("text=open the sign-in link");
  await parent.locator("text=You are about to sign in as").waitFor({ timeout: 5000 });
  await parent.getByRole("button", { name: /Sign in as/ }).click();
  await parent.locator("text=Who's training?").waitFor({ timeout: 5000 });
  await parent.fill("input[name=nickname]", "Nova");
  await parent.selectOption("select[name=grade]", "2");
  await parent.getByRole("button", { name: "Add", exact: true }).click();
  await parent.locator("text=training now").waitFor({ timeout: 5000 });
  await parent.goto(base + "/");
  await expect(await parent.locator("text=Training: Nova").count() === 1, "practice page should show the active child");
} catch (err) {
  problems.push(`harness: ${err.message}`);
} finally {
  if (browser) await browser.close();
  server.kill();
}

if (problems.length) {
  console.error("UI smoke failed:\n  " + problems.join("\n  "));
  console.error("\nserver log:\n" + serverLog);
  process.exit(1);
}
console.log("UI smoke passed");
