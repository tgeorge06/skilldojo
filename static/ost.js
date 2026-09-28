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
    terms: {},
    explained: "",
    canSpeak: "speechSynthesis" in window && "SpeechSynthesisUtterance" in window,
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
      this.terms = a.terms || {};
      this.explained = "";
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
    // Number pad, matching the dojo: digits, a decimal point, a fraction bar.
    padPress(k) {
      const cur = this.numberText();
      if (cur.length >= 8) return;
      if ((k === "/" || k === ".") && (cur === "" || cur.includes("/") || cur.includes("."))) return; // one bar or one point, never both
      this.typeNumber(cur + k);
    },
    padDelete() {
      this.typeNumber(this.numberText().slice(0, -1));
    },
    padKey(event) {
      if (this.view !== "question" || !this.current() || this.current().type !== "number" || event.defaultPrevented || event.metaKey || event.ctrlKey || event.altKey) return;
      const tag = document.activeElement ? document.activeElement.tagName : "";
      if (tag === "INPUT" || tag === "TEXTAREA" || tag === "A") return;
      if (tag === "BUTTON" && (event.key === "Enter" || event.key === " ")) return;
      if (/^[0-9./]$/.test(event.key)) this.padPress(event.key);
      else if (event.key === "Backspace") this.padDelete();
      else return;
      event.preventDefault();
    },
    // Glossary: split a prompt into plain text and tappable test words.
    // Longest terms first so "line plot" wins over "line".
    promptParts(text) {
      const words = Object.keys(this.terms).sort((a, b) => b.length - a.length);
      if (!words.length || !text) return [{ text: text || "" }];
      const re = new RegExp("\\b(" + words.map((w) => w.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")).join("|") + ")\\b", "gi");
      const parts = [];
      let last = 0;
      for (const m of text.matchAll(re)) {
        if (m.index > last) parts.push({ text: text.slice(last, m.index) });
        parts.push({ text: m[0], term: m[0].toLowerCase() });
        last = m.index + m[0].length;
      }
      if (last < text.length) parts.push({ text: text.slice(last) });
      return parts;
    },
    // Glossary words inside an answer choice, for a "?" chip beside it
    // (a button cannot live inside the choice's label).
    choiceTerms(text) {
      const seen = new Set();
      return this.promptParts(text).filter((p) => p.term && !seen.has(p.term) && seen.add(p.term)).map((p) => p.term);
    },
    explain(term) {
      this.explained = this.explained === term ? "" : term;
    },
    // Read the question and its choices with the browser voice.
    readAloud() {
      if (!this.canSpeak || !this.current()) return;
      const it = this.current();
      let text = it.prompt;
      if (it.figure && it.figure.alt) text += " " + it.figure.alt;
      if (it.choices && it.choices.length) {
        text += " " + it.choices.map((c, i) => `Choice ${"ABCD"[i] || i + 1}: ${c}.`).join(" ");
      }
      window.speechSynthesis.cancel();
      const u = new SpeechSynthesisUtterance(text.replace(/×/g, " times ").replace(/÷/g, " divided by ").replace(/−/g, " minus ").replace(/(\d+)\/(\d+)/g, "$1 over $2"));
      u.lang = "en-US";
      u.rate = 0.9;
      window.speechSynthesis.speak(u);
    },
    // Small pictures, drawn where the real test would show one.
    figureSVG(f) {
      const esc = (t) => String(t).replace(/[&<>"]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" }[c]));
      if (f.kind === "rect") {
        const w = 200, h = Math.max(60, Math.min(140, Math.round((200 * f.b) / Math.max(f.a, 1))));
        return `<svg width="300" height="${h + 44}" viewBox="0 0 300 ${h + 44}"><rect x="20" y="8" width="${w}" height="${h}" fill="#d7f5ea" stroke="#218a68" stroke-width="3" rx="4"/>` +
          `<text x="${20 + w / 2}" y="${h + 34}" text-anchor="middle" font-size="18" font-weight="700" fill="currentColor">${esc(f.label_a || f.a)}</text>` +
          `<text x="${20 + w + 10}" y="${8 + h / 2 + 6}" text-anchor="start" font-size="18" font-weight="700" fill="currentColor">${esc(f.label_b || f.b)}</text></svg>`;
      }
      if (f.kind === "grid") {
        const cell = Math.min(24, Math.floor(240 / Math.max(f.a, f.b)));
        let cells = "";
        for (let y = 0; y < f.b; y += 1) for (let x = 0; x < f.a; x += 1) cells += `<rect x="${x * cell}" y="${y * cell}" width="${cell}" height="${cell}" fill="#d7f5ea" stroke="#218a68" stroke-width="1.5"/>`;
        return `<svg width="${f.a * cell + 2}" height="${f.b * cell + 2}" viewBox="-1 -1 ${f.a * cell + 2} ${f.b * cell + 2}">${cells}</svg>`;
      }
      if (f.kind === "parts") {
        const w = 240, pw = w / f.a;
        let bars = "";
        for (let i = 0; i < f.a; i += 1) bars += `<rect x="${i * pw}" y="0" width="${pw}" height="48" fill="${i < f.b ? "#7c3aed" : "#ffffff"}" stroke="#4c1d95" stroke-width="2"/>`;
        return `<svg width="${w + 4}" height="52" viewBox="-2 -2 ${w + 4} 52">${bars}</svg>`;
      }
      if (f.kind === "bars") {
        const names = f.names || [], vals = f.values || [], scale = f.a || 1;
        const maxV = Math.max(scale, ...vals), rows = Math.ceil(maxV / scale), h = rows * 22, w = names.length * 60 + 40;
        let out = "";
        for (let g = 0; g <= rows; g += 1) {
          const y = 10 + h - g * 22;
          out += `<line x1="36" y1="${y}" x2="${w}" y2="${y}" stroke="currentColor" stroke-opacity="0.25"/><text x="30" y="${y + 5}" text-anchor="end" font-size="12" fill="currentColor">${g * scale}</text>`;
        }
        names.forEach((n, i) => {
          const bh = (vals[i] / scale) * 22, x = 46 + i * 60;
          out += `<rect x="${x}" y="${10 + h - bh}" width="40" height="${bh}" fill="#7c3aed"/><text x="${x + 20}" y="${h + 26}" text-anchor="middle" font-size="12" font-weight="700" fill="currentColor">${esc(n)}</text>`;
        });
        return `<svg width="${w + 6}" height="${h + 34}" viewBox="0 0 ${w + 6} ${h + 34}">${out}</svg>`;
      }
      if (f.kind === "lineplot") {
        const vals = f.values || [], n = f.a || 8, step = 260 / n, labels = ["0", "¼", "½", "¾", "1", "1¼", "1½", "1¾", "2"];
        let out = `<line x1="20" y1="70" x2="${20 + 260}" y2="70" stroke="currentColor" stroke-width="3"/>`;
        for (let i = 0; i <= n; i += 1) {
          const x = 20 + i * step;
          out += `<line x1="${x}" y1="62" x2="${x}" y2="78" stroke="currentColor" stroke-width="2"/><text x="${x}" y="94" text-anchor="middle" font-size="12" font-weight="700" fill="currentColor">${labels[i] || ""}</text>`;
          for (let k = 0; k < (vals[i] || 0); k += 1) out += `<text x="${x}" y="${56 - k * 16}" text-anchor="middle" font-size="16" font-weight="800" fill="#ef476f">X</text>`;
        }
        return `<svg width="300" height="100" viewBox="0 0 300 100">${out}<text x="150" y="99" font-size="1" fill="none">inches</text></svg>`;
      }
      if (f.kind === "numberline") {
        const w = 240, step = w / f.a;
        let ticks = "";
        for (let i = 0; i <= f.a; i += 1) ticks += `<line x1="${20 + i * step}" y1="18" x2="${20 + i * step}" y2="38" stroke="currentColor" stroke-width="2"/>`;
        return `<svg width="280" height="60" viewBox="0 0 280 60"><line x1="20" y1="28" x2="${20 + w}" y2="28" stroke="currentColor" stroke-width="3"/>${ticks}` +
          `<circle cx="${20 + f.b * step}" cy="28" r="8" fill="#ef476f"/><text x="20" y="56" text-anchor="middle" font-size="16" font-weight="700" fill="currentColor">0</text><text x="${20 + w}" y="56" text-anchor="middle" font-size="16" font-weight="700" fill="currentColor">1</text></svg>`;
      }
      return "";
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
      this.explained = "";
      if (this.canSpeak) window.speechSynthesis.cancel();
      window.scrollTo({ top: 0 });
    },
    next() {
      if (this.index < this.items.length - 1) this.index += 1;
      this.explained = "";
      if (this.canSpeak) window.speechSynthesis.cancel();
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
