// Practice test page (Alpine component). One question per screen, every
// answer saved to the server as it changes, so a closed tab resumes.
function ostTest() {
  // Per-item save chains and the ids whose last save failed. Kept out of
  // the Alpine object so proxies never wrap promises.
  const inflight = new Map();
  const failed = new Set();
  return {
    view: "loading",
    grade: Number(document.querySelector("[data-test-grade]")?.dataset.testGrade) || 3,
    attemptId: "",
    items: [],
    answers: {},
    index: 0,
    report: { score: 0, total: 0, percent: 0, level: "", categories: [], items: [] },
    showReview: false,
    saveState: "",
    busy: false,
    error: "",
    // One debounce timer per item, so moving on to the next question
    // never cancels the previous answer's save.
    saveTimers: {},

    async start() {
      try {
        const a = await this.post("/api/ost/start", { grade: this.grade });
        this.load(a);
      } catch (err) {
        this.error = err.message;
      }
    },
    load(a) {
      this.attemptId = a.id;
      this.items = a.items || [];
      this.answers = a.answers || {};
      if (a.report) {
        this.report = a.report;
        this.view = "done";
        return;
      }
      // Resume at the first unanswered question.
      const first = this.items.findIndex((it) => !this.answers[it.id]);
      this.index = first < 0 ? this.items.length - 1 : first;
      this.view = "question";
    },
    current() {
      return this.items[this.index] || null;
    },
    answeredCount() {
      return this.items.filter((it) => this.answers[it.id]).length;
    },
    isChosen(i) {
      const a = this.answers[this.current().id];
      return !!a && Array.isArray(a.choices) && a.choices.includes(i);
    },
    numberText() {
      const a = this.answers[this.current().id];
      return a ? a.text || "" : "";
    },
    choose(choices) {
      this.setAnswer({ choices });
    },
    toggleChoice(i) {
      const a = this.answers[this.current().id];
      const chosen = a && Array.isArray(a.choices) ? a.choices.slice() : [];
      const at = chosen.indexOf(i);
      if (at >= 0) chosen.splice(at, 1);
      else chosen.push(i);
      chosen.sort((x, y) => x - y);
      this.setAnswer({ choices: chosen });
    },
    typeNumber(text) {
      this.setAnswer({ text: text.slice(0, 20) });
    },
    setAnswer(a) {
      const id = this.current().id;
      const empty = (!a.choices || a.choices.length === 0) && !(a.text || "").trim();
      if (empty) delete this.answers[id];
      else this.answers[id] = a;
      this.answers = { ...this.answers };
      this.queueSave(id);
    },
    queueSave(id) {
      clearTimeout(this.saveTimers[id]);
      this.saveState = "Saving…";
      this.saveTimers[id] = setTimeout(() => this.save(id), 400);
    },
    async save(id) {
      clearTimeout(this.saveTimers[id]);
      delete this.saveTimers[id];
      // Serialize saves per item so an older request cannot land after a
      // newer one. Promises live outside the reactive object.
      const prior = inflight.get(id) || Promise.resolve();
      const run = prior.then(() => this.send(id));
      inflight.set(id, run);
      const ok = await run;
      if (inflight.get(id) === run) inflight.delete(id);
      return ok;
    },
    async send(id) {
      const a = this.answers[id] || {};
      try {
        await this.post("/api/ost/answer", { attempt_id: this.attemptId, item_id: id, choices: a.choices || [], text: a.text || "" });
        failed.delete(id);
        if (Object.keys(this.saveTimers).length === 0 && inflight.size <= 1) this.saveState = "Saved";
        this.error = "";
        return true;
      } catch (err) {
        failed.add(id);
        this.saveState = "";
        this.error = "Could not save that answer: " + err.message;
        return false;
      }
    },
    // flush sends every answer still waiting on its debounce, retries any
    // that failed, and waits for everything in flight. False if any failed.
    async flush() {
      const ids = new Set([...Object.keys(this.saveTimers), ...failed]);
      const results = await Promise.all([...ids].map((id) => this.save(id)));
      await Promise.all([...inflight.values()]);
      return results.every(Boolean) && failed.size === 0;
    },
    prev() {
      if (this.index > 0) this.index -= 1;
      window.scrollTo({ top: 0 });
    },
    next() {
      if (this.index < this.items.length - 1) this.index += 1;
      window.scrollTo({ top: 0 });
    },
    async submit() {
      if (this.busy) return;
      this.busy = true;
      try {
        if (!(await this.flush())) return; // the error is already on screen
        const a = await this.post("/api/ost/submit", { attempt_id: this.attemptId });
        this.load(a);
        window.scrollTo({ top: 0 });
      } catch (err) {
        this.error = err.message;
      } finally {
        this.busy = false;
      }
    },
    correctText(it) {
      if (it.type === "number") return it.numeric;
      return (it.answer || []).map((i) => it.choices[i]).join(", ");
    },
    givenText(it) {
      if (it.type === "number") return it.given.text || "";
      return (it.given.choices || []).map((i) => it.choices[i]).join(", ");
    },
    async post(url, body) {
      const res = await fetch(url, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) throw new Error(data.error || "Something went wrong");
      return data;
    },
  };
}
