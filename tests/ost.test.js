const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const test = require("node:test");
const vm = require("node:vm");

// static/ost.js runs inside a vm context; compare arrays by join(), never
// with deepEqual (the arrays belong to the other realm).
function loadTest() {
  const context = vm.createContext({
    console, Math, Set, Map, Object, Promise, String, RegExp, Number, Array, setTimeout, clearTimeout,
    window: { scrollTo() {} },
    document: { querySelector: () => null, activeElement: null, getElementById: () => null },
  });
  vm.runInContext(fs.readFileSync(path.join(__dirname, "..", "static", "ost.js"), "utf8"), context);
  const t = context.ostTest();
  t.terms = { perimeter: "all the way around", "line plot": "a number line with an X", rectangle: "4 sides" };
  return { t, context };
}

test("glossary matching is whole-word, case-insensitive, and longest first", () => {
  const { t } = loadTest();
  const parts = t.promptParts("A Rectangle's perimeter. This line plot shows lines.");
  assert.equal(parts.filter((p) => p.term).map((p) => p.term).join("|"), "rectangle|perimeter|line plot");
  assert.equal(parts.map((p) => p.text).join(""), "A Rectangle's perimeter. This line plot shows lines.");
  assert.equal(t.promptParts("").length, 1);
});

test("the pad builds numbers, decimals and fractions, and refuses nonsense", () => {
  const { t } = loadTest();
  t.items = [{ id: "q1", type: "number" }]; t.index = 0; t.view = "question";
  const saved = [];
  t.queueSave = (id) => saved.push(id);
  t.padPress("/"); assert.equal(t.numberText(), "", "a fraction bar cannot start an answer");
  t.padPress("3"); t.padPress("/"); t.padPress("/"); t.padPress("4"); assert.equal(t.numberText(), "3/4");
  t.padPress("."); assert.equal(t.numberText(), "3/4", "no decimal inside a fraction");
  t.padDelete(); t.padDelete(); t.padPress("."); t.padPress("5"); assert.equal(t.numberText(), "3.5");
  t.padPress("."); assert.equal(t.numberText(), "3.5", "one decimal point");
  t.padPress("/"); assert.equal(t.numberText(), "3.5", "no fraction bar after a decimal point");
  assert.ok(saved.length > 0, "every change queues a save");
  const ev = (key, extra = {}) => Object.assign({ key, preventDefault() { this.prevented = true; } }, extra);
  for (const mod of ["metaKey", "ctrlKey", "altKey"]) { t.padKey(ev("1", { [mod]: true })); }
  assert.equal(t.numberText(), "3.5", "browser shortcuts are not pad keys");
  assert.equal(t.choiceTerms("a rectangle or a Rectangle, then a line plot").join(","), "rectangle,line plot");
  const digit = ev("7"); t.padKey(digit);
  assert.equal(t.numberText(), "3.57"); assert.equal(digit.prevented, true);
});

test("figures escape labels and describe themselves to read-aloud", () => {
  const { t, context } = loadTest();
  const svg = t.figureSVG({ kind: "rect", a: 8, b: 3, label_a: "<img src=x onerror=alert(1)>", label_b: "3 ft" });
  assert.ok(svg.includes("&lt;img"), "labels are escaped");
  assert.ok(!svg.includes("<img"));
  assert.ok(svg.includes("currentColor") && !svg.includes("#16302b"), "figure text follows the theme");
  assert.ok(t.figureSVG({ kind: "numberline", a: 4, b: 1 }).includes("<circle"));
  assert.equal(t.figureSVG({ kind: "nope", a: 1, b: 1 }), "");
  const bars = t.figureSVG({ kind: "bars", a: 5, names: ["Dogs", "<b>Cats</b>", "Fish"], values: [10, 0, 25] });
  assert.ok(bars.includes("&lt;b&gt;Cats") && !bars.includes("<b>"), "bar names are escaped");
  assert.equal((bars.match(/<rect /g) || []).length, 3, "one bar per category, even a zero bar");
  assert.ok(bars.includes(">25<"), "the axis reaches the tallest bar");
  const plot = t.figureSVG({ kind: "lineplot", a: 8, values: [0, 0, 2, 0, 3, 0, 0, 1, 0] });
  assert.equal((plot.match(/>X</g) || []).length, 6, "one X per pencil");
  assert.ok(plot.includes(">1½<"), "quarter-inch labels are drawn");
  const spoken = [];
  context.window.speechSynthesis = { cancel() {}, speak(u) { spoken.push(u.text); } };
  context.SpeechSynthesisUtterance = function (text) { this.text = text; };
  t.canSpeak = true;
  t.items = [{ id: "q1", type: "choice", prompt: "What fraction is the dot on?", choices: ["1/4", "2/4"], figure: { kind: "numberline", a: 4, b: 1, alt: "A picture of a number line. The dot is 1 part from 0." } }];
  t.index = 0; t.view = "question";
  t.readAloud();
  assert.ok(spoken[0].includes("The dot is 1 part from 0"), "the picture is read out");
  assert.ok(spoken[0].includes("1 over 4"), "fractions are read as over");
});
