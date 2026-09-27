// Kata renderer. Draws a creature as an SVG string from an authored design
// (static/kata-designs.js) plus its discovery state, so the same kata looks
// the same on every device and no artwork ships with the app.
//
// Every kata is hand-specified: a body type from twelve silhouettes, its own
// four-color scheme, a cute or cool vibe, a pattern, and signature features
// tied to its name. Body parts and features are the color-by-number regions:
// they fill in a fixed order (big parts first) as the child earns fills, and
// pattern marks pad the count to the creature's region total. Unfilled parts
// render as white line art with a number badge; undiscovered kata are a
// silhouette with a "?". The belt shows the grade like a real dojo belt and
// turns black when the kata evolves.
(function () {
  "use strict";

  const OUT = "#1f2937";
  const PAPER = "#ffffff";
  const BADGE = "#e2e8f0";
  const CHEEK = "#fda4af";
  const BELTS = { 1: "#f8fafc", 2: "#facc15", 3: "#fb923c", 4: "#22c55e", 5: "#3b82f6" };

  const f1 = (n) => Number(n).toFixed(1);
  function esc(s) {
    return String(s).replace(/[&<>"']/g, (ch) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[ch]));
  }
  function rng(seed) {
    let a = seed >>> 0;
    return function () {
      a = (a + 0x6d2b79f5) >>> 0;
      let t = a;
      t = Math.imul(t ^ (t >>> 15), t | 1);
      t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
      return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
    };
  }
  const HEX = /^#[0-9a-f]{6}$/i;

  // Color math for cel shading: every part gets a shade, a highlight, and
  // an outline derived from its own fill, the way painted monster art does.
  function hexToRgb(h) { return [parseInt(h.slice(1, 3), 16), parseInt(h.slice(3, 5), 16), parseInt(h.slice(5, 7), 16)]; }
  function rgbToHex([r, g, b]) { return "#" + [r, g, b].map((v) => Math.max(0, Math.min(255, Math.round(v))).toString(16).padStart(2, "0")).join(""); }
  function mix(a, b, t) { const A = hexToRgb(a), B = hexToRgb(b); return rgbToHex([A[0] + (B[0] - A[0]) * t, A[1] + (B[1] - A[1]) * t, A[2] + (B[2] - A[2]) * t]); }
  const shadeOf = (hex) => mix(hex, "#1e1b4b", 0.42);   // cool shadow
  const outlineOf = (hex) => mix(hex, "#111827", 0.62); // colored ink

  // hsl -> hex, so designs can be authored as hues and stay harmonious.
  function hsl(h, s, l) {
    h = ((h % 360) + 360) % 360; s /= 100; l /= 100;
    const k = (n) => (n + h / 30) % 12;
    const a = s * Math.min(l, 1 - l);
    const f = (n) => l - a * Math.max(-1, Math.min(k(n) - 3, Math.min(9 - k(n), 1)));
    const to = (v) => Math.round(v * 255).toString(16).padStart(2, "0");
    return `#${to(f(0))}${to(f(8))}${to(f(4))}`;
  }
  // A scheme from a design: base hue, an accent hue, and a lightness mood.
  function scheme(d) {
    const h = Number(d.hue) || 0, a = d.accent === undefined ? h + 150 : Number(d.accent) || 0;
    const dark = d.mood === "dark";
    return {
      base: hsl(h, dark ? 52 : 70, dark ? 40 : 58),
      light: hsl(h, dark ? 45 : 65, dark ? 70 : 84),
      accent: hsl(a, 85, 58),
      dark: hsl(h, dark ? 55 : 65, dark ? 24 : 34),
    };
  }

  // ---- shape helpers (path "d" strings) ----
  const ellipse = (cx, cy, rx, ry) => `M${f1(cx - rx)},${f1(cy)} a${f1(rx)},${f1(ry)} 0 1 0 ${f1(rx * 2)},0 a${f1(rx)},${f1(ry)} 0 1 0 ${f1(-rx * 2)},0 Z`;
  const poly = (pts) => "M" + pts.map(([x, y]) => `${f1(x)},${f1(y)}`).join(" L") + " Z";
  function egg(cx, cy, rx, ry, top) {
    const t = top || 0.75;
    return `M${f1(cx)},${f1(cy - ry)} C${f1(cx + rx * t)},${f1(cy - ry)} ${f1(cx + rx)},${f1(cy - ry * 0.2)} ${f1(cx + rx)},${f1(cy + ry * 0.15)} ` +
      `C${f1(cx + rx)},${f1(cy + ry * 0.7)} ${f1(cx + rx * 0.6)},${f1(cy + ry)} ${f1(cx)},${f1(cy + ry)} ` +
      `C${f1(cx - rx * 0.6)},${f1(cy + ry)} ${f1(cx - rx)},${f1(cy + ry * 0.7)} ${f1(cx - rx)},${f1(cy + ry * 0.15)} ` +
      `C${f1(cx - rx)},${f1(cy - ry * 0.2)} ${f1(cx - rx * t)},${f1(cy - ry)} ${f1(cx)},${f1(cy - ry)} Z`;
  }
  const teardrop = (cx, cy, rx, ry, dir) =>
    `M${f1(cx - dir * rx * 0.2)},${f1(cy)} Q${f1(cx)},${f1(cy - ry)} ${f1(cx + dir * rx)},${f1(cy - ry * 0.4)} Q${f1(cx + dir * rx * 0.6)},${f1(cy + ry * 0.5)} ${f1(cx - dir * rx * 0.2)},${f1(cy)} Z`;
  const tailCurl = (x, y, dir, len, up) => {
    const ex = x + dir * len, ey = y - up;
    return `M${f1(x)},${f1(y - 6)} Q${f1(x + dir * len * 0.7)},${f1(y - 4)} ${f1(ex)},${f1(ey)} Q${f1(ex + dir * 6)},${f1(ey + 10)} ${f1(ex - dir * 4)},${f1(ey + 12)} Q${f1(x + dir * len * 0.6)},${f1(y + 8)} ${f1(x)},${f1(y + 6)} Z`;
  };
  const batWing = (x, y, d, s) =>
    `M${f1(x)},${f1(y)} L${f1(x + d * s * 0.9)},${f1(y - s * 0.9)} L${f1(x + d * s * 1.1)},${f1(y - s * 0.35)} L${f1(x + d * s * 1.25)},${f1(y + s * 0.15)} L${f1(x + d * s * 0.85)},${f1(y + s * 0.05)} L${f1(x + d * s * 0.55)},${f1(y + s * 0.45)} L${f1(x + d * s * 0.25)},${f1(y + s * 0.2)} Z`;
  const featherWing = (x, y, d, s) =>
    `M${f1(x)},${f1(y)} Q${f1(x + d * s * 0.5)},${f1(y - s * 0.8)} ${f1(x + d * s * 1.2)},${f1(y - s * 0.55)} Q${f1(x + d * s * 0.9)},${f1(y - s * 0.3)} ${f1(x + d * s * 1.25)},${f1(y - s * 0.15)} Q${f1(x + d * s * 0.9)},${f1(y)} ${f1(x + d * s * 1.1)},${f1(y + s * 0.25)} Q${f1(x + d * s * 0.5)},${f1(y + s * 0.3)} ${f1(x)},${f1(y + s * 0.1)} Z`;

  // Ink details: thin strokes drawn over the parts that suggest volume
  // (toe lines, inner ears, chest fluff). Never regions.
  const toes = (x, y, w) => `<path d="M${f1(x - w * 0.3)},${f1(y + 2)} l0,-5 M${f1(x)},${f1(y + 2)} l0,-6 M${f1(x + w * 0.3)},${f1(y + 2)} l0,-5" stroke="${OUT}" stroke-width="1.6" stroke-linecap="round" opacity="0.7"/>`;
  const innerEar = (x, y, rx, ry, color) => `<ellipse cx="${f1(x)}" cy="${f1(y)}" rx="${f1(rx)}" ry="${f1(ry)}" fill="${color}" opacity="0.85"/>`;
  const fluff = (x, y) => `<path d="M${f1(x - 10)},${f1(y)} q3,-6 6,0 q3,-6 6,0 q3,-6 6,0" fill="none" stroke="${OUT}" stroke-width="1.6" stroke-linecap="round" opacity="0.6"/>`;

  // A part: { name, d, color, cx, cy, big?, back?, small? }
  const P = (name, d, color, cx, cy, extra) => Object.assign({ name, d, color, cx, cy }, extra || {});

  // ---- body types: return { parts, face, area, float? } ----
  const BODIES = {
    blob(C, v) {
      const p = [];
      p.push(P("body", egg(70, 96, 34, 32, 0.9), C.base, 70, 108, { big: 2 }));
      p.push(P("head", ellipse(70, 54, v.cute ? 33 : 30, v.cute ? 30 : 27), C.base, 70, 42, { big: 1 }));
      p.push(P("belly", ellipse(70, 104, 20, 18), C.light, 70, 104, { big: 3 }));
      p.push(P("left arm", "M40,84 Q22,96 30,112 Q36,106 44,100 Z", C.base, 30, 102));
      p.push(P("right arm", "M100,84 Q118,96 110,112 Q104,106 96,100 Z", C.base, 110, 102));
      p.push(P("left foot", ellipse(52, 128, 13, 8), C.dark, 52, 128));
      p.push(P("right foot", ellipse(88, 128, 13, 8), C.dark, 88, 128));
      return { parts: p, face: { cx: 70, cy: 56, spread: 12, eye: v.cute ? 8.5 : 7.5 }, area: { cx: 70, cy: 104, rx: 16, ry: 13 }, details: toes(52, 128, 14) + toes(88, 128, 14) };
    },
    beast(C, v) {
      const p = [];
      p.push(P("tail", tailCurl(102, 98, 1, 26, 34), C.base, 122, 78, { back: true }));
      p.push(P("left back leg", ellipse(44, 118, 10, 14), C.dark, 44, 118, { back: true }));
      p.push(P("right back leg", ellipse(96, 118, 10, 14), C.dark, 96, 118, { back: true }));
      p.push(P("body", ellipse(70, 98, 38, 27), C.base, 70, 112, { big: 2 }));
      p.push(P("belly", ellipse(70, 106, 24, 15), C.light, 70, 106, { big: 3 }));
      p.push(P("left leg", ellipse(54, 124, 11, 12), C.base, 54, 126));
      p.push(P("right leg", ellipse(86, 124, 11, 12), C.base, 86, 126));
      p.push(P("head", ellipse(70, 56, 30, 26), C.base, 70, 44, { big: 1 }));
      p.push(P("muzzle", ellipse(70, 66, 13, 9), C.light, 70, 68));
      return { parts: p, face: { cx: 70, cy: 54, spread: 13, eye: 7.5, muzzle: true }, area: { cx: 70, cy: 106, rx: 19, ry: 11 }, details: toes(54, 130, 12) + toes(86, 130, 12) + fluff(70, 82) };
    },
    bird(C, v) {
      const p = [];
      p.push(P("left wing", featherWing(42, 86, -1, 30), C.dark, 24, 84, { back: true }));
      p.push(P("right wing", featherWing(98, 86, 1, 30), C.dark, 116, 84, { back: true }));
      p.push(P("tail feathers", poly([[62, 122], [78, 122], [92, 140], [70, 132], [48, 140]]), C.accent, 70, 133, { back: true }));
      p.push(P("body", egg(70, 86, 34, 40, 0.7), C.base, 70, 100, { big: 1 }));
      p.push(P("belly", egg(70, 98, 20, 24, 0.8), C.light, 70, 100, { big: 3 }));
      p.push(P("left foot", poly([[56, 126], [64, 126], [62, 136], [50, 136]]), C.accent, 57, 131));
      p.push(P("right foot", poly([[76, 126], [84, 126], [90, 136], [78, 136]]), C.accent, 83, 131));
      p.push(P("beak", poly([[62, 60], [78, 60], [70, 72]]), C.accent, 70, 63, { small: true }));
      return { parts: p, face: { cx: 70, cy: 52, spread: 11, eye: 7 }, area: { cx: 70, cy: 98, rx: 15, ry: 17 } };
    },
    bug(C, v) {
      const p = [];
      for (let i = 0; i < 3; i++) {
        const y = 84 + i * 14;
        p.push(P(`left leg ${i + 1}`, poly([[52, y], [30, y + 10], [28, y + 16], [34, y + 14], [54, y + 6]]), C.dark, 38, y + 10, { back: true, small: true }));
        p.push(P(`right leg ${i + 1}`, poly([[88, y], [110, y + 10], [112, y + 16], [106, y + 14], [86, y + 6]]), C.dark, 102, y + 10, { back: true, small: true }));
      }
      p.push(P("abdomen", ellipse(70, 112, 28, 24), C.base, 70, 120, { big: 2 }));
      p.push(P("left shell", teardrop(58, 108, 22, 18, -1), C.light, 52, 108, { big: 3 }));
      p.push(P("right shell", teardrop(82, 108, 22, 18, 1), C.light, 88, 108, { big: 3 }));
      p.push(P("thorax", ellipse(70, 82, 20, 15), C.base, 70, 86));
      p.push(P("head", ellipse(70, 52, 26, 24), C.base, 70, 42, { big: 1 }));
      return { parts: p, face: { cx: 70, cy: 54, spread: 11, eye: 7.5 }, area: { cx: 70, cy: 118, rx: 14, ry: 10 } };
    },
    fish(C, v) {
      const p = [];
      p.push(P("tail fin", poly([[104, 88], [130, 66], [126, 88], [130, 110]]), C.accent, 120, 88, { back: true }));
      p.push(P("belly fin", poly([[60, 110], [72, 128], [82, 108]]), C.accent, 71, 114, { back: true, small: true }));
      p.push(P("body", ellipse(64, 88, 44, 30), C.base, 84, 78, { big: 1 }));
      p.push(P("belly", ellipse(70, 100, 30, 14), C.light, 76, 100, { big: 3 }));
      p.push(P("side fin", teardrop(70, 96, 22, 12, 1), C.dark, 80, 94));
      p.push(P("lip", ellipse(24, 92, 8, 6), C.light, 24, 92, { small: true }));
      return { parts: p, face: { cx: 42, cy: 78, spread: 10, eye: 7.5, fish: true }, area: { cx: 86, cy: 80, rx: 18, ry: 14 } };
    },
    dragon(C, v) {
      const p = [];
      p.push(P("left wing", batWing(48, 84, -1, 34), C.dark, 20, 68, { back: true }));
      p.push(P("right wing", batWing(92, 84, 1, 34), C.dark, 120, 68, { back: true }));
      p.push(P("tail", tailCurl(100, 104, 1, 24, 22), C.base, 118, 92, { back: true }));
      p.push(P("body", egg(70, 98, 34, 34, 0.85), C.base, 70, 112, { big: 2 }));
      p.push(P("belly plates", egg(70, 106, 20, 22, 0.9), C.light, 70, 106, { big: 3 }));
      p.push(P("left arm", ellipse(38, 96, 8, 12), C.base, 38, 96, { small: true }));
      p.push(P("right arm", ellipse(102, 96, 8, 12), C.base, 102, 96, { small: true }));
      p.push(P("left foot", ellipse(52, 130, 13, 8), C.dark, 52, 130));
      p.push(P("right foot", ellipse(88, 130, 13, 8), C.dark, 88, 130));
      p.push(P("head", ellipse(70, 54, 30, 26), C.base, 70, 44, { big: 1 }));
      p.push(P("snout", ellipse(70, 66, 14, 9), C.light, 70, 68, { small: true }));
      return { parts: p, face: { cx: 70, cy: 52, spread: 12, eye: 7, brow: !v.cute }, area: { cx: 70, cy: 106, rx: 15, ry: 16 }, details: toes(52, 132, 14) + toes(88, 132, 14) };
    },
    ghost(C, v) {
      const p = [];
      p.push(P("body", "M38,92 Q38,50 70,44 Q102,50 102,92 L102,124 Q94,116 86,124 Q78,132 70,124 Q62,132 54,124 Q46,116 38,124 Z", C.base, 70, 108, { big: 1 }));
      p.push(P("left arm", ellipse(34, 92, 10, 7), C.base, 32, 92, { small: true }));
      p.push(P("right arm", ellipse(106, 92, 10, 7), C.base, 108, 92, { small: true }));
      p.push(P("glow", ellipse(70, 98, 18, 14), C.light, 70, 98, { big: 3 }));
      return { parts: p, face: { cx: 70, cy: 70, spread: 12, eye: 8.5 }, area: { cx: 70, cy: 96, rx: 18, ry: 8 }, float: true };
    },
    golem(C, v) {
      const p = [];
      p.push(P("body", poly([[36, 78], [104, 78], [110, 124], [30, 124]]), C.base, 70, 114, { big: 2 }));
      p.push(P("head", poly([[42, 34], [98, 34], [102, 72], [38, 72]]), C.base, 70, 44, { big: 1 }));
      p.push(P("chest plate", poly([[52, 90], [88, 90], [84, 116], [56, 116]]), C.light, 70, 100, { big: 3 }));
      p.push(P("left arm", poly([[30, 82], [18, 88], [20, 118], [34, 114]]), C.dark, 26, 100));
      p.push(P("right arm", poly([[110, 82], [122, 88], [120, 118], [106, 114]]), C.dark, 114, 100));
      p.push(P("left foot", poly([[36, 124], [60, 124], [58, 138], [34, 138]]), C.dark, 47, 131));
      p.push(P("right foot", poly([[80, 124], [104, 124], [106, 138], [82, 138]]), C.dark, 93, 131));
      return { parts: p, face: { cx: 70, cy: 52, spread: 14, eye: 7 }, area: { cx: 70, cy: 103, rx: 13, ry: 10 } };
    },
    sprout(C, v) {
      const p = [];
      p.push(P("pot", poly([[44, 100], [96, 100], [90, 136], [50, 136]]), C.dark, 70, 122, { big: 2 }));
      p.push(P("rim", poly([[40, 96], [100, 96], [100, 106], [40, 106]]), C.accent, 70, 101, { small: true }));
      p.push(P("head", ellipse(70, 66, 28, 26), C.base, 70, 56, { big: 1 }));
      p.push(P("left leaf", teardrop(50, 42, 26, 14, -1), C.light, 34, 36, { back: true }));
      p.push(P("right leaf", teardrop(90, 42, 26, 14, 1), C.light, 106, 36, { back: true }));
      p.push(P("stem", poly([[66, 46], [74, 46], [72, 24], [68, 24]]), C.dark, 70, 34, { back: true, small: true }));
      p.push(P("bud", ellipse(70, 22, 8, 8), C.accent, 70, 22, { back: true, small: true }));
      return { parts: p, face: { cx: 70, cy: 66, spread: 11, eye: 8 }, area: { cx: 70, cy: 120, rx: 18, ry: 10 } };
    },
    serpent(C, v) {
      const p = [];
      p.push(P("coil", "M36,116 Q30,138 60,136 L114,136 Q134,134 130,114 Q128,100 106,100 L58,100 Q40,100 42,112 Q44,122 58,122 L100,122 Q110,122 108,114 Q106,110 98,110 L64,110 Q58,110 58,114 Q58,116 64,116 L104,116 Q116,116 118,122 Q118,130 106,130 L58,130 Q38,130 36,116 Z", C.base, 84, 130, { big: 2 }));
      p.push(P("neck", poly([[54, 78], [86, 78], [98, 104], [42, 104]]), C.base, 70, 92, { big: 3 }));
      p.push(P("belly scales", poly([[60, 82], [80, 82], [88, 102], [52, 102]]), C.light, 70, 94));
      p.push(P("head", ellipse(70, 56, 32, 24), C.base, 70, 44, { big: 1 }));
      p.push(P("tail tip", poly([[128, 112], [140, 96], [136, 124]]), C.accent, 134, 110, { back: true, small: true }));
      return { parts: p, face: { cx: 70, cy: 56, spread: 13, eye: 7, brow: !v.cute }, area: { cx: 84, cy: 130, rx: 28, ry: 4 } };
    },
    octo(C, v) {
      const p = [];
      for (let i = 0; i < 6; i++) {
        const x = 34 + i * 14.4, d = i % 2 ? 1 : -1;
        p.push(P(`tentacle ${i + 1}`, `M${f1(x - 6)},100 Q${f1(x + d * 8)},124 ${f1(x)},138 Q${f1(x + 6)},130 ${f1(x + 6)},100 Z`, i % 2 ? C.dark : C.base, x, 124, { back: true, small: true }));
      }
      p.push(P("head", ellipse(70, 66, 40, 38), C.base, 70, 44, { big: 1 }));
      p.push(P("brow ridge", ellipse(70, 40, 24, 10), C.light, 70, 34, { small: true }));
      return { parts: p, face: { cx: 70, cy: 68, spread: 15, eye: 9 }, area: { cx: 70, cy: 90, rx: 24, ry: 8 } };
    },
    turtle(C, v) {
      const p = [];
      p.push(P("shell", ellipse(72, 98, 40, 30), C.dark, 84, 88, { big: 2 }));
      p.push(P("shell rim", ellipse(72, 112, 42, 12), C.accent, 72, 122, { big: 3 }));
      p.push(P("left leg", ellipse(40, 118, 10, 12), C.base, 38, 124));
      p.push(P("right leg", ellipse(104, 118, 10, 12), C.base, 106, 124));
      p.push(P("head", ellipse(58, 56, 26, 22), C.base, 56, 44, { big: 1 }));
      p.push(P("tail", poly([[110, 104], [128, 112], [112, 116]]), C.base, 122, 110, { back: true, small: true }));
      return { parts: p, face: { cx: 58, cy: 56, spread: 11, eye: 7.5 }, area: { cx: 74, cy: 96, rx: 28, ry: 18 } };
    },
    wisp(C, v) {
      const p = [];
      p.push(P("outer flame", "M70,16 Q112,60 100,104 Q92,130 70,132 Q48,130 40,104 Q28,60 70,16 Z", C.base, 70, 112, { big: 1 }));
      p.push(P("inner flame", "M70,44 Q92,72 86,100 Q82,118 70,120 Q58,118 54,100 Q48,72 70,44 Z", C.light, 70, 106, { big: 3 }));
      p.push(P("left spark", ellipse(30, 78, 6, 6), C.accent, 30, 78, { small: true }));
      p.push(P("right spark", ellipse(112, 62, 5, 5), C.accent, 112, 62, { small: true }));
      return { parts: p, face: { cx: 70, cy: 76, spread: 11, eye: 8 }, area: { cx: 70, cy: 100, rx: 8, ry: 6 }, float: true };
    },
  };

  // ---- signature features ----
  const FEATURES = {
    horns: (C) => [P("left horn", poly([[50, 36], [42, 10], [62, 30]]), C.accent, 50, 24, { back: true }), P("right horn", poly([[90, 36], [98, 10], [78, 30]]), C.accent, 90, 24, { back: true })],
    curvedHorns: (C) => [P("left horn", "M52,36 Q30,30 36,10 Q40,24 60,30 Z", C.accent, 42, 24, { back: true }), P("right horn", "M88,36 Q110,30 104,10 Q100,24 80,30 Z", C.accent, 98, 24, { back: true })],
    unicorn: (C) => [P("horn", poly([[62, 34], [70, 4], [78, 34]]), C.accent, 70, 20, { back: true })],
    antlers: (C) => [P("left antler", poly([[54, 34], [44, 16], [46, 6], [50, 14], [56, 10], [54, 18], [60, 30]]), C.dark, 50, 20, { back: true, small: true }), P("right antler", poly([[86, 34], [96, 16], [94, 6], [90, 14], [84, 10], [86, 18], [80, 30]]), C.dark, 90, 20, { back: true, small: true })],
    pointEars: (C) => [P("left ear", poly([[46, 40], [40, 14], [60, 32]]), C.base, 48, 26, { back: true }), P("right ear", poly([[94, 40], [100, 14], [80, 32]]), C.base, 92, 26, { back: true })],
    roundEars: (C) => [P("left ear", ellipse(46, 30, 10, 10), C.base, 46, 30, { back: true }), P("right ear", ellipse(94, 30, 10, 10), C.base, 94, 30, { back: true })],
    longEars: (C) => [P("left ear", ellipse(50, 20, 7, 18), C.base, 50, 14, { back: true }), P("right ear", ellipse(90, 20, 7, 18), C.base, 90, 14, { back: true })],
    floppyEars: (C) => [P("left ear", ellipse(40, 54, 8, 18), C.dark, 40, 56, { back: true }), P("right ear", ellipse(100, 54, 8, 18), C.dark, 100, 56, { back: true })],
    antennae: (C) => [P("left antenna", poly([[58, 30], [50, 10], [56, 8], [63, 28]]), C.dark, 54, 18, { back: true, small: true }), P("right antenna", poly([[82, 30], [90, 10], [84, 8], [77, 28]]), C.dark, 86, 18, { back: true, small: true }), P("left feeler tip", ellipse(52, 9, 5, 5), C.accent, 52, 9, { back: true, small: true }), P("right feeler tip", ellipse(88, 9, 5, 5), C.accent, 88, 9, { back: true, small: true })],
    mohawk: (C) => [P("mohawk", poly([[58, 34], [62, 8], [68, 24], [72, 4], [78, 24], [84, 10], [86, 34]]), C.accent, 72, 20, { back: true })],
    flameCrest: (C) => [P("flame crest", "M60,34 Q56,14 68,10 Q66,20 74,14 Q72,24 84,16 Q86,30 80,34 Z", C.accent, 72, 22, { back: true })],
    leafCrest: (C) => [P("leaf crest", teardrop(78, 22, 26, 12, 1), C.light, 92, 16, { back: true }), P("crest stem", poly([[66, 36], [72, 36], [70, 22], [68, 22]]), C.dark, 70, 30, { back: true, small: true })],
    crestFeathers: (C) => [0, 1, 2].map((i) => P(`crest ${i + 1}`, poly([[58 + i * 12 - 5, 50], [58 + i * 12 + 5, 50], [58 + i * 12 + (i - 1) * 4, 28]]), C.accent, 58 + i * 12, 42, { back: true, small: true })),
    mane: (C) => [P("mane", ellipse(70, 58, 40, 36), C.accent, 40, 40, { back: true, big: 4 })],
    ruff: (C) => [P("ruff", ellipse(70, 84, 34, 12), C.light, 40, 84, { back: true, small: true })],
    scarf: (C) => [P("scarf", "M40,80 Q70,92 100,80 L102,90 Q70,102 38,90 Z", C.accent, 70, 88, { small: true }), P("scarf tail", poly([[96, 86], [116, 96], [104, 108], [94, 94]]), C.accent, 108, 98, { small: true })],
    headband: (C) => [P("headband", "M40,44 Q70,52 100,44 L100,52 Q70,60 40,52 Z", C.accent, 70, 50, { small: true }), P("band tails", poly([[98, 46], [118, 40], [116, 52], [100, 54]]), C.accent, 110, 46, { small: true })],
    backSpikes: (C) => [0, 1, 2, 3].map((i) => P(`spike ${i + 1}`, poly([[42 + i * 18, 76], [50 + i * 18, 60], [58 + i * 18, 76]]), C.accent, 50 + i * 18, 70, { back: true, small: true })),
    tusks: (C) => [P("left tusk", poly([[56, 70], [50, 84], [60, 72]]), "#ffffff", 54, 76, { small: true }), P("right tusk", poly([[84, 70], [90, 84], [80, 72]]), "#ffffff", 86, 76, { small: true })],
    halo: (C) => [P("halo", "M40,18 a30,8 0 1 0 60,0 a30,8 0 1 0 -60,0 Z M48,18 a22,4 0 1 1 44,0 a22,4 0 1 1 -44,0 Z", C.accent, 70, 8, { back: true, small: true })],
    crystals: (C) => [P("left crystal", poly([[22, 60], [30, 44], [38, 62], [30, 72]]), C.light, 30, 58, { back: true, small: true }), P("right crystal", poly([[102, 50], [110, 34], [118, 52], [110, 62]]), C.light, 110, 48, { back: true, small: true }), P("top crystal", poly([[64, 20], [70, 4], [76, 20], [70, 30]]), C.light, 70, 16, { back: true, small: true })],
    lantern: (C) => [P("lantern arm", poly([[100, 92], [126, 70], [130, 74], [104, 96]]), C.dark, 116, 82, { back: true, small: true }), P("lantern", ellipse(130, 62, 9, 11), C.accent, 130, 62, { back: true }), P("lantern glow", ellipse(130, 62, 4, 6), "#ffffff", 130, 62, { back: true, small: true })],
    star: (C) => [P("star patch", "M70,74 l3,7 l7,1 l-5,5 l1,7 l-6,-3 l-6,3 l1,-7 l-5,-5 l7,-1 Z", C.accent, 70, 82, { small: true })],
    bubbles: (C) => [P("bubble 1", ellipse(20, 60, 5, 5), C.light, 20, 60, { back: true, small: true }), P("bubble 2", ellipse(14, 44, 3.5, 3.5), C.light, 14, 44, { back: true, small: true }), P("bubble 3", ellipse(26, 34, 2.5, 2.5), C.light, 26, 34, { back: true, small: true })],
    tailSpade: (C) => [P("tail spade", poly([[118, 78], [132, 70], [128, 86]]), C.accent, 126, 78, { back: true, small: true })],
    tailTip: (C) => [P("tail tip", ellipse(124, 66, 7, 7), C.accent, 124, 66, { back: true, small: true })],
    cape: (C) => [P("cape", "M40,78 Q70,70 100,78 L112,134 Q70,124 28,134 Z", C.dark, 70, 122, { back: true, big: 4 })],
    shellPlates: (C) => [P("plate 1", ellipse(72, 92, 12, 9), C.light, 72, 92, { small: true }), P("plate 2", ellipse(52, 102, 8, 7), C.light, 52, 102, { small: true }), P("plate 3", ellipse(92, 102, 8, 7), C.light, 92, 102, { small: true })],
    twinHead: (C) => [P("second head", ellipse(106, 50, 22, 20), C.base, 106, 38, { back: true, big: 4 })],
    wingsBat: (C) => [P("left wing", batWing(48, 84, -1, 32), C.dark, 22, 70, { back: true }), P("right wing", batWing(92, 84, 1, 32), C.dark, 118, 70, { back: true })],
    wingsFeather: (C) => [P("left wing", featherWing(44, 84, -1, 30), C.light, 24, 80, { back: true }), P("right wing", featherWing(96, 84, 1, 30), C.light, 116, 80, { back: true })],
    fins: (C) => [P("left fin", teardrop(38, 92, 20, 12, -1), C.accent, 26, 90, { back: true, small: true }), P("right fin", teardrop(102, 92, 20, 12, 1), C.accent, 114, 90, { back: true, small: true })],
    dorsalSail: (C) => [P("dorsal sail", poly([[50, 64], [58, 40], [66, 58], [76, 34], [84, 58], [94, 44], [98, 66]]), C.accent, 74, 54, { back: true })],
    stubbyTail: (C) => [P("tail", ellipse(110, 108, 12, 9), C.base, 112, 108, { back: true, small: true })],
    bigTail: (C) => [P("tail", "M104,100 Q140,90 134,60 Q128,84 100,92 Z", C.light, 126, 82, { back: true })],
    crown: (C) => [P("crown", poly([[54, 30], [58, 12], [64, 24], [70, 8], [76, 24], [82, 12], [86, 30]]), C.accent, 70, 22, { back: true })],
    tuft: (C) => [P("tuft", poly([[62, 32], [66, 12], [72, 26], [78, 14], [80, 32]]), C.light, 71, 22, { back: true, small: true })],
    plume: (C) => [P("plume", "M70,34 Q56,6 78,4 Q70,18 84,14 Q80,30 70,34 Z", C.accent, 74, 16, { back: true })],
    // face-only features (no regions): drawn by face()
    thirdEye: () => [], glasses: () => [], mask: () => [],
  };

  // ---- pattern marks pad the region count ----
  function pattern(r, kind, area, count, C) {
    const list = [];
    if (kind === "stripes") {
      for (let i = 0; i < count; i++) {
        const y = area.cy - area.ry + ((i + 0.5) / count) * area.ry * 2;
        const w = Math.max(10, area.rx * 1.6 * Math.sqrt(Math.max(0, 1 - Math.pow((y - area.cy) / area.ry, 2))));
        list.push(P(`stripe ${i + 1}`, poly([[area.cx - w / 2, y - 3], [area.cx + w / 2, y - 3], [area.cx + w / 2 - 3, y + 3], [area.cx - w / 2 + 3, y + 3]]), C.accent, area.cx, y, { small: true }));
      }
      return list;
    }
    if (kind === "checks") {
      for (let row = 0; row < 3 && list.length < count; row++) for (let col = 0; col < 3 && list.length < count; col++) {
        if ((row + col) % 2) continue;
        const x = area.cx - area.rx * 0.7 + col * area.rx * 0.7, y = area.cy - area.ry * 0.7 + row * area.ry * 0.7;
        list.push(P(`check ${list.length + 1}`, poly([[x - 6, y - 6], [x + 6, y - 6], [x + 6, y + 6], [x - 6, y + 6]]), C.accent, x, y, { small: true }));
      }
      return list;
    }
    if (kind === "patches") {
      const pts = [[-0.5, -0.4], [0.5, -0.3], [0, 0.5], [-0.5, 0.5], [0.55, 0.45], [0, -0.55]];
      for (let i = 0; i < Math.min(count, pts.length); i++) {
        const x = area.cx + pts[i][0] * area.rx, y = area.cy + pts[i][1] * area.ry;
        list.push(P(`patch ${i + 1}`, poly([[x - 7, y - 5], [x + 6, y - 7], [x + 7, y + 5], [x - 6, y + 6]]), i % 2 ? C.accent : C.light, x, y, { small: true }));
      }
      return list;
    }
    if (kind === "split") {
      list.push(P("left half", ellipse(area.cx - area.rx / 2, area.cy, area.rx / 2, area.ry * 0.9), C.accent, area.cx - area.rx / 2, area.cy, { small: true }));
      return list;
    }
    // "none" and "spots" both fall through to mirrored spots when the
    // region floor needs padding; "none" simply asks for fewer.
    // spots: mirrored, spaced pairs
    const placed = [];
    let guard = 0;
    while (list.length < count && guard++ < 400) {
      // Spacing relaxes as attempts pile up so a small body still fills its count.
      const gap = guard < 150 ? 11 : guard < 300 ? 7 : 3;
      const a = r() * Math.PI, d = 0.35 + r() * 0.6;
      const dx = Math.abs(Math.cos(a) * area.rx * d), dy = Math.sin(a * 2) * area.ry * d * 0.9;
      const cands = [[area.cx + dx, area.cy + dy], [area.cx - dx, area.cy + dy]];
      if (dx < 5) cands.pop();
      if (!cands.every(([x, y]) => placed.every(([px, py]) => Math.hypot(px - x, py - y) >= gap))) continue;
      const rad = 3 + r() * 1.4;
      for (const [x, y] of cands) {
        if (list.length >= count) break;
        placed.push([x, y]);
        list.push(P(`spot ${list.length + 1}`, ellipse(x, y, rad, rad * 0.92), list.length % 4 < 2 ? C.accent : C.light, x, y, { small: true }));
      }
    }
    return list;
  }

  // ---- face, drawn on top, never a region ----
  function face(F, C, d, uid) {
    const cute = d.vibe !== "cool";
    const eyes = d.eyes || (cute ? "sparkle" : "sharp");
    const mouth = d.mouth || (cute ? "smile" : "fang");
    let out = "";
    const pts = F.fish ? [[F.cx, F.cy]] : d.features.includes("thirdEye") ? [[F.cx - F.spread, F.cy], [F.cx + F.spread, F.cy], [F.cx, F.cy - F.spread * 1.1]] : [[F.cx - F.spread, F.cy], [F.cx + F.spread, F.cy]];
    for (const [ex, ey] of pts) {
      const e = F.eye * (eyes === "big" ? 1.2 : eyes === "sharp" ? 0.9 : 1);
      if (eyes === "sharp") out += `<path d="M${f1(ex - e)},${f1(ey + e * 0.6)} Q${f1(ex)},${f1(ey - e * 1.3)} ${f1(ex + e)},${f1(ey + e * 0.6)} Z" fill="#fff" stroke="${OUT}" stroke-width="2.4"/>`;
      else out += `<ellipse cx="${f1(ex)}" cy="${f1(ey)}" rx="${f1(e)}" ry="${f1(e * 1.12)}" fill="#fff" stroke="${OUT}" stroke-width="2.4"/>`;
      if (eyes === "sleepy") out += `<path d="M${f1(ex - e)},${f1(ey - e * 0.2)} a${f1(e)},${f1(e)} 0 0 1 ${f1(e * 2)},0" fill="${OUT}"/>`;
      out += `<circle cx="${f1(ex + 0.8)}" cy="${f1(ey + 1.6)}" r="${f1(e * 0.66)}" fill="url(#${uid}-iris)"/>`;
      out += `<ellipse cx="${f1(ex + 1)}" cy="${f1(ey + 2.2)}" rx="${f1(e * 0.34)}" ry="${f1(e * 0.4)}" fill="${OUT}"/>`;
      out += `<circle cx="${f1(ex - 1.8)}" cy="${f1(ey - 1.6)}" r="${f1(e * 0.3)}" fill="#fff"/>`;
      out += `<circle cx="${f1(ex + 2.4)}" cy="${f1(ey + 3.4)}" r="${f1(e * 0.14)}" fill="#fff"/>`;
      // Upper lid shadow gives the eye depth.
      out += `<path d="M${f1(ex - e)},${f1(ey - e * 0.2)} a${f1(e)},${f1(e * 1.12)} 0 0 1 ${f1(e * 2)},0" fill="${OUT}" opacity="0.18"/>`;
      if ((eyes === "sharp" || F.brow) && ex !== F.cx) {
        const dd = ex < F.cx ? -1 : 1;
        out += `<path d="M${f1(ex - dd * e)},${f1(ey - e * 1.3)} L${f1(ex + dd * e * 0.9)},${f1(ey - e * 1.85)}" stroke="${OUT}" stroke-width="2.6" stroke-linecap="round"/>`;
      }
    }
    if (d.features.includes("glasses")) {
      for (const [ex, ey] of pts.slice(0, 2)) out += `<circle cx="${f1(ex)}" cy="${f1(ey)}" r="${f1(F.eye + 3)}" fill="none" stroke="${OUT}" stroke-width="2.6"/>`;
      out += `<path d="M${f1(F.cx - F.spread + F.eye + 3)},${f1(F.cy)} L${f1(F.cx + F.spread - F.eye - 3)},${f1(F.cy)}" stroke="${OUT}" stroke-width="2.6"/>`;
    }
    if (d.features.includes("mask")) {
      out += `<path d="M${f1(F.cx - F.spread - 12)},${f1(F.cy - 4)} L${f1(F.cx + F.spread + 12)},${f1(F.cy - 6)} L${f1(F.cx + F.spread + 8)},${f1(F.cy + 8)} L${f1(F.cx - F.spread - 8)},${f1(F.cy + 10)} Z" fill="${C.accent}" stroke="${OUT}" stroke-width="2.2"/>`;
      for (const [ex, ey] of pts.slice(0, 2)) out += `<ellipse cx="${f1(ex)}" cy="${f1(ey)}" rx="${f1(F.eye * 0.9)}" ry="${f1(F.eye)}" fill="#fff" stroke="${OUT}" stroke-width="2"/><circle cx="${f1(ex + 1)}" cy="${f1(ey + 1)}" r="${f1(F.eye * 0.45)}" fill="${OUT}"/>`;
    }
    if (d.features.includes("pointEars")) out += innerEar(49, 28, 4, 7, C.light) + innerEar(91, 28, 4, 7, C.light);
    if (d.features.includes("roundEars")) out += innerEar(46, 30, 5, 5, C.light) + innerEar(94, 30, 5, 5, C.light);
    if (d.features.includes("longEars")) out += innerEar(50, 20, 3.5, 12, C.light) + innerEar(90, 20, 3.5, 12, C.light);
    if (cute && !F.fish) {
      out += `<ellipse cx="${f1(F.cx - F.spread - 6)}" cy="${f1(F.cy + 8)}" rx="4" ry="2.4" fill="${CHEEK}" opacity="0.8"/>`;
      out += `<ellipse cx="${f1(F.cx + F.spread + 6)}" cy="${f1(F.cy + 8)}" rx="4" ry="2.4" fill="${CHEEK}" opacity="0.8"/>`;
    }
    const my = F.cy + (F.muzzle ? 14 : 12);
    const mx = F.fish ? F.cx - 12 : F.cx;
    if (mouth === "smile") out += `<path d="M${f1(mx - 7)},${f1(my)} Q${f1(mx)},${f1(my + 7)} ${f1(mx + 7)},${f1(my)}" fill="none" stroke="${OUT}" stroke-width="2.4" stroke-linecap="round"/>`;
    else if (mouth === "open") out += `<path d="M${f1(mx - 7)},${f1(my)} Q${f1(mx)},${f1(my + 10)} ${f1(mx + 7)},${f1(my)} Z" fill="${OUT}"/><path d="M${f1(mx - 3)},${f1(my + 3)} Q${f1(mx)},${f1(my + 7)} ${f1(mx + 3)},${f1(my + 3)} Z" fill="${CHEEK}"/>`;
    else if (mouth === "fang") out += `<path d="M${f1(mx - 7)},${f1(my)} Q${f1(mx)},${f1(my + 6)} ${f1(mx + 7)},${f1(my)}" fill="none" stroke="${OUT}" stroke-width="2.4" stroke-linecap="round"/><path d="M${f1(mx + 3)},${f1(my + 1)} L${f1(mx + 5.5)},${f1(my + 6)} L${f1(mx + 7.5)},${f1(my + 0.5)} Z" fill="#fff" stroke="${OUT}" stroke-width="1.2"/>`;
    else if (mouth === "grin") out += `<path d="M${f1(mx - 9)},${f1(my - 1)} Q${f1(mx)},${f1(my + 9)} ${f1(mx + 9)},${f1(my - 1)} Z" fill="#fff" stroke="${OUT}" stroke-width="2.2"/><path d="M${f1(mx - 6)},${f1(my + 1)} L${f1(mx + 6)},${f1(my + 1)}" stroke="${OUT}" stroke-width="1.4"/>`;
    else if (mouth === "cat") out += `<path d="M${f1(mx - 7)},${f1(my)} Q${f1(mx - 3.5)},${f1(my + 5)} ${f1(mx)},${f1(my)} Q${f1(mx + 3.5)},${f1(my + 5)} ${f1(mx + 7)},${f1(my)}" fill="none" stroke="${OUT}" stroke-width="2.4" stroke-linecap="round"/>`;
    else if (mouth === "shh") out += `<ellipse cx="${f1(mx)}" cy="${f1(my + 2)}" rx="3" ry="4" fill="${OUT}"/><path d="M${f1(mx + 4)},${f1(my - 10)} L${f1(mx + 4)},${f1(my + 8)}" stroke="${OUT}" stroke-width="3.2" stroke-linecap="round"/><path d="M${f1(mx + 4)},${f1(my - 10)} L${f1(mx + 4)},${f1(my + 8)}" stroke="${C.light}" stroke-width="1.6" stroke-linecap="round"/>`;
    else if (mouth === "flat") out += `<path d="M${f1(mx - 6)},${f1(my + 1)} L${f1(mx + 6)},${f1(my + 1)}" stroke="${OUT}" stroke-width="2.4" stroke-linecap="round"/>`;
    if (F.muzzle) out += `<ellipse cx="${f1(F.cx)}" cy="${f1(F.cy + 9)}" rx="3.4" ry="2.4" fill="${OUT}"/>`;
    return out;
  }

  /**
   * creature(seed, opts) -> SVG string.
   * opts: { design, grade, regions, fills, evolved, name, label, size, regionColors, silhouette }
   *   design: an authored spec { body, vibe, hue, accent, mood, pattern, features, eyes, mouth }.
   *           Without one, a design is derived from the seed so old callers keep working.
   *   grade: 1-5 sets the belt color. regionColors: per-region hex for free painting (strict #rrggbb only).
   */
  function creature(seed, opts) {
    const o = Object.assign({ regions: 16, fills: 0, evolved: false, name: "", label: "", size: 120, silhouette: false, grade: 1 }, opts || {});
    const r = rng(seed);
    const bodies = Object.keys(BODIES);
    const d = Object.assign({ body: bodies[Math.floor(r() * bodies.length)], vibe: "cute", hue: (seed * 47) % 360, pattern: "spots", features: [] }, o.design || {});
    if (!BODIES[d.body]) d.body = "blob";
    if (!Array.isArray(d.features)) d.features = [];
    const C = scheme(d);
    const v = { cute: d.vibe !== "cool" };
    const built = BODIES[d.body](C, v);
    let parts = built.parts;
    for (const fname of d.features) if (FEATURES[fname]) parts = parts.concat(FEATURES[fname](C));

    const ordered = [...parts].sort((a, b) => (a.big || 9) - (b.big || 9));
    const padCount = Math.max(0, o.regions - ordered.length);
    const pads = pattern(r, d.pattern, built.area, padCount, C);
    const regions = [...ordered, ...pads].slice(0, o.regions);
    regions.forEach((rg, i) => { rg.index = i; });

    const fills = Math.max(0, Math.min(o.regions, o.fills));
    const custom = Array.isArray(o.regionColors) ? o.regionColors : null;
    const fillOf = (rg) => {
      const painted = custom && HEX.test(custom[rg.index] || "") ? custom[rg.index] : "";
      return painted || (rg.index < fills ? rg.color : "");
    };

    let back = "", front = "", badges = "", defs = "";
    const uid = `k${seed}`;
    const draw = (rg) => {
      if (o.silhouette) return `<path data-region="${rg.index}" d="${rg.d}" fill="#64748b" stroke="#334155" stroke-width="${rg.small ? 2.2 : 3}" stroke-linejoin="round"/>`;
      const fill = fillOf(rg);
      const sw = rg.small ? 2 : 2.8;
      if (!fill) {
        badges += `<circle cx="${f1(rg.cx)}" cy="${f1(rg.cy)}" r="5.2" fill="${BADGE}" stroke="${OUT}" stroke-width="1"/>` +
          `<text x="${f1(rg.cx)}" y="${f1(rg.cy + 2.2)}" font-size="6.5" font-weight="700" text-anchor="middle" fill="${OUT}" font-family="ui-sans-serif, system-ui, sans-serif">${rg.index + 1}</text>`;
        return `<path data-region="${rg.index}" d="${rg.d}" fill="${PAPER}" stroke="${OUT}" stroke-width="${sw}" stroke-linejoin="round"/>`;
      }
      // Cel shading: paint the part in its shadow color, lay the lit color
      // over it shifted up-left (leaving a shadow crescent lower-right),
      // then a soft highlight top-left, all clipped to the part.
      const cid = `${uid}-${rg.index}`;
      defs += `<clipPath id="${cid}"><path d="${rg.d}"/></clipPath>`;
      const lift = rg.small ? 2.4 : 4.4;
      return `<g clip-path="url(#${cid})">` +
        `<path data-region="${rg.index}" d="${rg.d}" fill="${shadeOf(fill)}"/>` +
        `<path d="${rg.d}" fill="${fill}" transform="translate(${f1(-lift)},${f1(-lift * 1.3)})"/>` +
        `<ellipse cx="${f1(rg.cx - lift * 2.5)}" cy="${f1(rg.cy - lift * 3.5)}" rx="${f1(rg.small ? 5 : 14)}" ry="${f1(rg.small ? 3 : 8)}" fill="#ffffff" opacity="0.3"/>` +
        `</g>` +
        `<path d="${rg.d}" fill="none" stroke="${outlineOf(fill)}" stroke-width="${sw}" stroke-linejoin="round"/>`;
    };
    for (const rg of parts) { if (rg.index === undefined) continue; if (rg.back) back += draw(rg); else front += draw(rg); }
    let padSVG = "";
    for (const rg of pads) if (rg.index !== undefined) padSVG += draw(rg);

    const by = built.float ? 104 : 108;
    const beltColor = o.evolved ? OUT : (BELTS[o.grade] || BELTS[1]);
    const knot = o.evolved ? "#f59e0b" : C.dark;
    const belt = `<path d="M38,${f1(by - 4)} Q70,${f1(by + 4)} 102,${f1(by - 4)} L102,${f1(by + 4)} Q70,${f1(by + 12)} 38,${f1(by + 4)} Z" fill="${beltColor}" stroke="${OUT}" stroke-width="2.2"/>` +
      `<path d="M66,${f1(by - 1)} l8,0 l2,6 l-12,0 Z" fill="${knot}" stroke="${OUT}" stroke-width="1.8"/>` +
      `<path d="M62,${f1(by + 5)} l-6,12 M78,${f1(by + 5)} l6,12" stroke="${OUT}" stroke-width="3" stroke-linecap="round"/>` +
      `<path d="M62,${f1(by + 5)} l-6,12 M78,${f1(by + 5)} l6,12" stroke="${beltColor}" stroke-width="1.4" stroke-linecap="round"/>`;

    const aura = o.evolved
      ? `<circle cx="70" cy="80" r="66" fill="none" stroke="${C.accent}" stroke-width="3" stroke-dasharray="7 5" opacity="0.9"/>` +
        `<path d="M14,30 l3,7 l7,3 l-7,3 l-3,7 l-3,-7 l-7,-3 l7,-3 Z M124,22 l2.5,6 l6,2.5 l-6,2.5 l-2.5,6 l-2.5,-6 l-6,-2.5 l6,-2.5 Z M126,128 l2,5 l5,2 l-5,2 l-2,5 l-2,-5 l-5,-2 l5,-2 Z" fill="${C.accent}"/>`
      : "";
    const crown = o.evolved ? `<path d="M52,14 L58,2 L64,12 L70,0 L76,12 L82,2 L88,14 Z" fill="#f59e0b" stroke="${OUT}" stroke-width="2" stroke-linejoin="round"/>` : "";
    const shadow = `<ellipse cx="70" cy="${built.float ? 138 : 140}" rx="${built.float ? 22 : 34}" ry="5" fill="${OUT}" opacity="0.12"/>`;
    const label = o.silhouette ? "An undiscovered kata, shown as a silhouette"
      : o.label || `${o.name || "Unknown kata"}, ${fills} of ${o.regions} regions colored${o.evolved ? ", evolved" : ""}`;
    const open = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 140 148" width="${o.size}" height="${f1(o.size * 148 / 140)}" role="img" aria-label="${esc(label)}">`;
    if (o.silhouette) {
      return open + shadow + back + front + padSVG + `<text x="${f1(built.face.cx)}" y="${f1(built.face.cy + 12)}" font-size="34" font-weight="900" text-anchor="middle" fill="#f8fafc" font-family="ui-sans-serif, system-ui, sans-serif">?</text></svg>`;
    }
    const irisDark = d.eyeColor && HEX.test(d.eyeColor) ? d.eyeColor : mix(C.accent, "#111827", 0.35);
    defs += `<radialGradient id="${uid}-iris" cx="0.4" cy="0.35" r="0.8"><stop offset="0" stop-color="${mix(irisDark, "#ffffff", 0.45)}"/><stop offset="0.55" stop-color="${irisDark}"/><stop offset="1" stop-color="${mix(irisDark, "#000000", 0.5)}"/></radialGradient>`;
    // A slight lean (direction from the seed) so the figure stands like a
    // character rather than a diagram. Badges lean with it.
    const lean = (seed % 2 ? -1 : 1) * (built.float ? 2 : 4);
    const figure = back + front + padSVG + belt + (o.fills > 0 ? built.details || "" : "") + face(built.face, C, d, uid) + crown + badges;
    return open + `<defs>${defs}</defs>` + aura + shadow + `<g transform="rotate(${lean} 70 100)">` + figure + `</g></svg>`;
  }

  // regionCount reports how many regions a design has with its pattern
  // marks: the roster stores this so the server knows when a kata is caught.
  function regionCount(design) {
    const d = Object.assign({ body: "blob", vibe: "cute", hue: 0, pattern: "spots", features: [] }, design || {});
    if (!BODIES[d.body]) d.body = "blob";
    const C = scheme(d);
    let n = BODIES[d.body](C, { cute: d.vibe !== "cool" }).parts.length;
    for (const f of d.features) if (FEATURES[f]) n += FEATURES[f](C).length;
    // A floor keeps every kata a few rounds away from "caught".
    return Math.max(MIN_REGIONS, n + (PATTERN_MARKS[d.pattern] || 0));
  }
  const MIN_REGIONS = 10;
  const PATTERN_MARKS = { none: 0, split: 1, stripes: 3, checks: 3, patches: 3, spots: 3 };

  const api = { creature, regionCount, rng, scheme, BODIES: Object.keys(BODIES), FEATURES: Object.keys(FEATURES) };
  if (typeof module !== "undefined" && module.exports) module.exports = api;
  if (typeof window !== "undefined") window.KataSVG = api;
  if (typeof globalThis !== "undefined") globalThis.KataSVG = api;
})();
