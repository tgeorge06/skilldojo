// Kata collection: the index of creatures and the reveal on results. Mixed
// into dojo() by app.js. Everything here needs a signed-in child; the index
// endpoint refuses anonymous callers and the UI hides the entry points.
function kataMixin() {
  return {
    kata: null,
    kataLoading: false,
    kataError: "",
    kataGrade: null,

    async openKata() {
      this.error = "";
      this.view = "kata-index";
      this.moveToTop("#kata-heading");
      await Promise.all([
        this.loadKata(),
        typeof this.loadMosaic === "function" ? this.loadMosaic() : null,
        typeof this.loadBattleCredits === "function" ? this.loadBattleCredits() : null,
      ]);
    },
    async loadKata() {
      this.kataLoading = true;
      this.kataError = "";
      try {
        const res = await fetch("/api/kata/index", { headers: { Accept: "application/json" } });
        const data = await res.json().catch(() => ({}));
        if (!res.ok) throw new Error(data.error || "Could not load your kata.");
        this.kata = data;
        if (this.kataGrade === null) this.kataGrade = data.child_grade || 1;
      } catch (e) {
        this.kataError = e.message;
      } finally {
        this.kataLoading = false;
      }
    },
    kataGrades() {
      return [1, 2, 3, 4, 5];
    },
    kataEntries(grade) {
      if (!this.kata) return [];
      return this.kata.entries.filter((e) => e.grade === grade);
    },
    kataProgress(grade) {
      const t = this.kata && this.kata.by_grade ? this.kata.by_grade[grade] : null;
      return t ? { done: t[0], total: t[1] } : { done: 0, total: 0 };
    },
    kataName(entry) {
      return entry.state === "unknown" ? "???" : entry.name;
    },
    kataStateLabel(entry) {
      switch (entry.state) {
        case "evolved": return "Evolved";
        case "caught": return "Caught";
        case "seen": return `${entry.fills} of ${entry.regions} colored`;
        default: return "Not found yet";
      }
    },
    kataSVG(entry, size) {
      if (typeof KataSVG === "undefined") return "";
      return KataSVG.creature(entry.seed, {
        palette: entry.palette, regions: entry.regions, fills: entry.fills,
        evolved: entry.state === "evolved", name: this.kataName(entry), size: size || 120,
        silhouette: entry.state === "unknown",
      });
    },
    kataAboveGrade(entry) {
      return this.child && entry.grade > this.child.grade;
    },
    // Train here: point the setup at this creature's skill and start. A
    // failure to start surfaces on the index, where the child still is.
    async trainHere(entry) {
      if (entry.kind === "math") {
        this.subject = "math";
        this.ops = [entry.focus];
        this.grade = entry.grade;
      } else {
        this.subject = "spelling";
        this.spellingFocus = entry.focus;
        this.spellingGrade = entry.grade;
      }
      await this.startTraining();
      if (this.view === "kata-index" && this.error) this.kataError = this.error;
    },
    async trainReview() {
      this.subject = "spelling";
      this.spellingFocus = "review";
      this.spellingGrade = this.child ? this.child.grade : this.spellingGrade;
      this.spellingCount = 5;
      await this.startTraining();
      if (this.view === "kata-index" && this.error) this.kataError = this.error;
    },
    // Creatures touched by the round just finished, for the reveal.
    rewardCreatures() {
      if (!this.reward || !this.reward.creatures) return [];
      try {
        const list = typeof this.reward.creatures === "string" ? JSON.parse(this.reward.creatures) : this.reward.creatures;
        return Array.isArray(list) ? list : [];
      } catch {
        return [];
      }
    },
    revealSVG(t) {
      if (typeof KataSVG === "undefined") return "";
      return KataSVG.creature(t.seed, {
        palette: t.palette, regions: t.regions, fills: t.fills, evolved: t.state === "evolved", name: t.name, size: 96,
      });
    },
    revealLabel(t) {
      if (t.evolved) return `${t.name} evolved!`;
      if (t.caught) return `You caught ${t.name}!`;
      if (t.newly_seen) return `You found ${t.name}!`;
      if (t.fills_added > 0) return `${t.name}: +${t.fills_added} colored`;
      return `${t.name} is watching`;
    },
  };
}
