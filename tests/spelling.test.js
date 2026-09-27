const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const test = require("node:test");
const vm = require("node:vm");

const root = path.join(__dirname, "..");

// blankedSentence() masks only the first whole-word match, so a sentence that
// uses the answer twice would show the answer on screen.
function assertSingleOccurrence({ word, sentence }) {
  const matches = sentence.match(new RegExp(`\\b${word}\\b`, "gi")) || [];
  assert.equal(matches.length, 1, `"${sentence}" must contain "${word}" exactly once`);
}

function loadGame({ withAudio = false, rejectPlayback = false } = {}) {
  const audioInstances = [];
  const spoken = [];
  class FakeAudio {
    constructor(src) {
      this.src = src;
      this.currentTime = 0;
      this.playCount = 0;
      this.pauseCount = 0;
      audioInstances.push(this);
    }

    play() {
      this.playCount += 1;
      return rejectPlayback ? Promise.reject(new Error("NotAllowedError")) : Promise.resolve();
    }

    pause() {
      this.pauseCount += 1;
    }
  }

  class FakeUtterance { constructor(text) { this.text = text; } }
  const fakeSpeech = { cancel() {}, speak(utterance) { spoken.push(utterance.text); } };
  const context = vm.createContext({
    console,
    ...(rejectPlayback ? { SpeechSynthesisUtterance: FakeUtterance, setTimeout } : {}),
    Math,
    Set,
    RegExp,
    window: {
      matchMedia: () => ({ matches: true }),
      scrollTo: () => {},
      ...(withAudio ? { Audio: FakeAudio } : {}),
      ...(rejectPlayback ? { speechSynthesis: fakeSpeech, SpeechSynthesisUtterance: FakeUtterance } : {}),
    },
    document: {
      activeElement: null,
      querySelector: () => null,
    },
    requestAnimationFrame: (callback) => callback(),
  });
  vm.runInContext(fs.readFileSync(path.join(root, "static", "words.js"), "utf8"), context);
  vm.runInContext(fs.readFileSync(path.join(root, "static", "audio", "spelling", "manifest.js"), "utf8"), context);
  vm.runInContext(fs.readFileSync(path.join(root, "static", "app.js"), "utf8"), context);
  return { game: context.dojo(), context, audioInstances, spoken };
}

test("every unique curriculum word has a nonempty recorded clip", () => {
  const { context } = loadGame();
  const banks = vm.runInContext("SPELLING_WORDS", context);
  const sightBanks = vm.runInContext("SIGHT_WORDS", context);
  const audio = vm.runInContext("SPELLING_AUDIO", context);
  const expected = [...new Set(
    [...Object.values(banks).flat(), ...Object.values(sightBanks).flat()].map(({ word }) => word)
  )].sort();

  assert.equal(audio.voice, "Samantha");
  assert.equal(audio.format, "opus");
  assert.deepEqual([...audio.words], expected);
  for (const word of expected) {
    const clip = path.join(root, audio.basePath.replace("/static/", "static/"), `${word}.${audio.format}`);
    assert.ok(fs.statSync(clip).size > 500, `missing or empty audio for ${word}`);
  }
});

test("hear the word plays only the current pre-rendered clip", () => {
  const { game, audioInstances } = loadGame({ withAudio: true });
  game.spellingCount = 1;
  game.startSpelling();

  assert.equal(audioInstances.length, 1);
  assert.match(audioInstances[0].src, new RegExp(`/samantha-v1/${game.currentWord.word}\\.opus$`));
  assert.equal(audioInstances[0].preload, "auto");
  game.speakWord();
  assert.equal(audioInstances[0].playCount, 1);
  assert.equal(game.statusMessage, "Playing the word aloud.");

  game.reset();
  assert.equal(audioInstances[0].pauseCount, 1);
});

test("a late playback rejection never speaks the next word aloud", async () => {
  const { game, spoken } = loadGame({ withAudio: true, rejectPlayback: true });
  game.spellingCount = 2;
  game.startSpelling();
  const first = game.currentWord.word;

  game.speakWord();
  await new Promise((resolve) => setTimeout(resolve, 0));
  assert.deepEqual(spoken, [`${first}.`], "rejection during the round falls back to speech");

  game.speakWord();
  game.reset();
  await new Promise((resolve) => setTimeout(resolve, 0));
  assert.deepEqual(spoken, [`${first}.`], "rejection after reset is ignored");
});

test("every grade has valid words and at least five examples per skill", () => {
  const { context } = loadGame();
  const banks = vm.runInContext("SPELLING_WORDS", context);
  const curricula = vm.runInContext("SPELLING_SKILLS", context);
  const sightBanks = vm.runInContext("SIGHT_WORDS", context);
  const allWords = [];
  const allSightWords = [];
  for (let grade = 1; grade <= 5; grade += 1) {
    const skillIds = new Set(curricula[grade].map(({ id }) => id));
    assert.ok(banks[grade].length >= 25);
    assert.equal(new Set(banks[grade].map(({ word }) => word)).size, banks[grade].length);
    for (const entry of banks[grade]) {
      assert.match(entry.word, /^[a-z]+$/);
      assert.ok(entry.clue.length > 5);
      assertSingleOccurrence(entry);
      assert.ok(skillIds.has(entry.skill), `unknown skill ${entry.skill} for ${entry.word}`);
      allWords.push(entry.word);
    }
    for (const skill of curricula[grade]) {
      assert.ok(skill.label.length > 3);
      assert.ok(skill.tip.length > 10);
      assert.ok(banks[grade].filter((entry) => entry.skill === skill.id).length >= 5);
    }
    assert.equal(sightBanks[grade].length, 10);
    for (const entry of sightBanks[grade]) {
      assert.match(entry.word, /^[a-z]+$/);
      assert.equal(entry.skill, "sight-words");
      assert.ok(Number.isInteger(entry.rank));
      assertSingleOccurrence(entry);
      allSightWords.push(entry.word);
    }
  }
  assert.equal(allWords.length, 130);
  assert.equal(Object.values(curricula).flat().length, 26);
  assert.equal(allSightWords.length, 50);
  assert.equal(new Set(allWords).size, allWords.length, "words should not repeat across grades");
  assert.equal(new Set(allSightWords).size, allSightWords.length, "sight words should not repeat across grades");
});

test("starting Word Rescue selects the requested number of unique words", () => {
  const { game } = loadGame();
  game.spellingGrade = 3;
  game.spellingCount = 5;
  game.startSpelling();

  assert.equal(game.view, "spelling-game");
  assert.equal(game.spellingWords.length, 5);
  assert.equal(new Set(game.spellingWords.map(({ word }) => word)).size, 5);
  assert.equal(game.maskedLetters().length, game.currentWord.word.length);
  assert.ok(game.maskedLetters().every((letter) => letter === "_"));
  assert.ok(new Set(game.spellingWords.map(({ skill }) => skill)).size > 1);
});

test("focused practice begins with five words from the selected skill", () => {
  const { game } = loadGame();
  game.spellingGrade = 4;
  game.spellingCount = 10;
  game.spellingFocus = "g4-roots";
  game.startSpelling();

  assert.ok(game.spellingWords.slice(0, 5).every(({ skill }) => skill === "g4-roots"));
  assert.ok(game.spellingWords.slice(5).every(({ skill }) => skill !== "g4-roots"));
  assert.equal(game.currentSkillLabel(), "Roots and prefixes");
  assert.match(game.currentSkillTip(), /roots/i);
});

test("sight-word focus produces a full sight-word session", () => {
  const { game, context } = loadGame();
  const sightBanks = vm.runInContext("SIGHT_WORDS", context);
  game.spellingGrade = 2;
  game.spellingCount = 10;
  game.spellingFocus = "sight-words";
  game.startSpelling();

  assert.equal(game.spellingWords.length, 10);
  assert.ok(game.spellingWords.every(({ skill }) => skill === "sight-words"));
  assert.equal(game.spellingWords.filter(({ rank }) => rank >= 12 && rank <= 21).length, 6);
  assert.equal(game.spellingWords.filter(({ rank }) => rank <= 11).length, 4);
  assert.equal(game.currentSkillLabel(), "Sight words");

  game.currentWord = sightBanks[1].find(({ word }) => word === "a");
  assert.equal(game.blankedSentence(), "_____ bird landed on the fence.");
});

test("the browser round rules match the shared replay fixtures", () => {
  const fixtures = JSON.parse(fs.readFileSync(path.join(root, "tests", "fixtures", "replay.json"), "utf8"));
  assert.ok(fixtures.length >= 10);
  for (const fixture of fixtures) {
    const { game } = loadGame();
    game.spellingCount = 1;
    game.startSpelling();
    // Point the round at the fixture word; Alpine state is plain data.
    game.currentWord = { word: fixture.word, skill: "test", clue: "", sentence: fixture.word };
    game.spellingWords = [game.currentWord];
    for (const guess of fixture.guesses) {
      if (guess.kind === "letter") game.guessLetter(guess.value);
      else {
        game.wholeWordGuess = guess.value;
        game.guessWholeWord();
      }
    }
    assert.deepEqual(
      { won: game.roundWon, done: game.roundDone, mistakes: game.mistakes },
      fixture.want,
      fixture.name
    );
    // The log the server will replay records every event the component
    // acted on, tagged, in order. Events the component ignored (repeats,
    // blanks, post-round) are absent, which the server also ignores.
    const acted = fixture.guesses
      .filter((g, i, all) => {
        if (g.kind === "word") return g.value.trim() !== "";
        return !all.slice(0, i).some((p) => p.kind === "letter" && p.value === g.value);
      })
      // Word guesses are logged as compared: trimmed and lower-cased.
      .map((g) => (g.kind === "word" ? { kind: "word", value: g.value.trim().toLowerCase() } : g));
    const logged = game.guessLog[0];
    assert.ok(logged.length <= acted.length, `${fixture.name}: log has extra events`);
    // JSON round-trip: the log lives in the vm realm, so prototypes differ.
    assert.equal(JSON.stringify(logged), JSON.stringify(acted.slice(0, logged.length)), `${fixture.name}: log order`);
  }
});

test("a signed-in child starts at their own grade", () => {
  const { context } = loadGame();
  context.document.querySelector = (sel) => sel === "[data-child-id]" ? { dataset: { childId: "7", childNickname: "Nova", childGrade: "3" } } : null;
  const game = context.dojo();
  assert.equal(game.child.grade, 3);
  assert.equal(game.grade, 3);
  assert.equal(game.spellingGrade, 3);
  assert.ok(game.spellingSkillChoices().some((s) => s.id === "review"), "signed-in children get the review focus");
});

test("the paint mixin draws the mosaic and lays out pages deterministically", () => {
  const { context } = loadGame();
  vm.runInContext(fs.readFileSync(path.join(root, "static", "kata-svg.js"), "utf8"), context);
  vm.runInContext(fs.readFileSync(path.join(root, "static", "kata.js"), "utf8"), context);
  vm.runInContext(fs.readFileSync(path.join(root, "static", "paint.js"), "utf8"), context);
  const game = context.dojo();
  const mosaic = { size: 2, palette: ["#eee", "#f00"], cells: "0110", revealed: 2, total: 4, order: [3, 1, 0, 2] };
  const cells = game.mosaicCells(mosaic);
  assert.equal(cells.length, 4);
  assert.equal(JSON.stringify(cells.map((c) => c.shown)), JSON.stringify([false, true, false, true]));
  assert.equal(cells[1].color, "#f00");
  assert.equal(cells[3].color, "#eee");
  assert.equal(game.mosaicLabel(mosaic), "This week's mosaic, 2 of 4 tiles revealed");
  const a = game.regionShape(3, 14, 123);
  const b = game.regionShape(3, 14, 123);
  assert.equal(a.path, b.path, "layout is a pure function of the seed");
  assert.notEqual(a.path, game.regionShape(3, 14, 124).path);
  game.page = { regions: [{ idx: 0, filled: true, answer: "7", color: 1 }, { idx: 1, filled: true, answer: "7", color: 1 }, { idx: 2, filled: false, color: -1 }] };
  assert.equal(JSON.stringify(game.pageLegend().map((l) => l.answer)), JSON.stringify(["7"]));
  assert.equal(game.regionColor(game.page.regions[1]), game.pageLegend()[0].color);
  assert.equal(game.regionColor(game.page.regions[2]), "");
  // Colors come from the server's per-answer index, so solving more never recolors.
  const before = game.regionColor(game.page.regions[0]);
  game.page.regions[2] = { idx: 2, filled: true, answer: "12", color: 0 };
  assert.equal(game.regionColor(game.page.regions[0]), before);
  assert.notEqual(game.regionColor(game.page.regions[2]), before);
  game.child = null;
  game.startCooldown();
  assert.notEqual(game.view, "cooldown", "anonymous play never enters cooldown");
  game.pageSelected = 1;
  game.page = { seed: 5, total: 3, regions: [{ idx: 0, prompt: "3 + 4", filled: true, answer: "7" }, { idx: 1, prompt: "<b>", filled: false }, { idx: 2, prompt: "9 - 2", filled: false }] };
  const svg = game.pageSVG();
  assert.match(svg, /role="group" aria-label="Color by number page"/);
  assert.equal((svg.match(/data-region=/g) || []).length, 3);
  assert.match(svg, /aria-pressed="true"/);
  assert.ok(svg.includes("&lt;b&gt;") && !svg.includes("<b>"), "prompts are escaped");
  // Cooldown painting only honours strict hex colors.
  game.cooldown = { seed: 5, palette: "tide", regions: 20, colors: Array(20).fill("") };
  game.cooldown.colors[0] = "#ef476f";
  assert.match(game.cooldownSVG(), /#ef476f/);
});

test("the battle mixin ignores responses for a battle that is no longer on screen", async () => {
  const { context } = loadGame();
  vm.runInContext(fs.readFileSync(path.join(root, "static", "kata-svg.js"), "utf8"), context);
  vm.runInContext(fs.readFileSync(path.join(root, "static", "kata.js"), "utf8"), context);
  vm.runInContext(fs.readFileSync(path.join(root, "static", "battle.js"), "utf8"), context);
  const game = context.dojo();
  assert.equal(game.canBattle(), false, "anonymous children never battle");
  assert.equal(JSON.stringify(game.hearts(3)), JSON.stringify([true, true, true, false, false]));
  game.battleId = "battle-a";
  game.battle = { turn: 2, done: false, child: {}, opponent: {} };
  game.battleAnswer = "cat";
  let resolve;
  game.post = () => new Promise((r) => { resolve = r; });
  const pending = game.submitBattleAnswer();
  // The child leaves before the answer comes back.
  game.battleId = "";
  game.battle = null;
  resolve({ turn: 3, done: true, won: true, child: {}, opponent: {}, log: [] });
  await pending;
  assert.equal(game.battle, null, "a stale response must not resurrect the battle");
  assert.equal(game.battleBusy, true, "busy flag belongs to the abandoned battle and is left alone");
});

test("the kata mixin is merged and renders deterministic creatures", () => {
  const { context } = loadGame();
  vm.runInContext(fs.readFileSync(path.join(root, "static", "kata-svg.js"), "utf8"), context);
  vm.runInContext(fs.readFileSync(path.join(root, "static", "kata.js"), "utf8"), context);
  const game = context.dojo();
  assert.equal(typeof game.openKata, "function");
  const entry = { id: "g1-blends", name: "Brindle", seed: 3, palette: "meadow", regions: 20, fills: 7, state: "seen", kind: "spelling", grade: 1, focus: "g1-blends" };
  const svg = game.kataSVG(entry, 96);
  assert.match(svg, /role="img"/);
  assert.match(svg, /aria-label="Brindle, 7 of 20 regions colored"/);
  assert.equal(svg, game.kataSVG(entry, 96), "same seed, same picture");
  assert.equal(game.kataName({ ...entry, state: "unknown" }), "???");
  assert.equal(game.kataStateLabel(entry), "7 of 20 colored");
  game.trainHere({ kind: "math", focus: "mul", grade: 3 });
  assert.equal(game.subject, "math");
  assert.deepEqual([...game.ops], ["mul"]);
  assert.equal(game.grade, 3);
});

test("anonymous play never calls the round endpoints and keeps no log server could use", () => {
  const { game, context } = loadGame();
  let fetched = false;
  context.fetch = () => { fetched = true; return Promise.reject(new Error("no")); };
  game.spellingCount = 5;
  game.startSpelling();
  assert.equal(game.child, null);
  assert.equal(game.roundId, "");
  assert.equal(game.guessLog.length, 5);
  assert.equal(fetched, false);
});

test("guessing every distinct letter rescues a word", () => {
  const { game } = loadGame();
  game.spellingCount = 1;
  game.startSpelling();

  const letters = new Set(game.currentWord.word.toUpperCase());
  for (const letter of letters) game.guessLetter(letter);

  assert.equal(game.roundDone, true);
  assert.equal(game.roundWon, true);
  assert.equal(game.spellingScore, 1);
  assert.deepEqual(game.maskedLetters().join(""), game.currentWord.word.toUpperCase());
});

test("six incorrect letters ends the round and reveals the answer", () => {
  const { game } = loadGame();
  game.spellingCount = 1;
  game.startSpelling();

  const answer = game.currentWord.word.toUpperCase();
  const wrongLetters = game.alphabet.filter((letter) => !answer.includes(letter)).slice(0, 6);
  for (const letter of wrongLetters) game.guessLetter(letter);

  assert.equal(game.triesLeft(), 0);
  assert.equal(game.roundDone, true);
  assert.equal(game.roundWon, false);
  assert.deepEqual(game.maskedLetters().join(""), answer);
});

test("whole-word guesses cost one try or rescue the word", () => {
  const { game } = loadGame();
  game.spellingCount = 1;
  game.startSpelling();

  game.wholeWordGuess = "nottherightword";
  game.guessWholeWord();
  assert.equal(game.mistakes, 1);
  assert.equal(game.roundDone, false);

  game.wholeWordGuess = game.currentWord.word.toUpperCase();
  game.guessWholeWord();
  assert.equal(game.roundDone, true);
  assert.equal(game.roundWon, true);
  game.nextSpellingWord();
  assert.equal(game.view, "spelling-results");
});
