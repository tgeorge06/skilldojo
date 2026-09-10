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
