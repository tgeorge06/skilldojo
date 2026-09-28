// SkillDojo front-end state (Alpine component).
function dojo() {
  const core = {
    view: "home",
    subject: "math",
    showPatterns: false,
    quests: null, // today's three, for a signed-in child
    questsError: "",
    holdPct: 0,
    holdTimer: null,
    hashStart: false,
    hashTables: false,
    // Signed-in child, read from data attributes the server renders on the
    // root element (never interpolated into x-data). Null when anonymous.
    child: readChild(),
    ...defaultGrades(),
    // Times-table practice: which table, and whether facts come in order.
    table: 7,
    tableOrdered: true,
    tables: [2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12],
    roundId: "",
    guessLog: [],
    reward: null,
    busy: false,
    error: "",

    // Math dojo state. Round length is a parent setting on the profile;
    // anonymous play uses 10.
    ops: ["addsub"],
    count: defaultRoundLen(),
    sheetId: "",
    questions: [],
    answers: [],
    qIndex: 0,
    revisited: false, // Finish already sent the child back to a blank once
    roundSeq: 0, // bumped on every start and on Home, so a late response is ignored
    report: { results: [], score: 0, total: 0, percent: 0 },
    opChoices: [
      { id: "addsub", label: "Add & Subtract", emoji: "➕", hint: "big numbers!" },
      { id: "mul", label: "Multiplication", emoji: "✖️", hint: "times tables" },
      { id: "div", label: "Division", emoji: "➗", hint: "no remainders" },
      { id: "frac", label: "Fractions", emoji: "🍕", hint: "answer like 3/4" },
      { id: "tables", label: "Times tables", emoji: "✖️", hint: "One table, 1 to 12, in order or mixed" },
    ],
    gradeHints: {
      1: "numbers up to 20",
      2: "numbers up to 100",
      3: "3-digit numbers, full times tables",
      4: "big numbers, tricky tables",
      5: "the toughest problems",
    },

    // Spelling dojo state.
    spellingCount: defaultRoundLen() > 10 ? 10 : 5,
    spellingFocus: "mixed",
    spellingWords: [],
    spellingRound: 0,
    spellingScore: 0,
    spellingResults: [],
    currentWord: null,
    guessedLetters: [],
    mistakes: 0,
    wholeWordGuess: "",
    showWholeWord: false, // the typing step is hidden until asked for
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
      if (this.child) return `Hi ${this.child.nickname}!`;
      return "Tap and go";
    },
    // init runs once Alpine mounts: a parent-portal link (#math/frac)
    // starts the round straight away, and a signed-in child's kata load
    // so the home screen can show their creature.
    init() {
      if (this.child && typeof this.loadKata === "function") this.loadKata();
      if (this.child) this.loadQuests();
      window.addEventListener("blur", () => this.holdEnd());
      if (this.hashStart) {
        this.hashStart = false;
        // Consume the hash so a reload or back does not start another round.
        if (window.history && window.history.replaceState) window.history.replaceState(null, "", window.location.pathname + window.location.search);
        if (this.hashTables) this.openTables();
        else if (this.subject === "spelling") this.openSpelling();
        else this.startSheet();
      }
    },
    chooseSubject(subject) {
      this.subject = subject;
      this.error = "";
    },
    goHome() {
      this.reset();
      if (this.child) this.loadQuests();
    },
    // Daily quests: three picked by the server for today. A finished round
    // completes a quest on the server; Home reloads them.
    async loadQuests() {
      try {
        const res = await fetch("/api/quests/today", { headers: { Accept: "application/json" } });
        const data = await res.json().catch(() => ({}));
        if (!res.ok) throw new Error(data.error || "Could not load today's quests.");
        this.quests = data;
        this.questsError = "";
      } catch (e) {
        this.questsError = e.message;
      }
    },
    nextQuest() {
      if (!this.quests) return null;
      return this.quests.quests.find((q) => !q.done) || null;
    },
    questState(q) {
      if (q.done) return "done";
      const next = this.nextQuest();
      return next && next.id === q.id ? "now" : "locked";
    },
    async startQuest(q) {
      if (!q || this.busy) return;
      this.error = "";
      if (this.child) {
        this.grade = this.child.grade;
        this.spellingGrade = this.child.grade;
      }
      if (q.kind === "math" && q.focus === "tables") {
        this.tableOrdered = true;
        await this.pickTable(q.table);
      } else if (q.kind === "math") {
        await this.pickMath(q.focus);
      } else {
        await this.pickSpelling(q.focus);
      }
    },
    // Buddy voice: the home creature speaks for itself. Cute designs are
    // bubbly; cool designs are short. Everything else stays factual.
    buddyVibe() {
      const k = this.heroKata();
      const d = k && typeof kataDesign === "function" ? kataDesign(k.id) : null;
      return d && d.vibe === "cool" ? "cool" : "cute";
    },
    buddySays() {
      const cool = this.buddyVibe() === "cool";
      if (this.quests && this.quests.all_done) return cool ? "All three done. Respect." : "We did all three quests! 🎉";
      const q = this.nextQuest();
      if (q) return cool ? `${q.label}. Let's go.` : `Let's do ${q.label.toLowerCase()} today!`;
      if (this.kata && this.kata.review_due > 0) return cool ? "Some words got away. Rescue them." : "Some words are waiting for you!";
      return cool ? "Pick something. I'm ready." : "What should we train today?";
    },
    buddyReacts(percent) {
      const cool = this.buddyVibe() === "cool";
      if (percent === 100) return cool ? "Flawless." : "PERFECT! I feel amazing!";
      if (percent >= 80) return cool ? "Strong work." : "Wow! I feel stronger!";
      if (percent >= 60) return cool ? "Good. Again?" : "Nice! Let's go again!";
      return cool ? "Tough one. We'll get it." : "That was hard. I'm still with you!";
    },
    // Anonymous grade picker on the home screen; one grade for both dojos.
    setGrade(g) {
      this.grade = g;
      this.spellingGrade = g;
      this.spellingFocus = "mixed";
    },
    openMath() {
      this.subject = "math";
      this.error = "";
      this.view = "math-pick";
      this.moveToTop("#math-pick-heading");
    },
    openTables() {
      this.error = "";
      this.view = "tables-pick";
      this.moveToTop("#tables-pick-heading");
    },
    openSpelling() {
      this.subject = "spelling";
      this.error = "";
      this.showPatterns = false;
      this.view = "spelling-pick";
      this.moveToTop("#spelling-pick-heading");
    },
    openPlay() {
      this.error = "";
      this.view = "play-pick";
      this.moveToTop("#play-pick-heading");
      if (typeof this.loadBattleCredits === "function") this.loadBattleCredits();
    },
    // pickMath starts a round of one operation at the profile's length.
    async pickMath(op) {
      this.ops = [op];
      this.count = defaultRoundLen();
      this.normalizeCount();
      await this.startSheet();
    },
    async surpriseMath() {
      const ops = ["addsub", "mul", "div", "frac"];
      await this.pickMath(ops[Math.floor(Math.random() * ops.length)]);
    },
    async pickTable(n) {
      this.table = n;
      this.ops = ["tables"];
      this.count = 12;
      await this.startSheet();
    },
    spellingPatterns() {
      return SPELLING_SKILLS[this.spellingGrade] || [];
    },
    async pickSpelling(focus) {
      this.spellingFocus = focus;
      this.spellingCount = defaultRoundLen() > 10 ? 10 : 5;
      await this.startSpelling();
    },
    // The most loved creature for the home screen: evolved beats caught
    // beats seen, then the most colored.
    heroKata() {
      if (!this.kata || !this.kata.entries) return null;
      const rank = { evolved: 3, caught: 2, seen: 1 };
      let best = null;
      for (const e of this.kata.entries) {
        if (!rank[e.state]) continue;
        if (!best || rank[e.state] > rank[best.state] || (rank[e.state] === rank[best.state] && e.fills > best.fills)) best = e;
      }
      return best;
    },
    heroHint() {
      const k = this.heroKata();
      if (!k) return "";
      if (k.state === "evolved") return "Evolved! Find the next one.";
      if (k.state === "caught") return "All colored. Keep training to evolve it!";
      return `${k.fills} of ${k.regions} colors. Train ${k.kind === "math" ? "math" : "spelling"} to add more!`;
    },
    kataFound() {
      return this.kata && this.kata.entries ? this.kata.entries.filter((e) => e.state !== "unknown").length : 0;
    },
    // Parent gate: hold the lock for three seconds. Letting go resets.
    holdStart() {
      if (this.holdTimer) return;
      const started = Date.now();
      this.holdTimer = setInterval(() => {
        this.holdPct = Math.min(100, ((Date.now() - started) / 3000) * 100);
        if (this.holdPct >= 100) {
          this.holdEnd();
          window.location.href = this.parentsHref();
        }
      }, 50);
    },
    holdEnd() {
      clearInterval(this.holdTimer);
      this.holdTimer = null;
      this.holdPct = 0;
    },
    // A synthesized click (screen reader, switch control) has no pointer
    // to hold, so it opens the parent page directly.
    holdClick(event) {
      if (event.detail === 0 && !this.holdTimer) window.location.href = this.parentsHref();
    },
    parentsHref() {
      return this.child ? "/family" : "/login";
    },
    async startTraining() {
      if (this.subject === "spelling") await this.startSpelling();
      else await this.startSheet();
    },

    // Math dojo methods.
    toggleOp(id) {
      // Times tables is a mode of its own: it cannot mix with other operations.
      if (id === "tables") {
        this.ops = this.ops.includes("tables") ? [] : ["tables"];
      } else {
        const without = this.ops.filter((o) => o !== id && o !== "tables");
        this.ops = this.ops.includes(id) ? without : [...without, id];
      }
      this.normalizeCount();
    },
    // Keep the count on the current mode's list after any mode change.
    normalizeCount() {
      if (!this.countChoices().includes(this.count)) this.count = this.tablesMode() ? 12 : defaultRoundLen();
    },
    tablesMode() {
      return this.ops.length === 1 && this.ops[0] === "tables";
    },
    countChoices() {
      return this.tablesMode() ? [12, 24] : [5, 10, 20, 30];
    },
    // The request body for a math sheet, shared by the free and signed-in paths.
    sheetRequest() {
      const body = { ops: this.ops, grade: this.grade, count: this.count };
      if (this.tablesMode()) {
        body.table = this.table;
        body.ordered = this.tableOrdered;
      }
      return body;
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
      if (p >= 90) return "Amazing! Almost perfect!";
      if (p >= 80) return "Great job! Keep training!";
      if (p >= 60) return "Good effort! Practice makes perfect!";
      return "Every ninja starts somewhere. Try again!";
    },
    async post(url, body) {
      const res = await fetch(url, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        const err = new Error(data.error || "Something went wrong — try again.");
        err.status = res.status; // callers distinguish a refusal from a dropped request
        throw err;
      }
      return data;
    },
    async startSheet() {
      if (this.busy) return; // a double tap must not start two rounds
      this.busy = true;
      this.error = "";
      this.reward = null;
      const seq = ++this.roundSeq;
      try {
        let data;
        if (this.child) {
          this.roundId = newRoundId();
          data = await this.post("/api/round/start", Object.assign({ round_id: this.roundId, kind: "math" }, this.sheetRequest()));
          data = { id: data.sheet_id, questions: data.questions };
        } else {
          this.roundId = "";
          data = await this.post("/api/sheet", this.sheetRequest());
        }
        if (seq !== this.roundSeq) return; // the child went Home while this loaded
        this.sheetId = data.id;
        this.questions = data.questions;
        this.answers = data.questions.map(() => "");
        this.qIndex = 0;
        this.revisited = false;
        this.view = "math-play";
        this.moveToTop();
      } catch (e) {
        if (seq === this.roundSeq) this.error = e.message;
      } finally {
        if (seq === this.roundSeq) this.busy = false; // a stale request must not unlock a newer one
      }
    },
    // Number pad. Answers are strings, as the sheet API expects; a fraction
    // question gets a "/" key. Up to 7 characters keeps 5-digit sums and
    // fractions like 12/100 typeable without runaway input.
    padPress(k) {
      const cur = this.answers[this.qIndex] || "";
      if (cur.length >= 7) return;
      if (k === "/" && (cur === "" || cur.includes("/"))) return;
      this.answers[this.qIndex] = cur + k;
    },
    padDelete() {
      const cur = this.answers[this.qIndex] || "";
      this.answers[this.qIndex] = cur.slice(0, -1);
    },
    async padGo() {
      if (!this.answers[this.qIndex]) return; // nothing typed: Skip is the way past
      await this.advance();
    },
    async skipQuestion() {
      await this.advance();
    },
    prevQuestion() {
      if (this.qIndex > 0) this.qIndex -= 1;
    },
    async advance() {
      if (this.qIndex + 1 < this.questions.length) {
        this.qIndex += 1;
        return;
      }
      // Last one: the first Finish with blanks goes back to the first
      // blank; a second Finish grades anyway so a child who wants to skip
      // a question is never trapped.
      const blank = this.answers.findIndex((a) => !a);
      if (blank >= 0 && blank !== this.qIndex && !this.revisited) {
        this.revisited = true;
        this.qIndex = blank;
        return;
      }
      await this.submitSheet();
    },
    // Physical keyboard on a laptop: digits, slash, backspace, enter.
    mathKey(event) {
      if (this.view !== "math-play" || event.defaultPrevented || event.metaKey || event.ctrlKey || event.altKey) return;
      const tag = document.activeElement ? document.activeElement.tagName : "";
      // Text fields and links keep every key. A focused button (the one
      // just tapped) keeps Enter and Space, but digits still type.
      if (tag === "INPUT" || tag === "TEXTAREA" || tag === "A") return;
      if (tag === "BUTTON" && (event.key === "Enter" || event.key === " ")) return;
      if (/^[0-9]$/.test(event.key)) this.padPress(event.key);
      else if (event.key === "/") this.padPress("/");
      else if (event.key === "Backspace") this.padDelete();
      else if (event.key === "Enter") this.padGo();
      else return;
      event.preventDefault();
    },
    async submitSheet() {
      if (this.busy) return;
      this.busy = true;
      this.error = "";
      const seq = this.roundSeq;
      try {
        if (this.roundId) {
          const data = await this.post("/api/round/finish", { round_id: this.roundId, answers: this.answers });
          if (seq !== this.roundSeq) return; // quit while grading: stay Home
          this.report = { results: data.results, score: data.score, total: data.total, percent: data.percent };
          this.reward = data.reward;
          if (typeof this.loadBattleCredits === "function") this.loadBattleCredits();
        } else {
          const report = await this.post("/api/grade", { id: this.sheetId, answers: this.answers });
          if (seq !== this.roundSeq) return;
          this.report = report;
        }
        this.view = "math-results";
        this.moveToTop("#math-results-heading");
        if (this.report.percent === 100) confettiBurst();
      } catch (e) {
        if (seq === this.roundSeq) this.error = e.message;
      } finally {
        if (seq === this.roundSeq) this.busy = false;
      }
    },

    // Spelling dojo methods.
    spellingGradeHint() {
      return this.spellingGradeHints[this.spellingGrade] || "";
    },
    spellingSkillChoices() {
      const choices = [...(SPELLING_SKILLS[this.spellingGrade] || []), SIGHT_WORD_SKILL];
      if (this.child) choices.push(REVIEW_SKILL);
      return choices;
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
    async startSpelling() {
      if (this.busy) return; // a double tap must not start two rounds
      this.subject = "spelling";
      this.error = "";
      this.reward = null;
      const seq = ++this.roundSeq;
      if (this.child) {
        this.busy = true;
        try {
          const roundId = newRoundId();
          const data = await this.post("/api/round/start", {
            round_id: roundId, kind: "spelling", focus: this.spellingFocus,
            grade: this.spellingGrade, count: this.spellingCount,
          });
          if (seq !== this.roundSeq) return; // the child went Home while this loaded
          this.roundId = roundId;
          this.spellingWords = data.words;
        } catch (e) {
          if (seq !== this.roundSeq) return;
          this.error = e.message;
          this.roundId = "";
          return;
        } finally {
          if (seq === this.roundSeq) this.busy = false;
        }
      } else {
        this.roundId = "";
        this.spellingWords = this.selectSpellingWords();
      }
      this.guessLog = this.spellingWords.map(() => []);
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
      this.showWholeWord = false; // each word starts on the tiles
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
      this.logGuess("letter", letter);
      if (this.currentWord.word.toUpperCase().includes(letter)) {
        if (this.isWordRevealed()) this.finishSpellingRound(true);
        else this.statusMessage = `Nice! ${letter} is in the word.`;
      } else {
        this.mistakes += 1;
        if (this.mistakes >= 6) this.finishSpellingRound(false);
        else this.statusMessage = `No ${letter} this time. ${this.triesLeft()} tries left.`;
      }
    },
    // The typing step: the only place the system keyboard is welcome.
    openWholeWord() {
      this.showWholeWord = true;
      this.$nextTick(() => { const el = document.getElementById("whole-word"); if (el) el.focus(); });
    },
    guessWholeWord() {
      if (this.roundDone || !this.wholeWordGuess.trim()) return;
      const guess = this.wholeWordGuess.trim().toLowerCase();
      // Log what was compared, not what was typed: the server requires the
      // exact lowercase word, and this is what the browser judged.
      this.logGuess("word", guess);
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
    logGuess(kind, value) {
      const log = this.guessLog[this.spellingRound];
      if (log) log.push({ kind, value });
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
    async nextSpellingWord() {
      if (!this.roundDone) return;
      if (this.spellingRound + 1 >= this.spellingWords.length) {
        // If the server could not record the round, stay here so the
        // child can tap again; the same roundId makes the retry harmless.
        if (this.roundId && !(await this.finishSpellingRound_())) return;
        this.view = "spelling-results";
        this.moveToTop("#spelling-results-heading");
        if (this.spellingScore === this.spellingWords.length) confettiBurst();
        return;
      }
      this.spellingRound += 1;
      this.loadSpellingWord();
      this.moveToTop("#spelling-word-heading");
    },
    // The server replays the guess log and decides the reward. Its verdict
    // is what the results show; the local score was only for instant feedback.
    async finishSpellingRound_() {
      this.busy = true;
      this.error = "";
      const seq = this.roundSeq;
      try {
        const data = await this.post("/api/round/finish", { round_id: this.roundId, guesses: this.guessLog });
        if (seq !== this.roundSeq) return false; // quit while saving: stay Home
        this.reward = data.reward;
        if (typeof this.loadBattleCredits === "function") this.loadBattleCredits();
        this.spellingScore = data.score;
        this.spellingResults = data.word_results.map(({ word, won }) => ({ word, won }));
        return true;
      } catch (e) {
        if (seq !== this.roundSeq) return false;
        this.error = e.message;
        this.statusMessage = "Could not save this round. Tap again to retry.";
        return false;
      } finally {
        if (seq === this.roundSeq) this.busy = false;
      }
    },
    rewardSummary() {
      if (!this.reward) return "";
      const parts = [];
      if (this.reward.fills > 0) parts.push(`+${this.reward.fills} energy`);
      if (this.reward.mosaic_cells > 0) parts.push(`+${this.reward.mosaic_cells} mosaic tiles`);
      if (this.reward.evolved && this.reward.evolved.length) parts.push("a skill evolved!");
      if (this.reward.review_due > 0) parts.push(`${this.reward.review_due} words to review`);
      return parts.join(" · ");
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
      // While the typing step is open, every key belongs to the word field.
      if (this.view !== "spelling-game" || this.roundDone || this.showWholeWord || event.metaKey || event.ctrlKey || event.altKey) return;
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
      const audio = this.wordAudio;
      const fallBackToSpeech = () => {
        // A late rejection (after reset/next word) must not speak the new answer.
        if (fellBack || this.wordAudio !== audio || this.roundDone) return;
        fellBack = true;
        this.speakWithBrowserVoice();
      };
      audio.onerror = fallBackToSpeech;
      audio.currentTime = 0;
      const playback = audio.play();
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
      this.view = "home";
      this.error = "";
      this.roundSeq += 1; // anything still loading belongs to the old round
      this.busy = false;
      this.questions = [];
      this.answers = [];
      // Training a kata from another grade must not change the child's
      // grade or round length for the next thing they pick.
      if (this.child) {
        this.grade = this.child.grade;
        this.spellingGrade = this.child.grade;
      }
      this.count = defaultRoundLen();
      this.spellingCount = defaultRoundLen() > 10 ? 10 : 5;
      this.moveToTop();
    },
  };
  applyHash(core);
  return Object.assign(
    core,
    typeof kataMixin === "function" ? kataMixin() : {},
    typeof paintMixin === "function" ? paintMixin() : {},
    typeof battleMixin === "function" ? battleMixin() : {}
  );
}

// applyHash opens the dojo on a mode named in the URL hash, e.g. "#math/frac"
// or "#math/tables", which is how parent-portal recommendations link in.
// Unknown modes are ignored.
function applyHash(state) {
  const hash = typeof location !== "undefined" ? location.hash : "";
  const m = /^#(math|spelling)(?:\/([a-z,]+))?$/.exec(hash || "");
  if (!m) return;
  state.subject = m[1];
  if (m[1] === "math" && m[2]) {
    const known = state.opChoices.map((o) => o.id);
    const ops = m[2].split(",").filter((op) => known.includes(op));
    if (ops.includes("tables")) {
      state.hashStart = true;
      state.hashTables = true; // init() opens the table picker
    } else if (ops.length) {
      state.ops = ops;
      state.hashStart = true; // init() starts the round
    }
  } else if (m[1] === "spelling") {
    state.hashStart = true;
  }
}

// defaultRoundLen is the parent's setting from the profile, or 10 when
// nobody is signed in.
function defaultRoundLen() {
  const child = readChild();
  return child && [5, 10, 20].includes(child.round) ? child.round : 10;
}

// readChild pulls the signed-in child from the root element's data
// attributes, which html/template escapes. Absent when anonymous or in tests.
// Signed-in children start at their own grade; anonymous play keeps grade 1.
function defaultGrades() {
  const child = readChild();
  const grade = child ? child.grade : 1;
  return { grade, spellingGrade: grade };
}

// Review focus exists only for signed-in children; the server picks the
// words this child keeps missing.
const REVIEW_SKILL = { id: "review", label: "Words I keep missing", tip: "The server picks the words you have missed lately. Rescue them and they leave the list." };

function readChild() {
  const root = typeof document !== "undefined" && document.querySelector ? document.querySelector("[data-child-id]") : null;
  if (!root) return null;
  return {
    id: Number(root.dataset.childId),
    nickname: root.dataset.childNickname || "",
    grade: Number(root.dataset.childGrade) || 1,
    round: Number(root.dataset.childRound) || 10,
  };
}

// newRoundId is the idempotency key for a round; a retried finish is a
// duplicate, not a double reward.
function newRoundId() {
  if (typeof crypto !== "undefined" && crypto.randomUUID) return crypto.randomUUID();
  return Array.from({ length: 32 }, () => Math.floor(Math.random() * 16).toString(16)).join("");
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
