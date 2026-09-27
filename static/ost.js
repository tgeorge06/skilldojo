// Practice test page (Alpine component). One question per screen, every
// answer saved to the server as it changes, so a closed tab resumes.
function ostTest() {
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
      const a = this.answers[id] || {};
      try {
        await this.post("/api/ost/answer", { attempt_id: this.attemptId, item_id: id, choices: a.choices || [], text: a.text || "" });
        if (Object.keys(this.saveTimers).length === 0) this.saveState = "Saved";
        this.error = "";
        return true;
      } catch (err) {
        this.saveState = "";
        this.error = "Could not save that answer: " + err.message;
        return false;
      }
    },
    // flush sends every answer still waiting on its debounce.
    async flush() {
      const pending = Object.keys(this.saveTimers);
      const results = await Promise.all(pending.map((id) => this.save(id)));
      return results.every(Boolean);
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
