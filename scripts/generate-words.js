#!/usr/bin/env node
// Generates static/words.js from internal/curriculum/spelling.json so the
// browser and the Go server share one curriculum. Run via `make words`.
// The output is deterministic; CI fails if the committed file is stale.
"use strict";

const fs = require("node:fs");
const path = require("node:path");

const root = path.join(__dirname, "..");
const source = path.join(root, "internal", "curriculum", "spelling.json");
const target = path.join(root, "static", "words.js");
const data = JSON.parse(fs.readFileSync(source, "utf8"));

const js = (value) => JSON.stringify(value);

// Grade keys must be the canonical strings "1".."5" and identical across
// banks, and every word must be lowercase a-z: the Go side enforces the same
// invariants, so a bad JSON fails here before it can reach the browser.
const CANONICAL = ["1", "2", "3", "4", "5"];
function validateGrades(name, bank) {
  const keys = Object.keys(bank).sort();
  if (keys.some((k) => !CANONICAL.includes(k))) throw new Error(`${name}: grade keys must be "1".."5", got ${keys}`);
  return keys.map(Number).sort((a, b) => a - b);
}
const skillGrades = validateGrades("skills", data.skills);
for (const [name, bank] of [["words", data.words], ["sightWords", data.sightWords]]) {
  const grades = validateGrades(name, bank);
  if (grades.join() !== skillGrades.join()) throw new Error(`${name} grades ${grades} differ from skills grades ${skillGrades}`);
  for (const entry of Object.values(bank).flat()) {
    if (!/^[a-z]+$/.test(entry.word)) throw new Error(`${name}: word ${js(entry.word)} must be lowercase a-z`);
    if (name === "sightWords" && !Number.isInteger(entry.rank)) throw new Error(`sightWords: ${entry.word} needs an integer rank`);
  }
}
const grades = (bank) => Object.keys(bank).map(Number).sort((a, b) => a - b);

const lines = [];
lines.push(
  "// GENERATED FILE — do not edit. Source: internal/curriculum/spelling.json.",
  "// Regenerate with `make words`.",
  "//",
  "// SkillDojo's spelling curriculum is organized by teachable patterns instead",
  "// of word length alone. Each focus contains at least five words so it can power",
  "// a complete focused session; ten-word sessions add mixed review words.",
  "const SPELLING_SKILLS = {"
);
for (const grade of grades(data.skills)) {
  lines.push(`  ${grade}: [`);
  for (const skill of data.skills[grade]) {
    lines.push(`    { id: ${js(skill.id)}, label: ${js(skill.label)}, tip: ${js(skill.tip)} },`);
  }
  lines.push("  ],");
}
lines.push("};", "", "const spellingWord = (word, skill, clue, sentence) => ({ word, skill, clue, sentence });", "", "const SPELLING_WORDS = {");
for (const grade of grades(data.words)) {
  lines.push(`  ${grade}: [`);
  let previousSkill = null;
  for (const entry of data.words[grade]) {
    if (previousSkill && entry.skill !== previousSkill) lines.push("");
    previousSkill = entry.skill;
    lines.push(`    spellingWord(${js(entry.word)}, ${js(entry.skill)}, ${js(entry.clue)}, ${js(entry.sentence)}),`);
  }
  lines.push("  ],");
}
lines.push(
  "};",
  "",
  "// The first 50 non-contraction selections from the 2024 Children's Picture",
  "// Book Sight Words ranking, split into progressive frequency bands. \"I\" is",
  "// omitted because a one-letter pronoun does not produce meaningful spelling",
  "// practice. Rank preserves source provenance for later curriculum review.",
  "// https://doi.org/10.1002/trtr.2309",
  "const SIGHT_WORD_SKILL = {",
  `  id: ${js(data.sightWordSkill.id)},`,
  `  label: ${js(data.sightWordSkill.label)},`,
  `  tip: ${js(data.sightWordSkill.tip)},`,
  "};",
  "",
  "const sightWord = (word, rank, clue, sentence) => ({",
  "  word,",
  "  rank,",
  "  skill: SIGHT_WORD_SKILL.id,",
  "  clue,",
  "  sentence,",
  "});",
  "",
  "const SIGHT_WORDS = {"
);
for (const grade of grades(data.sightWords)) {
  lines.push(`  ${grade}: [`);
  for (const entry of data.sightWords[grade]) {
    lines.push(`    sightWord(${js(entry.word)}, ${js(entry.rank)}, ${js(entry.clue)}, ${js(entry.sentence)}),`);
  }
  lines.push("  ],");
}
lines.push("};", "");

const output = lines.join("\n");
if (process.argv.includes("--check")) {
  const current = fs.existsSync(target) ? fs.readFileSync(target, "utf8") : "";
  if (current !== output) {
    console.error("static/words.js is out of date; run `make words`.");
    process.exit(1);
  }
  console.log("static/words.js is in sync.");
} else {
  fs.writeFileSync(target, output);
  console.log(`Wrote ${path.relative(root, target)}`);
}
