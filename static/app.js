// SkillDojo front-end state (Alpine component).
function dojo() {
  return {
    view: "setup",
    busy: false,
    error: "",
    ops: ["addsub"],
    grade: 1,
    count: 10,
    sheetId: "",
    questions: [],
    answers: [],
    grade: { results: [], score: 0, total: 0, percent: 0 },

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
      const p = this.grade.percent;
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
        this.view = "sheet";
        window.scrollTo(0, 0);
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
        this.grade = await this.post("/api/grade", { id: this.sheetId, answers: this.answers });
        this.view = "results";
        window.scrollTo(0, 0);
        if (this.grade.percent === 100) confettiBurst();
      } catch (e) {
        this.error = e.message;
      } finally {
        this.busy = false;
      }
    },

    reset() {
      this.view = "setup";
      this.error = "";
      this.questions = [];
      this.answers = [];
      window.scrollTo(0, 0);
    },
  };
}

// Tiny dependency-free confetti for a perfect score.
function confettiBurst() {
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
        {
          transform:
            "translateY(" + (window.innerHeight + 40) + "px) rotate(" + (360 + Math.random() * 720) + "deg)",
          opacity: 0.7,
        },
      ],
      { duration: 2200 + Math.random() * 1800, easing: "cubic-bezier(.2,.6,.4,1)" }
    ).onfinish = () => el.remove();
  }
}
