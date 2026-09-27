// Kata battles: turn-based quiz duels. The server holds the state and the
// answers; the client shows two creatures, hearts, the current item, and
// sends one answer per turn. Continuations capture the battle id and turn
// so a stale response can never act on a newer battle.
function battleMixin() {
  return {
    battle: null,
    battleId: "",
    battleAnswer: "",
    battleBusy: false,
    battleError: "",
    battleCredits: null,
    battleRequest: 0,
    battleSeq: 0,

    async loadBattleCredits() {
      try {
        const res = await fetch("/api/battle/credits", { headers: { Accept: "application/json" } });
        const data = await res.json().catch(() => ({}));
        this.battleCredits = res.ok ? data : null;
      } catch {
        this.battleCredits = null;
      }
    },
    canBattle() {
      return !!(this.child && this.battleCredits && this.battleCredits.enabled && this.battleCredits.credits > 0);
    },
    // Start with the first creature this round touched, else the child's
    // most-colored known creature from the index.
    battleCreatureId() {
      const touched = typeof this.rewardCreatures === "function" ? this.rewardCreatures() : [];
      if (touched.length) return touched[0].id;
      const known = this.kata ? this.kata.entries.filter((e) => e.state !== "unknown") : [];
      known.sort((a, b) => b.fills - a.fills);
      return known.length ? known[0].id : "";
    },
    // The pending battle (id + creature) is persisted before the request so
    // a dropped response or a reload retries the same id: the server replays
    // it and the credit is spent once. Cleared when the battle ends.
    pendingKey() {
      return `sd_pending_battle:${this.child ? this.child.id : "anon"}`;
    },
    pendingBattle() {
      try {
        const raw = sessionStorage.getItem(this.pendingKey());
        return raw ? JSON.parse(raw) : null;
      } catch {
        return null;
      }
    },
    // Returns false when the browser refuses storage: without a durable id
    // a retry could spend a second credit, so the start does not proceed.
    setPendingBattle(value) {
      try {
        if (value) sessionStorage.setItem(this.pendingKey(), JSON.stringify(value));
        else sessionStorage.removeItem(this.pendingKey());
        return true;
      } catch {
        return !value;
      }
    },
    async startBattle(creatureId) {
      if (this.battleBusy) return;
      const pending = this.pendingBattle();
      const id = (pending && pending.creature) || creatureId || this.battleCreatureId();
      if (!id) {
        this.battleError = "Find a kata first by training its skill.";
        return;
      }
      const battleId = (pending && pending.id) || newRoundId();
      if (!this.setPendingBattle({ id: battleId, creature: id })) {
        this.battleError = "This browser cannot remember a battle. Try again in a normal window.";
        return;
      }
      this.battleBusy = true;
      this.battleError = "";
      const token = (this.battleSeq = (this.battleSeq || 0) + 1); // unique per request
      this.battleRequest = token;
      try {
        const st = await this.post("/api/battle/start", { battle_id: battleId, creature_id: id });
        if (this.battleRequest !== token) return;
        this.battleId = battleId;
        this.battle = st;
        this.battleAnswer = "";
        if (st.done) this.setPendingBattle(null);
        this.view = "battle";
        this.moveToTop("#battle-heading");
        await this.loadBattleCredits();
      } catch (e) {
        if (this.battleRequest !== token) return;
        this.battleError = e.message;
        this.error = e.message;
      } finally {
        if (this.battleRequest === token) this.battleBusy = false;
      }
    },
    async submitBattleAnswer() {
      if (!this.battle || this.battle.done || this.battleBusy || !this.battleAnswer.trim()) return;
      const battleId = this.battleId;
      const turn = this.battle.turn;
      this.battleBusy = true;
      this.battleError = "";
      try {
        const st = await this.post("/api/battle/turn", { battle_id: battleId, turn, answer: this.battleAnswer.trim() });
        // Ignore a response for a battle that is no longer on screen.
        if (this.battleId !== battleId) return;
        this.battle = st;
        this.battleAnswer = "";
        if (st.done) this.setPendingBattle(null);
        if (st.done && st.won) confettiBurst();
        requestAnimationFrame(() => document.querySelector("#battle-answer")?.focus());
      } catch (e) {
        if (this.battleId === battleId) this.battleError = e.message;
      } finally {
        if (this.battleId === battleId) this.battleBusy = false;
      }
    },
    hearts(hp) {
      return Array.from({ length: 5 }, (_, i) => i < hp);
    },
    fighterSVG(f, size) {
      if (!f || typeof KataSVG === "undefined") return "";
      return KataSVG.creature(f.seed, { palette: f.palette, regions: f.regions, fills: f.fills, evolved: f.evolved, name: f.name, size: size || 120 });
    },
    leaveBattle() {
      // Invalidate any in-flight request and hand the buttons back.
      this.battleId = "";
      this.battleRequest = 0;
      this.battle = null;
      this.battleBusy = false;
      this.reset();
    },
    // A battle interrupted by a reload can be resumed from the index.
    hasPendingBattle() {
      return !!(this.child && this.pendingBattle());
    },
  };
}
