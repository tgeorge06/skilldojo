// SkillDojo front-end state (Alpine component).
function dojo() {
  return {
    view: "setup",
    subject: "math",
    busy: false,
    error: "",

    // Math dojo state.
    ops: ["addsub"],
    grade: 1,
    count: 10,
    sheetId: "",
    questions: [],
    answers: [],
    report: { results: [], score: 0, total: 0, percent: 0 },
    opChoices: [
      { id: "addsub", label: "Add & Subtract", emoji: "➕", hint: "big numbers!" },
      { id: "mul", label: "Multiplication", emoji: "✖️", hint: "times tables" },
      { id: "div", label: "Division", emoji: "➗", hint: "no remainders" },
      { id: "frac", label: "Fractions", emoji: "🍕", hint: "answer like 3/4" },
    ],
    gradeHints: {
      1: "numbers up to 20",
      2: "numbers up to 100",
      3: "3-digit numbers, full times tables",
      4: "big numbers, tricky tables",
      5: "the toughest problems",
    },

    // Spelling dojo state.
    spellingGrade: 1,
    spellingCount: 5,
    spellingFocus: "mixed",
    spellingWords: [],
    spellingRound: 0,
    spellingScore: 0,
    spellingResults: [],
    currentWord: null,
    guessedLetters: [],
    mistakes: 0,
    wholeWordGuess: "",
    roundDone: false,
    roundWon: false,
    statusMessage: "",
    audioAvailable: "Audio" in window,
    speechAvailable: "speechSynthesis" in window && "SpeechSynthesisUtterance" in window,
    wordAudio: null,
    alphabet: "ABCDEFGHIJKLMNOPQRSTUVWXYZ".split(""),
    spellingGradeHints: {
      1: "short everyday words",
      2: "common words and patterns",
      3: "longer words and blends",
      4: "tricky endings and vowels",
      5: "challenging school words",
    },

    headerSubtitle() {
      return this.subject === "spelling"
        ? "Spelling Dojo — listen, guess, and rescue words"
        : "Math Dojo — pick your grade, train your skills";
    },
    chooseSubject(subject) {
      this.subject = subject;
      this.error = "";
    },
    startTraining() {
      if (this.subject === "spelling") this.startSpelling();
      else this.startSheet();
    },

    // Math dojo methods.
    toggleOp(id) {
      this.ops = this.ops.includes(id) ? this.ops.filter((o) => o !== id) : [...this.ops, id];
    },
    gradeHint() {
      return this.gradeHints[this.grade] || "";
    },
    answeredCount() {
      return this.answers.filter((a) => a && a.trim() !== "").length;
    },
    gradeMessage() {
      const p = this.report.percent;
      if (p === 100) return "PERFECT! A true SkillDojo master!";
      if (p >= 90) return "Amazing work — almost perfect!";
      if (p >= 80) return "Great job! Keep training!";
      if (p >= 60) return "Good effort — practice makes perfect!";
      return "Every ninja starts somewhere. Try again!";
    },
    async post(url, body) {
      const res = await fetch(url, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) throw new Error(data.error || "Something went wrong — try again.");
      return data;
    },
    async startSheet() {
      this.busy = true;
      this.error = "";
      try {
        const data = await this.post("/api/sheet", {
          ops: this.ops,
          grade: this.grade,
          count: this.count,
        });
        this.sheetId = data.id;
        this.questions = data.questions;
        this.answers = data.questions.map(() => "");
        this.view = "math-sheet";
        this.moveToTop();
      } catch (e) {
        this.error = e.message;
      } finally {
        this.busy = false;
      }
    },
    async submitSheet() {
      if (this.answeredCount() < this.questions.length &&
          !window.confirm("Some questions are blank — grade anyway?")) {
        return;
      }
      this.busy = true;
      this.error = "";
      try {
        this.report = await this.post("/api/grade", { id: this.sheetId, answers: this.answers });
        this.view = "math-results";
        this.moveToTop();
        if (this.report.percent === 100) confettiBurst();
      } catch (e) {
        this.error = e.message;
      } finally {
        this.busy = false;
      }
    },

    // Spelling dojo methods.
    spellingGradeHint() {
      return this.spellingGradeHints[this.spellingGrade] || "";
    },
    spellingSkillChoices() {
      return [...(SPELLING_SKILLS[this.spellingGrade] || []), SIGHT_WORD_SKILL];
    },
    spellingFocusDescription() {
      if (this.spellingFocus === "mixed") {
        return "A balanced session that rotates through several skills.";
      }
      return this.spellingSkillChoices().find((skill) => skill.id === this.spellingFocus)?.tip || "";
    },
    selectSpellingWords() {
      const bank = [...(SPELLING_WORDS[this.spellingGrade] || [])];
      if (this.spellingFocus === SIGHT_WORD_SKILL.id) {
        const currentBand = shuffle([...(SIGHT_WORDS[this.spellingGrade] || [])]);
        const earlierBands = shuffle(Object.entries(SIGHT_WORDS)
          .filter(([grade]) => Number(grade) < this.spellingGrade)
          .flatMap(([, words]) => words));
        if (!earlierBands.length) return currentBand.slice(0, this.spellingCount);

        const currentCount = Math.min(currentBand.length, Math.ceil(this.spellingCount * 0.6));
        return shuffle([
          ...currentBand.slice(0, currentCount),
          ...earlierBands.slice(0, this.spellingCount - currentCount),
        ]);
      }
      if (this.spellingFocus !== "mixed") {
        const focused = shuffle(bank.filter((entry) => entry.skill === this.spellingFocus));
        const review = shuffle(bank.filter((entry) => entry.skill !== this.spellingFocus));
        return [...focused, ...review].slice(0, this.spellingCount);
      }

      // Round-robin across shuffled skill buckets so a mixed session really
      // contains varied patterns instead of relying on a lucky random draw.
      const buckets = shuffle((SPELLING_SKILLS[this.spellingGrade] || []).map((skill) =>
        shuffle(bank.filter((entry) => entry.skill === skill.id))
      ));
      const selected = [];
      for (let round = 0; selected.length < this.spellingCount; round += 1) {
        let added = false;
        for (const bucket of buckets) {
          if (bucket[round] && selected.length < this.spellingCount) {
            selected.push(bucket[round]);
            added = true;
          }
        }
        if (!added) break;
      }
      return selected;
    },
    startSpelling() {
      this.subject = "spelling";
      this.spellingWords = this.selectSpellingWords();
      this.spellingRound = 0;
      this.spellingScore = 0;
      this.spellingResults = [];
      this.view = "spelling-game";
      this.loadSpellingWord();
      this.moveToTop("#spelling-word-heading");
    },
    loadSpellingWord() {
      this.stopWordAudio();
      this.currentWord = this.spellingWords[this.spellingRound];
      this.guessedLetters = [];
      this.mistakes = 0;
      this.wholeWordGuess = "";
      this.roundDone = false;
      this.roundWon = false;
      this.statusMessage = "New word ready. Choose a letter or hear the word.";
      this.prepareWordAudio();
    },
    currentSkill() {
      if (!this.currentWord) return null;
      return this.spellingSkillChoices().find((skill) => skill.id === this.currentWord.skill) || null;
    },
    currentSkillLabel() {
      return this.currentSkill()?.label || "Spelling pattern";
    },
    currentSkillTip() {
      return this.currentSkill()?.tip || "Say the word slowly and notice each sound and letter pattern.";
    },
    maskedLetters() {
      if (!this.currentWord) return [];
      return this.currentWord.word.toUpperCase().split("").map((letter) =>
        this.roundDone || this.guessedLetters.includes(letter) ? letter : "_"
      );
    },
    maskedWordForScreenReader() {
      const letters = this.maskedLetters();
      const revealed = letters.map((letter) => letter === "_" ? "blank" : letter).join(", ");
      return `${letters.length} letters: ${revealed}`;
    },
    blankedSentence() {
      if (!this.currentWord) return "";
      const escapedWord = this.currentWord.word.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
      return this.currentWord.sentence.replace(new RegExp(`\\b${escapedWord}\\b`, "i"), "_____");
    },
    triesLeft() {
      return Math.max(0, 6 - this.mistakes);
    },
    spellingProgress() {
      if (!this.spellingWords.length) return 0;
      return ((this.spellingRound + (this.roundDone ? 1 : 0)) / this.spellingWords.length) * 100;
    },
    isWordRevealed() {
      return this.currentWord.word.toUpperCase().split("").every((letter) => this.guessedLetters.includes(letter));
    },
    guessLetter(letter) {
      if (this.view !== "spelling-game" || this.roundDone || this.guessedLetters.includes(letter)) return;
      this.guessedLetters = [...this.guessedLetters, letter];
      if (this.currentWord.word.toUpperCase().includes(letter)) {
        if (this.isWordRevealed()) this.finishSpellingRound(true);
        else this.statusMessage = `Nice! ${letter} is in the word.`;
      } else {
        this.mistakes += 1;
        if (this.mistakes >= 6) this.finishSpellingRound(false);
        else this.statusMessage = `No ${letter} this time. ${this.triesLeft()} tries left.`;
      }
    },
    guessWholeWord() {
      if (this.roundDone || !this.wholeWordGuess.trim()) return;
      const guess = this.wholeWordGuess.trim().toLowerCase();
      if (guess === this.currentWord.word.toLowerCase()) {
        this.guessedLetters = [...new Set(this.currentWord.word.toUpperCase().split(""))];
        this.finishSpellingRound(true);
        return;
      }
      this.mistakes += 1;
      this.wholeWordGuess = "";
      if (this.mistakes >= 6) this.finishSpellingRound(false);
      else this.statusMessage = `Not quite. Check the clue and try again — ${this.triesLeft()} tries left.`;
    },
    finishSpellingRound(won) {
      this.roundDone = true;
      this.roundWon = won;
      if (won) {
        this.spellingScore += 1;
        this.statusMessage = `Rescued! ${this.currentWord.word.toUpperCase()} is spelled ${this.spelledOutWord()}.`;
      } else {
        this.statusMessage = `Good try! The word was ${this.currentWord.word.toUpperCase()}, spelled ${this.spelledOutWord()}.`;
      }
      this.spellingResults.push({ word: this.currentWord.word, won });
      requestAnimationFrame(() => document.querySelector("#next-spelling-button")?.focus());
    },
    spelledOutWord() {
      return this.currentWord.word.toUpperCase().split("").join("–");
    },
    nextSpellingWord() {
      if (!this.roundDone) return;
      if (this.spellingRound + 1 >= this.spellingWords.length) {
        this.view = "spelling-results";
        this.moveToTop("#spelling-results-heading");
        if (this.spellingScore === this.spellingWords.length) confettiBurst();
        return;
      }
      this.spellingRound += 1;
      this.loadSpellingWord();
      this.moveToTop("#spelling-word-heading");
    },
    spellingResultMessage() {
      if (this.spellingScore === this.spellingWords.length) return "Perfect rescue! Every word is glowing.";
      if (this.spellingScore >= Math.ceil(this.spellingWords.length * 0.8)) return "Fantastic spelling! Those tricky patterns are getting stronger.";
      if (this.spellingScore >= Math.ceil(this.spellingWords.length * 0.5)) return "Strong training! Review the words with a book icon and play again.";
      return "Every word you review makes you a stronger speller. Keep going!";
    },
    letterState(letter) {
      if (!this.guessedLetters.includes(letter)) return "letter-key-ready";
      return this.currentWord.word.toUpperCase().includes(letter) ? "letter-key-correct" : "letter-key-wrong";
    },
    letterButtonLabel(letter) {
      if (!this.guessedLetters.includes(letter)) return `Guess letter ${letter}`;
      return this.currentWord.word.toUpperCase().includes(letter)
        ? `${letter}, guessed, in the word`
        : `${letter}, guessed, not in the word`;
    },
    handleKey(event) {
      if (this.view !== "spelling-game" || this.roundDone || event.metaKey || event.ctrlKey || event.altKey) return;
      const tag = document.activeElement ? document.activeElement.tagName : "";
      if (tag === "INPUT" || tag === "TEXTAREA") return;
      const letter = event.key.toUpperCase();
      if (/^[A-Z]$/.test(letter)) this.guessLetter(letter);
    },
    hearingAvailable() {
      return (this.audioAvailable && this.hasRecordedWord()) || this.speechAvailable;
    },
    hasRecordedWord() {
      return this.currentWord &&
        typeof SPELLING_AUDIO !== "undefined" &&
        SPELLING_AUDIO.words.includes(this.currentWord.word);
    },
    wordAudioUrl() {
      if (!this.hasRecordedWord()) return "";
      return `${SPELLING_AUDIO.basePath}/${encodeURIComponent(this.currentWord.word)}.${SPELLING_AUDIO.format}`;
    },
    prepareWordAudio() {
      if (!this.audioAvailable || !this.hasRecordedWord()) return;
      this.wordAudio = new window.Audio(this.wordAudioUrl());
      // Only the active round's tiny clip is eligible for preloading.
      this.wordAudio.preload = "auto";
    },
    stopWordAudio() {
      if (!this.wordAudio) return;
      this.wordAudio.pause();
      this.wordAudio.onerror = null;
      this.wordAudio = null;
    },
    speakWithBrowserVoice() {
      if (!this.speechAvailable) {
        this.statusMessage = "Audio is unavailable. Use the clue and sentence for this word.";
        return;
      }
      window.speechSynthesis.cancel();
      const utterance = new SpeechSynthesisUtterance(`${this.currentWord.word}.`);
      utterance.lang = "en-US";
      utterance.rate = 0.88;
      window.speechSynthesis.speak(utterance);
      this.statusMessage = "Playing the word aloud.";
    },
    speakWord() {
      if (!this.hearingAvailable() || !this.currentWord || this.roundDone) return;
      window.speechSynthesis?.cancel();
      if (!this.wordAudio) this.prepareWordAudio();
      if (!this.wordAudio) {
        this.speakWithBrowserVoice();
        return;
      }

      let fellBack = false;
      const fallBackToSpeech = () => {
        if (fellBack) return;
        fellBack = true;
        this.speakWithBrowserVoice();
      };
      this.wordAudio.onerror = fallBackToSpeech;
      this.wordAudio.currentTime = 0;
      const playback = this.wordAudio.play();
      if (playback?.catch) playback.catch(fallBackToSpeech);
      this.statusMessage = "Playing the word aloud.";
    },

    moveToTop(focusSelector) {
      window.scrollTo(0, 0);
      if (focusSelector) requestAnimationFrame(() => document.querySelector(focusSelector)?.focus());
    },
    reset() {
      this.stopWordAudio();
      window.speechSynthesis?.cancel();
      this.view = "setup";
      this.error = "";
      this.questions = [];
      this.answers = [];
      this.moveToTop();
    },
  };
}

function shuffle(items) {
  for (let i = items.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [items[i], items[j]] = [items[j], items[i]];
  }
  return items;
}

// Reduced-motion users get the same written celebration without animation.
function confettiBurst() {
  if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) return;
  const colors = ["#39bd95", "#60f2da", "#fbbf24", "#38bdf8", "#a78bfa", "#fb7185"];
  for (let i = 0; i < 80; i++) {
    const el = document.createElement("div");
    const size = 6 + Math.random() * 8;
    el.style.cssText =
      "position:fixed;top:-20px;z-index:50;pointer-events:none;border-radius:2px;" +
      "left:" + Math.random() * 100 + "vw;" +
      "width:" + size + "px;height:" + size + "px;" +
      "background:" + colors[i % colors.length] + ";";
    document.body.appendChild(el);
    el.animate(
      [
        { transform: "translateY(0) rotate(0deg)", opacity: 1 },
        { transform: "translateY(" + (window.innerHeight + 40) + "px) rotate(" + (360 + Math.random() * 720) + "deg)", opacity: 0.7 },
      ],
      { duration: 2200 + Math.random() * 1800, easing: "cubic-bezier(.2,.6,.4,1)" }
    ).onfinish = () => el.remove();
  }
}
