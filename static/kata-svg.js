// Procedural kata renderer. A pure function of (seed, palette, regions,
// fills, evolved) to an SVG string, so the same creature looks the same on
// every device and no artwork ships with the app.
//
// Every creature is a full body built from an archetype (blob, beast, bird,
// bug, aquatic, dragon) in a thick-outline, big-eyed monster style. Body
// parts are the color-by-number regions: they fill in a fixed order as the
// child earns fills, big parts first, and markings pad the count out to the
// creature's region total. Unfilled parts render as white line art with a
// small number badge, so an undiscovered kata reads as a coloring page.
(function () {
  "use strict";

  const PALETTES = {
    meadow: ["#4f9d69", "#a7e3b1", "#f2c14e", "#2f5d3a"],
    tide:   ["#3a86ff", "#a5d8ff", "#ffd166", "#1d4ed8"],
    ember:  ["#ef476f", "#ffb4a2", "#ffd166", "#9f1239"],
    dusk:   ["#7b5cff", "#d5c8ff", "#ffb703", "#4c1d95"],
    aurora: ["#06d6a0", "#b3f5e6", "#f4a261", "#0f766e"],
  };
  const OUT = "#1f2937";      // ink outline
  const PAPER = "#ffffff";    // unfilled region
  const BADGE = "#e2e8f0";
  const CHEEK = "#fda4af";

  // mulberry32: tiny seeded PRNG so a seed always yields the same creature.
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
  const pick = (r, list) => list[Math.floor(r() * list.length)];
  const f1 = (n) => Number(n).toFixed(1);

  function esc(s) {
    return String(s).replace(/[&<>"']/g, (ch) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[ch]));
  }

  // ---- shape helpers (all return path "d" strings) ----
  function ellipse(cx, cy, rx, ry, rot) {
    // Four-arc ellipse; rotation via a transform-free approximation is not
    // needed for our cases, so rot is applied by rotating the two extreme
    // points only when small. We keep it simple: no rotation.
    return `M${f1(cx - rx)},${f1(cy)} a${f1(rx)},${f1(ry)} 0 1 0 ${f1(rx * 2)},0 a${f1(rx)},${f1(ry)} 0 1 0 ${f1(-rx * 2)},0 Z`;
  }
  function poly(points) {
    return "M" + points.map(([x, y]) => `${f1(x)},${f1(y)}`).join(" L") + " Z";
  }
  function egg(cx, cy, rx, ry, top) {
    // Egg: narrower at the top (top < 1) or bottom (top > 1).
    const t = top || 0.75;
    return `M${f1(cx)},${f1(cy - ry)} C${f1(cx + rx * t)},${f1(cy - ry)} ${f1(cx + rx)},${f1(cy - ry * 0.2)} ${f1(cx + rx)},${f1(cy + ry * 0.15)} ` +
      `C${f1(cx + rx)},${f1(cy + ry * 0.7)} ${f1(cx + rx * 0.6)},${f1(cy + ry)} ${f1(cx)},${f1(cy + ry)} ` +
      `C${f1(cx - rx * 0.6)},${f1(cy + ry)} ${f1(cx - rx)},${f1(cy + ry * 0.7)} ${f1(cx - rx)},${f1(cy + ry * 0.15)} ` +
      `C${f1(cx - rx)},${f1(cy - ry * 0.2)} ${f1(cx - rx * t)},${f1(cy - ry)} ${f1(cx)},${f1(cy - ry)} Z`;
  }
  function teardrop(cx, cy, rx, ry, dir) {
    // Leaf/teardrop pointing in dir (-1 left, 1 right), used for wings/fins.
    const tip = cx + dir * rx;
    return `M${f1(cx - dir * rx * 0.2)},${f1(cy)} Q${f1(cx)},${f1(cy - ry)} ${f1(tip)},${f1(cy - ry * 0.4)} Q${f1(cx + dir * rx * 0.6)},${f1(cy + ry * 0.5)} ${f1(cx - dir * rx * 0.2)},${f1(cy)} Z`;
  }
  function tailCurl(x, y, dir, len, up) {
    // A tapering curled tail starting at (x, y).
    const ex = x + dir * len, ey = y - up;
    return `M${f1(x)},${f1(y - 6)} Q${f1(x + dir * len * 0.7)},${f1(y - 4)} ${f1(ex)},${f1(ey)} ` +
      `Q${f1(ex + dir * 6)},${f1(ey + 10)} ${f1(ex - dir * 4)},${f1(ey + 12)} ` +
      `Q${f1(x + dir * len * 0.6)},${f1(y + 8)} ${f1(x)},${f1(y + 6)} Z`;
  }
  function batWing(x, y, dir, size) {
    const s = size, d = dir;
    return `M${f1(x)},${f1(y)} L${f1(x + d * s * 0.9)},${f1(y - s * 0.9)} L${f1(x + d * s * 1.1)},${f1(y - s * 0.35)} ` +
      `L${f1(x + d * s * 1.25)},${f1(y + s * 0.15)} L${f1(x + d * s * 0.85)},${f1(y + s * 0.05)} L${f1(x + d * s * 0.55)},${f1(y + s * 0.45)} ` +
      `L${f1(x + d * s * 0.25)},${f1(y + s * 0.2)} Z`;
  }
  function spike(x, y, w, h, dir) {
    return poly([[x - w / 2, y], [x + w / 2, y], [x + (dir || 0) * w * 0.3, y - h]]);
  }

  // ---- archetypes: return { parts, face, crest, extraSpots } ----
  // A part: { d, color, cx, cy, name, thin }. Order = draw order and fill order
  // is decided later (big parts first).
  function archetypeBlob(r, C) {
    const parts = [];
    const cx = 70;
    parts.push({ name: "body", d: egg(cx, 96, 34, 32, 0.9), color: C.base, cx, cy: 106, big: 2 });
    parts.push({ name: "head", d: ellipse(cx, 54, 31, 28), color: C.base, cx, cy: 44, big: 1 });
    parts.push({ name: "belly", d: ellipse(cx, 104, 20, 18), color: C.light, cx, cy: 104, big: 3 });
    if (r() < 0.5) {
      parts.push({ name: "left arm", d: ellipse(34, 94, 9, 14), color: C.base, cx: 34, cy: 94 });
      parts.push({ name: "right arm", d: ellipse(106, 94, 9, 14), color: C.base, cx: 106, cy: 94 });
    } else {
      parts.push({ name: "left arm", d: "M40,84 Q22,96 30,112 Q36,106 44,100 Z", color: C.base, cx: 32, cy: 100 });
      parts.push({ name: "right arm", d: "M100,84 Q118,96 110,112 Q104,106 96,100 Z", color: C.base, cx: 108, cy: 100 });
    }
    parts.push({ name: "left foot", d: ellipse(52, 128, 13, 8), color: C.dark, cx: 52, cy: 128 });
    parts.push({ name: "right foot", d: ellipse(88, 128, 13, 8), color: C.dark, cx: 88, cy: 128 });
    const ears = pick(r, ["round", "point", "none"]);
    if (ears === "round") {
      parts.push({ name: "left ear", d: ellipse(46, 29, 9, 11), color: C.base, cx: 46, cy: 29, back: true });
      parts.push({ name: "right ear", d: ellipse(94, 29, 9, 11), color: C.base, cx: 94, cy: 29, back: true });
    } else if (ears === "point") {
      parts.push({ name: "left ear", d: poly([[44, 36], [38, 14], [58, 28]]), color: C.base, cx: 46, cy: 26, back: true });
      parts.push({ name: "right ear", d: poly([[96, 36], [102, 14], [82, 28]]), color: C.base, cx: 94, cy: 26, back: true });
    }
    return { parts, face: { cx, cy: 56, spread: 12, eye: 8 }, spotArea: { cx, cy: 92, rx: 26, ry: 22 } };
  }

  function archetypeBeast(r, C) {
    const parts = [];
    const cx = 70;
    parts.push({ name: "tail", d: tailCurl(102, 98, 1, 26, 34), color: C.base, cx: 122, cy: 78, back: true });
    parts.push({ name: "left back leg", d: ellipse(44, 118, 10, 14), color: C.dark, cx: 44, cy: 118, back: true });
    parts.push({ name: "right back leg", d: ellipse(96, 118, 10, 14), color: C.dark, cx: 96, cy: 118, back: true });
    parts.push({ name: "body", d: ellipse(cx, 98, 38, 27), color: C.base, cx, cy: 110, big: 2 });
    parts.push({ name: "belly", d: ellipse(cx, 106, 24, 15), color: C.light, cx, cy: 106, big: 3 });
    parts.push({ name: "left leg", d: ellipse(54, 124, 11, 12), color: C.base, cx: 54, cy: 126 });
    parts.push({ name: "right leg", d: ellipse(86, 124, 11, 12), color: C.base, cx: 86, cy: 126 });
    parts.push({ name: "head", d: ellipse(cx, 56, 30, 26), color: C.base, cx, cy: 44, big: 1 });
    parts.push({ name: "muzzle", d: ellipse(cx, 66, 13, 9), color: C.light, cx, cy: 68 });
    const ears = pick(r, ["point", "round", "long", "floppy"]);
    if (ears === "point") {
      parts.push({ name: "left ear", d: poly([[46, 40], [40, 16], [60, 32]]), color: C.base, cx: 48, cy: 28, back: true });
      parts.push({ name: "right ear", d: poly([[94, 40], [100, 16], [80, 32]]), color: C.base, cx: 92, cy: 28, back: true });
    } else if (ears === "round") {
      parts.push({ name: "left ear", d: ellipse(46, 32, 10, 10), color: C.base, cx: 46, cy: 32, back: true });
      parts.push({ name: "right ear", d: ellipse(94, 32, 10, 10), color: C.base, cx: 94, cy: 32, back: true });
    } else if (ears === "long") {
      parts.push({ name: "left ear", d: ellipse(50, 22, 7, 18), color: C.base, cx: 50, cy: 18, back: true });
      parts.push({ name: "right ear", d: ellipse(90, 22, 7, 18), color: C.base, cx: 90, cy: 18, back: true });
    } else {
      parts.push({ name: "left ear", d: ellipse(40, 52, 8, 16), color: C.dark, cx: 40, cy: 52, back: true });
      parts.push({ name: "right ear", d: ellipse(100, 52, 8, 16), color: C.dark, cx: 100, cy: 52, back: true });
    }
    if (r() < 0.5) parts.push({ name: "tail tip", d: ellipse(124, 66, 7, 7), color: C.accent, cx: 124, cy: 66, back: true, small: true });
    return { parts, face: { cx, cy: 54, spread: 13, eye: 7.5, muzzle: true }, spotArea: { cx, cy: 92, rx: 30, ry: 16 } };
  }

  function archetypeBird(r, C) {
    const parts = [];
    const cx = 70;
    parts.push({ name: "left wing", d: teardrop(38, 92, 30, 22, -1), color: C.dark, cx: 26, cy: 88, back: true });
    parts.push({ name: "right wing", d: teardrop(102, 92, 30, 22, 1), color: C.dark, cx: 114, cy: 88, back: true });
    parts.push({ name: "tail feathers", d: poly([[62, 122], [78, 122], [92, 140], [70, 132], [48, 140]]), color: C.accent, cx: 70, cy: 133, back: true });
    parts.push({ name: "body", d: egg(cx, 86, 34, 40, 0.7), color: C.base, cx, cy: 100, big: 1 });
    parts.push({ name: "belly", d: egg(cx, 98, 20, 24, 0.8), color: C.light, cx, cy: 100, big: 3 });
    parts.push({ name: "left foot", d: poly([[56, 126], [64, 126], [62, 136], [50, 136]]), color: C.accent, cx: 57, cy: 131 });
    parts.push({ name: "right foot", d: poly([[76, 126], [84, 126], [90, 136], [78, 136]]), color: C.accent, cx: 83, cy: 131 });
    parts.push({ name: "beak", d: poly([[62, 60], [78, 60], [70, 72]]), color: C.accent, cx: 70, cy: 63, small: true });
    const crest = pick(r, [1, 2, 3]);
    for (let i = 0; i < crest; i++) {
      const x = 70 + (i - (crest - 1) / 2) * 12;
      parts.push({ name: `crest ${i + 1}`, d: poly([[x - 5, 50], [x + 5, 50], [x + (i - 1) * 4, 30]]), color: C.accent, cx: x, cy: 44, back: true, small: true });
    }
    return { parts, face: { cx, cy: 52, spread: 11, eye: 7 }, spotArea: { cx, cy: 78, rx: 24, ry: 18 } };
  }

  function archetypeBug(r, C) {
    const parts = [];
    const cx = 70;
    // six thin legs
    for (let i = 0; i < 3; i++) {
      const y = 84 + i * 14;
      parts.push({ name: `left leg ${i + 1}`, d: poly([[52, y], [30, y + 10], [28, y + 16], [34, y + 14], [54, y + 6]]), color: C.dark, cx: 38, cy: y + 10, back: true, small: true });
      parts.push({ name: `right leg ${i + 1}`, d: poly([[88, y], [110, y + 10], [112, y + 16], [106, y + 14], [86, y + 6]]), color: C.dark, cx: 102, cy: y + 10, back: true, small: true });
    }
    parts.push({ name: "abdomen", d: ellipse(cx, 112, 28, 24), color: C.base, cx, cy: 120, big: 2 });
    parts.push({ name: "left shell", d: teardrop(58, 108, 22, 18, -1), color: C.light, cx: 52, cy: 108, big: 3 });
    parts.push({ name: "right shell", d: teardrop(82, 108, 22, 18, 1), color: C.light, cx: 88, cy: 108, big: 3 });
    parts.push({ name: "thorax", d: ellipse(cx, 82, 20, 15), color: C.base, cx, cy: 86 });
    parts.push({ name: "head", d: ellipse(cx, 52, 26, 24), color: C.base, cx, cy: 42, big: 1 });
    parts.push({ name: "left antenna", d: poly([[58, 30], [50, 10], [56, 8], [63, 28]]), color: C.dark, cx: 54, cy: 16, back: true, small: true });
    parts.push({ name: "right antenna", d: poly([[82, 30], [90, 10], [84, 8], [77, 28]]), color: C.dark, cx: 86, cy: 16, back: true, small: true });
    if (r() < 0.6) {
      parts.push({ name: "left feeler tip", d: ellipse(52, 9, 5, 5), color: C.accent, cx: 52, cy: 9, back: true, small: true });
      parts.push({ name: "right feeler tip", d: ellipse(88, 9, 5, 5), color: C.accent, cx: 88, cy: 9, back: true, small: true });
    } else {
      parts.push({ name: "left feeler tip", d: poly([[48, 14], [52, 2], [58, 12]]), color: C.accent, cx: 52, cy: 9, back: true, small: true });
      parts.push({ name: "right feeler tip", d: poly([[92, 14], [88, 2], [82, 12]]), color: C.accent, cx: 88, cy: 9, back: true, small: true });
    }
    return { parts, face: { cx, cy: 54, spread: 11, eye: 7.5 }, spotArea: { cx, cy: 118, rx: 16, ry: 12 } };
  }

  function archetypeAquatic(r, C) {
    const parts = [];
    const cx = 64;
    parts.push({ name: "tail fin", d: poly([[104, 88], [130, 66], [126, 88], [130, 110]]), color: C.accent, cx: 120, cy: 88, back: true });
    const fin = pick(r, ["sail", "spiky", "round"]);
    if (fin === "sail") parts.push({ name: "dorsal fin", d: poly([[56, 62], [74, 34], [92, 62]]), color: C.accent, cx: 74, cy: 52, back: true });
    else if (fin === "spiky") parts.push({ name: "dorsal fin", d: poly([[50, 64], [58, 40], [66, 58], [76, 34], [84, 58], [94, 44], [98, 66]]), color: C.accent, cx: 74, cy: 54, back: true });
    else parts.push({ name: "dorsal fin", d: "M52,64 Q74,26 98,64 Z", color: C.accent, cx: 74, cy: 50, back: true });
    parts.push({ name: "belly fin", d: poly([[60, 110], [72, 128], [82, 108]]), color: C.accent, cx: 71, cy: 114, back: true, small: true });
    parts.push({ name: "body", d: ellipse(cx, 88, 44, 30), color: C.base, cx: 84, cy: 78, big: 1 });
    parts.push({ name: "belly", d: ellipse(cx + 6, 100, 30, 14), color: C.light, cx: 76, cy: 100, big: 3 });
    parts.push({ name: "side fin", d: teardrop(70, 96, 22, 12, 1), color: C.dark, cx: 80, cy: 94 });
    parts.push({ name: "cheek fin", d: teardrop(34, 90, 14, 10, -1), color: C.dark, cx: 28, cy: 90, small: true });
    parts.push({ name: "lip", d: ellipse(24, 92, 8, 6), color: C.light, cx: 24, cy: 92, small: true });
    return { parts, face: { cx: 42, cy: 78, spread: 10, eye: 7.5, fish: true }, spotArea: { cx: 86, cy: 80, rx: 18, ry: 14 } };
  }

  function archetypeDragon(r, C) {
    const parts = [];
    const cx = 70;
    parts.push({ name: "left wing", d: batWing(48, 84, -1, 34), color: C.dark, cx: 20, cy: 68, back: true });
    parts.push({ name: "right wing", d: batWing(92, 84, 1, 34), color: C.dark, cx: 120, cy: 68, back: true });
    parts.push({ name: "tail", d: tailCurl(100, 104, 1, 24, 22), color: C.base, cx: 118, cy: 92, back: true });
    parts.push({ name: "tail spade", d: poly([[118, 78], [132, 70], [128, 86]]), color: C.accent, cx: 126, cy: 78, back: true, small: true });
    parts.push({ name: "body", d: egg(cx, 98, 34, 34, 0.85), color: C.base, cx, cy: 110, big: 2 });
    parts.push({ name: "belly plates", d: egg(cx, 106, 20, 22, 0.9), color: C.light, cx, cy: 106, big: 3 });
    parts.push({ name: "left arm", d: ellipse(38, 96, 8, 12), color: C.base, cx: 38, cy: 96, small: true });
    parts.push({ name: "right arm", d: ellipse(102, 96, 8, 12), color: C.base, cx: 102, cy: 96, small: true });
    parts.push({ name: "left foot", d: ellipse(52, 130, 13, 8), color: C.dark, cx: 52, cy: 130 });
    parts.push({ name: "right foot", d: ellipse(88, 130, 13, 8), color: C.dark, cx: 88, cy: 130 });
    parts.push({ name: "head", d: ellipse(cx, 54, 30, 26), color: C.base, cx, cy: 44, big: 1 });
    parts.push({ name: "snout", d: ellipse(cx, 66, 14, 9), color: C.light, cx, cy: 68, small: true });
    const horns = pick(r, ["twin", "curved", "single"]);
    if (horns === "twin") {
      parts.push({ name: "left horn", d: poly([[50, 36], [42, 10], [62, 30]]), color: C.accent, cx: 50, cy: 24, back: true });
      parts.push({ name: "right horn", d: poly([[90, 36], [98, 10], [78, 30]]), color: C.accent, cx: 90, cy: 24, back: true });
    } else if (horns === "curved") {
      parts.push({ name: "left horn", d: "M52,36 Q30,30 36,10 Q40,24 60,30 Z", color: C.accent, cx: 42, cy: 24, back: true });
      parts.push({ name: "right horn", d: "M88,36 Q110,30 104,10 Q100,24 80,30 Z", color: C.accent, cx: 98, cy: 24, back: true });
    } else {
      parts.push({ name: "horn", d: poly([[62, 32], [70, 6], [78, 32]]), color: C.accent, cx: 70, cy: 22, back: true });
      parts.push({ name: "left frill", d: poly([[44, 44], [30, 34], [46, 60]]), color: C.light, cx: 40, cy: 46, back: true, small: true });
      parts.push({ name: "right frill", d: poly([[96, 44], [110, 34], [94, 60]]), color: C.light, cx: 100, cy: 46, back: true, small: true });
    }
    return { parts, face: { cx, cy: 52, spread: 12, eye: 7, brow: true }, spotArea: { cx, cy: 92, rx: 22, ry: 14 } };
  }

  const ARCHETYPES = [archetypeBlob, archetypeBeast, archetypeBird, archetypeBug, archetypeAquatic, archetypeDragon];

  // ---- face, drawn on top and never a region ----
  function face(r, F, C, opts) {
    const style = pick(r, ["round", "round", "sparkle", "sleepy", "brave"]);
    const mouth = pick(r, ["smile", "open", "fang", "cat"]);
    let out = "";
    const eyes = F.fish ? [[F.cx, F.cy]] : [[F.cx - F.spread, F.cy], [F.cx + F.spread, F.cy]];
    for (const [ex, ey] of eyes) {
      const e = F.eye;
      out += `<ellipse cx="${f1(ex)}" cy="${f1(ey)}" rx="${f1(e)}" ry="${f1(e * 1.12)}" fill="#fff" stroke="${OUT}" stroke-width="2.4"/>`;
      if (style === "sleepy") {
        out += `<path d="M${f1(ex - e)},${f1(ey - e * 0.2)} a${f1(e)},${f1(e)} 0 0 1 ${f1(e * 2)},0" fill="${OUT}"/>`;
      }
      out += `<circle cx="${f1(ex + 1)}" cy="${f1(ey + 1.5)}" r="${f1(e * 0.62)}" fill="${C.dark}"/>`;
      out += `<circle cx="${f1(ex + 1.2)}" cy="${f1(ey + 1.8)}" r="${f1(e * 0.36)}" fill="${OUT}"/>`;
      out += `<circle cx="${f1(ex - 1.6)}" cy="${f1(ey - 1.8)}" r="${f1(e * 0.26)}" fill="#fff"/>`;
      if (style === "sparkle") out += `<circle cx="${f1(ex + 2.6)}" cy="${f1(ey + 3.2)}" r="${f1(e * 0.12)}" fill="#fff"/>`;
      if (style === "brave" || F.brow) {
        const d = ex < F.cx ? -1 : 1;
        out += `<path d="M${f1(ex - d * e)},${f1(ey - e * 1.4)} L${f1(ex + d * e * 0.9)},${f1(ey - e * 1.9)}" stroke="${OUT}" stroke-width="2.6" stroke-linecap="round"/>`;
      }
    }
    // cheeks
    if (!F.fish) {
      out += `<ellipse cx="${f1(F.cx - F.spread - 6)}" cy="${f1(F.cy + 8)}" rx="4" ry="2.4" fill="${CHEEK}" opacity="0.8"/>`;
      out += `<ellipse cx="${f1(F.cx + F.spread + 6)}" cy="${f1(F.cy + 8)}" rx="4" ry="2.4" fill="${CHEEK}" opacity="0.8"/>`;
    }
    const my = F.cy + (F.muzzle ? 14 : 12);
    const mx = F.fish ? F.cx - 12 : F.cx;
    if (mouth === "smile") {
      out += `<path d="M${f1(mx - 7)},${f1(my)} Q${f1(mx)},${f1(my + 7)} ${f1(mx + 7)},${f1(my)}" fill="none" stroke="${OUT}" stroke-width="2.4" stroke-linecap="round"/>`;
    } else if (mouth === "open") {
      out += `<path d="M${f1(mx - 7)},${f1(my)} Q${f1(mx)},${f1(my + 10)} ${f1(mx + 7)},${f1(my)} Z" fill="${OUT}"/>`;
      out += `<path d="M${f1(mx - 3)},${f1(my + 3)} Q${f1(mx)},${f1(my + 7)} ${f1(mx + 3)},${f1(my + 3)} Z" fill="${CHEEK}"/>`;
    } else if (mouth === "fang") {
      out += `<path d="M${f1(mx - 7)},${f1(my)} Q${f1(mx)},${f1(my + 6)} ${f1(mx + 7)},${f1(my)}" fill="none" stroke="${OUT}" stroke-width="2.4" stroke-linecap="round"/>`;
      out += `<path d="M${f1(mx + 3)},${f1(my + 1)} L${f1(mx + 5.5)},${f1(my + 6)} L${f1(mx + 7.5)},${f1(my + 0.5)} Z" fill="#fff" stroke="${OUT}" stroke-width="1.2"/>`;
    } else {
      out += `<path d="M${f1(mx - 7)},${f1(my)} Q${f1(mx - 3.5)},${f1(my + 5)} ${f1(mx)},${f1(my)} Q${f1(mx + 3.5)},${f1(my + 5)} ${f1(mx + 7)},${f1(my)}" fill="none" stroke="${OUT}" stroke-width="2.4" stroke-linecap="round"/>`;
    }
    if (F.muzzle) out += `<ellipse cx="${f1(F.cx)}" cy="${f1(F.cy + 9)}" rx="3.4" ry="2.4" fill="${OUT}"/>`;
    return out;
  }

  // ---- markings pad the region count ----
  // Mirrored pairs of well-spaced spots read as a designed pattern rather
  // than clutter, and leave room for the number badges while unfilled.
  function markings(r, area, count, C) {
    const list = [];
    const placed = [];
    const minGap = 11;
    let guard = 0;
    while (list.length < count && guard++ < 200) {
      const a = r() * Math.PI, d = 0.35 + r() * 0.6; // right half only, then mirror
      const dx = Math.abs(Math.cos(a) * area.rx * d), dy = Math.sin(a * 2) * area.ry * d * 0.9;
      const candidates = [[area.cx + dx, area.cy + dy], [area.cx - dx, area.cy + dy]];
      if (dx < 5) candidates.pop(); // near the centre line, a single spot
      const ok = candidates.every(([x, y]) => placed.every(([px, py]) => Math.hypot(px - x, py - y) >= minGap));
      if (!ok) continue;
      const rad = 3 + r() * 1.4;
      const color = list.length % 4 < 2 ? C.accent : C.light;
      for (const [x, y] of candidates) {
        if (list.length >= count) break;
        placed.push([x, y]);
        list.push({ name: `spot ${list.length + 1}`, d: ellipse(x, y, rad, rad * 0.92), color, cx: x, cy: y, small: true });
      }
    }
    return list;
  }

  /**
   * creature(seed, opts) -> SVG string.
   * opts: { palette, regions, fills, evolved, name, label, size, regionColors, silhouette }
   * silhouette: render an undiscovered kata as a dark shape with a "?", the
   * hunt hook; its outline still hints at what lives there.
   * regionColors: optional per-region hex colors for free painting; only
   * strict #rrggbb values are honoured so nothing else can reach the markup.
   */
  function creature(seed, opts) {
    const o = Object.assign({ palette: "meadow", regions: 20, fills: 0, evolved: false, name: "", label: "", size: 120, silhouette: false }, opts || {});
    const r = rng(seed);
    const p = PALETTES[o.palette] || PALETTES.meadow;
    const C = { base: p[0], light: p[1], accent: p[2], dark: p[3] };
    const build = pick(r, ARCHETYPES);
    const { parts, face: F, spotArea } = build(r, C);

    // Fill order: big parts first, then the rest in draw order, then
    // markings. Region index follows fill order; draw order is separate.
    const ordered = [...parts].sort((a, b) => (a.big || 9) - (b.big || 9));
    const padCount = Math.max(0, o.regions - ordered.length);
    const pads = markings(r, spotArea, padCount, C);
    const regions = [...ordered, ...pads].slice(0, o.regions);
    regions.forEach((rg, i) => { rg.index = i; });

    const fills = Math.max(0, Math.min(o.regions, o.fills));
    const custom = Array.isArray(o.regionColors) ? o.regionColors : null;
    const isFilled = (rg) => {
      const painted = custom && /^#[0-9a-f]{6}$/i.test(custom[rg.index] || "") ? custom[rg.index] : "";
      return painted || (rg.index < fills ? rg.color : "");
    };

    // Belt: a dojo band across the body, not a region. Black once evolved.
    const beltY = 108;
    const belt = `<path d="M${f1(38)},${f1(beltY - 4)} Q70,${f1(beltY + 4)} 102,${f1(beltY - 4)} L102,${f1(beltY + 4)} Q70,${f1(beltY + 12)} 38,${f1(beltY + 4)} Z" fill="${o.evolved ? OUT : C.accent}" stroke="${OUT}" stroke-width="2.2"/>` +
      `<path d="M66,${f1(beltY - 1)} l8,0 l2,6 l-12,0 Z" fill="${o.evolved ? "#f59e0b" : C.dark}" stroke="${OUT}" stroke-width="1.8"/>` +
      `<path d="M62,${f1(beltY + 5)} l-6,12 M78,${f1(beltY + 5)} l6,12" stroke="${OUT}" stroke-width="3" stroke-linecap="round"/>` +
      `<path d="M62,${f1(beltY + 5)} l-6,12 M78,${f1(beltY + 5)} l6,12" stroke="${o.evolved ? "#f59e0b" : C.accent}" stroke-width="1.4" stroke-linecap="round"/>`;

    let back = "", front = "", badges = "";
    const draw = (rg) => {
      if (o.silhouette) {
        return `<path data-region="${rg.index}" d="${rg.d}" fill="#64748b" stroke="#334155" stroke-width="${rg.small ? 2.2 : 3}" stroke-linejoin="round"/>`;
      }
      const fill = isFilled(rg);
      const shape = `<path data-region="${rg.index}" d="${rg.d}" fill="${fill || PAPER}" stroke="${OUT}" stroke-width="${rg.small ? 2.2 : 3}" stroke-linejoin="round"/>`;
      if (!fill) {
        badges += `<circle cx="${f1(rg.cx)}" cy="${f1(rg.cy)}" r="5.2" fill="${BADGE}" stroke="${OUT}" stroke-width="1"/>` +
          `<text x="${f1(rg.cx)}" y="${f1(rg.cy + 2.2)}" font-size="6.5" font-weight="700" text-anchor="middle" fill="${OUT}" font-family="ui-sans-serif, system-ui, sans-serif">${rg.index + 1}</text>`;
      }
      return shape;
    };
    for (const rg of parts) {
      if (rg.index === undefined) continue; // trimmed by a small region budget
      if (rg.back) back += draw(rg); else front += draw(rg);
    }
    let padSVG = "";
    for (const rg of pads) if (rg.index !== undefined) padSVG += draw(rg);

    const aura = o.evolved
      ? `<circle cx="70" cy="80" r="66" fill="none" stroke="${C.accent}" stroke-width="3" stroke-dasharray="7 5" opacity="0.9"/>` +
        `<path d="M14,30 l3,7 l7,3 l-7,3 l-3,7 l-3,-7 l-7,-3 l7,-3 Z M124,22 l2.5,6 l6,2.5 l-6,2.5 l-2.5,6 l-2.5,-6 l-6,-2.5 l6,-2.5 Z M126,128 l2,5 l5,2 l-5,2 l-2,5 l-2,-5 l-5,-2 l5,-2 Z" fill="${C.accent}"/>`
      : "";
    const crown = o.evolved ? `<path d="M52,14 L58,2 L64,12 L70,0 L76,12 L82,2 L88,14 Z" fill="#f59e0b" stroke="${OUT}" stroke-width="2" stroke-linejoin="round"/>` : "";
    // A soft ground shadow anchors the figure.
    const shadow = `<ellipse cx="70" cy="140" rx="34" ry="5" fill="${OUT}" opacity="0.12"/>`;
    const label = o.silhouette
      ? "An undiscovered kata, shown as a silhouette"
      : o.label || `${o.name || "Unknown kata"}, ${fills} of ${o.regions} regions colored${o.evolved ? ", evolved" : ""}`;
    const open = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 140 148" width="${o.size}" height="${f1(o.size * 148 / 140)}" role="img" aria-label="${esc(label)}">`;
    if (o.silhouette) {
      const mark = `<text x="${f1(F.cx)}" y="${f1(F.cy + 12)}" font-size="34" font-weight="900" text-anchor="middle" fill="#f8fafc" font-family="ui-sans-serif, system-ui, sans-serif">?</text>`;
      return open + shadow + back + front + padSVG + mark + `</svg>`;
    }
    return open + aura + shadow + back + front + padSVG + belt + face(r, F, C, o) + crown + badges + `</svg>`;
  }

  const api = { creature, rng, PALETTES };
  if (typeof module !== "undefined" && module.exports) module.exports = api;
  if (typeof window !== "undefined") window.KataSVG = api;
  if (typeof globalThis !== "undefined") globalThis.KataSVG = api;
})();
