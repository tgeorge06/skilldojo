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
  const answers = page.locator("input[inputmode=numeric]:visible");
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

  // Signed in, a Word Rescue round goes through the server and earns a reward.
  await parent.getByRole("button", { name: /Spelling/ }).click();
  await parent.getByRole("button", { name: /Start word rescue/ }).click();
  await parent.locator("#spelling-word-heading").waitFor({ timeout: 5000 });
  for (let i = 0; i < 5; i += 1) {
    // Read the answer from component state (a test harness privilege) and
    // rescue it, so the reward is deterministic.
    const word = await parent.evaluate(() => document.querySelector("[x-data]")._x_dataStack[0].currentWord.word);
    await parent.fill("#whole-word", word);
    await parent.getByRole("button", { name: /Rescue word/ }).click();
    await parent.locator("#next-spelling-button").waitFor({ timeout: 5000 });
    await parent.locator("#next-spelling-button").click();
  }
  await parent.locator("#spelling-results-heading").waitFor({ timeout: 5000 });
  await parent.getByText(/\+\d+ mosaic tiles/).locator("visible=true").waitFor({ timeout: 5000 });
  await expect(await parent.locator("text=You rescued 5 of 5").count() === 1, "server should agree all five were rescued");
  await expect(await parent.locator("[aria-label='Kata touched by this round'] svg").count() >= 1, "results should reveal a creature");

  // Five rescued words earned a battle credit: fight, lose every heart, rest.
  await parent.getByRole("button", { name: /Battle!/ }).first().waitFor({ timeout: 5000 });
  await parent.getByRole("button", { name: /Battle!/ }).first().click();
  await parent.locator("#battle-heading").waitFor({ timeout: 5000 });
  await expect(await parent.locator("[aria-label='Fighters'] svg").count() === 2, "two fighters should render");
  for (let i = 0; i < 5; i += 1) {
    await parent.fill("#battle-answer", "zzzz");
    await parent.getByRole("button", { name: /Strike!/ }).click();
    await parent.locator(`[aria-label='Fighters'] [aria-label='${4 - i} of 5 hearts']`).first().waitFor({ timeout: 5000 });
  }
  await parent.locator("text=rests for now").waitFor({ timeout: 5000 });

  // The index renders every creature and Train here starts a round.
  await parent.getByRole("button", { name: /My kata/ }).click();
  await parent.locator("#kata-heading").waitFor({ timeout: 5000 });
  await parent.locator("ul[aria-label=Kata] li svg").first().waitFor({ timeout: 5000 });
  const cards = await parent.locator("ul[aria-label=Kata] li").count();
  await expect(cards === 10 || cards === 11, `grade tab should list its creatures, saw ${cards}`);
  await expect(await parent.locator("[aria-label^='This week']").count() === 1, "the index should show this week's mosaic");
  await parent.locator("ul[aria-label=Kata] li").first().getByRole("button", { name: /Train here/ }).click();
  await parent.locator("#spelling-word-heading, input[inputmode=numeric]").first().waitFor({ timeout: 5000 });

  // Color by number: open a page, pick a region, answer wrong, see it wait.
  await parent.goto(base + "/");
  await parent.getByRole("button", { name: /Math/ }).click();
  await parent.getByRole("button", { name: /Color by number/ }).click();
  await parent.locator("#paint-heading").waitFor({ timeout: 5000 });
  const regions = await parent.locator("svg[aria-label='Color by number page'] g[role=button]").count();
  await expect(regions === 14, `grade-2 page should have 14 regions, saw ${regions}`);
  await parent.locator("svg[aria-label='Color by number page'] g[role=button]").first().click();
  await parent.fill("#paint-answer", "999999");
  await parent.getByRole("button", { name: /Color it/ }).click();
  await parent.locator("text=Not yet").waitFor({ timeout: 5000 });

  // Cooldown painting after a lost round needs no server.
  await parent.goto(base + "/");
  await parent.getByRole("button", { name: /Spelling/ }).click();
  await parent.getByRole("button", { name: /Start word rescue/ }).click();
  await parent.locator("#spelling-word-heading").waitFor({ timeout: 5000 });
  for (let i = 0; i < 5; i += 1) {
    for (let k = 0; k < 6; k += 1) {
      await parent.fill("#whole-word", "zzzz");
      await parent.getByRole("button", { name: /Rescue word/ }).click();
    }
    await parent.locator("#next-spelling-button").click();
  }
  await parent.locator("#spelling-results-heading").waitFor({ timeout: 5000 });
  await parent.getByRole("button", { name: /Paint to relax/ }).click();
  await parent.locator("#cooldown-heading").waitFor({ timeout: 5000 });
  const spot = parent.locator("[x-html='cooldownSVG()'] [data-region='0']").first();
  await spot.dispatchEvent("click");
  // Painted parts are cel-shaded; the lit layer carries the chosen color.
  const painted = await parent.locator("[x-html='cooldownSVG()'] [data-region='0'][data-lit]").getAttribute("fill");
  await expect(painted === "#ef476f", `first spot should take the first color, got ${painted}`);
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
