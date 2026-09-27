// Procedural creature renderer. A pure function of (seed, palette, regions,
// fills, evolved) to an SVG string, so the same creature looks the same on
// every device and no artwork ships with the app. Every part is a numbered
// region; regions fill in a fixed order as the child earns fills.
(function () {
  "use strict";

  const PALETTES = {
    meadow: ["#4f9d69", "#8fd694", "#f2e94e", "#2f5d3a"],
    tide:   ["#3a86ff", "#8ecae6", "#ffd166", "#023e8a"],
    ember:  ["#ef476f", "#ffb4a2", "#ffd166", "#7b2d26"],
    dusk:   ["#7b5cff", "#c7b8ff", "#ffb703", "#3d2c8d"],
    aurora: ["#06d6a0", "#a0f0e0", "#f4a261", "#0b6e4f"],
  };
  const OUTLINE = "#94a3b8";
  const UNFILLED = "#f8fafc";

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

  const BODIES = [
    { name: "round", path: "M60,20 C85,20 100,40 100,65 C100,90 82,110 60,110 C38,110 20,90 20,65 C20,40 35,20 60,20 Z" },
    { name: "tall",  path: "M60,12 C80,12 92,35 92,62 C92,92 78,112 60,112 C42,112 28,92 28,62 C28,35 40,12 60,12 Z" },
    { name: "wide",  path: "M60,30 C95,30 110,50 110,70 C110,95 90,108 60,108 C30,108 10,95 10,70 C10,50 25,30 60,30 Z" },
    { name: "bean",  path: "M50,18 C80,14 104,40 100,70 C96,100 70,114 48,108 C24,102 14,78 20,52 C25,30 34,20 50,18 Z" },
    { name: "drop",  path: "M60,14 C70,34 100,52 100,76 C100,98 82,112 60,112 C38,112 20,98 20,76 C20,52 50,34 60,14 Z" },
  ];
  const CRESTS = [
    (c) => `<path d="M40,26 L34,6 L50,20 Z M80,26 L86,6 L70,20 Z" fill="${c}"/>`,                              // horns
    (c) => `<ellipse cx="38" cy="18" rx="9" ry="14" fill="${c}"/><ellipse cx="82" cy="18" rx="9" ry="14" fill="${c}"/>`, // ears
    (c) => `<path d="M60,4 L72,26 L48,26 Z" fill="${c}"/>`,                                                   // fin
    (c) => `<path d="M60,22 L60,6" stroke="${c}" stroke-width="3" fill="none"/><circle cx="60" cy="5" r="4" fill="${c}"/>`, // antenna
  ];
  const EYES = [
    () => `<circle cx="48" cy="58" r="4" fill="#1e293b"/><circle cx="72" cy="58" r="4" fill="#1e293b"/>`,
    () => `<circle cx="47" cy="58" r="8" fill="#fff"/><circle cx="73" cy="58" r="8" fill="#fff"/><circle cx="49" cy="59" r="4" fill="#1e293b"/><circle cx="71" cy="59" r="4" fill="#1e293b"/>`,
    () => `<path d="M42,58 Q48,52 54,58" stroke="#1e293b" stroke-width="3" fill="none"/><path d="M66,58 Q72,52 78,58" stroke="#1e293b" stroke-width="3" fill="none"/>`,
    () => `<circle cx="48" cy="58" r="5" fill="#1e293b"/><circle cx="72" cy="58" r="5" fill="#1e293b"/><circle cx="50" cy="56" r="1.5" fill="#fff"/><circle cx="74" cy="56" r="1.5" fill="#fff"/>`,
  ];
  const MOUTHS = [
    () => `<path d="M52,76 Q60,84 68,76" stroke="#1e293b" stroke-width="2.5" fill="none"/>`,
    () => `<path d="M54,78 L66,78" stroke="#1e293b" stroke-width="2.5"/>`,
    () => `<ellipse cx="60" cy="78" rx="5" ry="3" fill="#1e293b"/>`,
  ];

  // Region layout: concentric rings of small shapes inside the body box.
  function regionLayout(count, random) {
    const spots = [];
    const rings = [[60, 66, 0, 1], [60, 66, 16, 6], [60, 66, 30, 12], [60, 66, 42, 16]];
    for (const [cx, cy, r, n] of rings) {
      for (let i = 0; i < n && spots.length < count; i += 1) {
        const a = (i / n) * Math.PI * 2 + (r ? random() * 0.3 : 0);
        spots.push({ x: cx + Math.cos(a) * r, y: cy + Math.sin(a) * r * 0.85 + (r ? 4 : 0) });
      }
    }
    while (spots.length < count) spots.push({ x: 30 + random() * 60, y: 40 + random() * 60 });
    return spots.slice(0, count);
  }

  function esc(s) {
    return String(s).replace(/[&<>"]/g, (ch) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" }[ch]));
  }

  /**
   * creature(seed, opts) -> SVG string.
   * opts: { palette, regions, fills, evolved, name, label, size }
   */
  function creature(seed, opts) {
    const o = Object.assign({ palette: "meadow", regions: 20, fills: 0, evolved: false, name: "", label: "", size: 120 }, opts || {});
    const random = rng(seed);
    const colors = PALETTES[o.palette] || PALETTES.meadow;
    const body = BODIES[Math.floor(random() * BODIES.length)];
    const crest = CRESTS[Math.floor(random() * CRESTS.length)];
    const eyes = EYES[Math.floor(random() * EYES.length)];
    const mouth = MOUTHS[Math.floor(random() * MOUTHS.length)];
    const fills = Math.max(0, Math.min(o.regions, o.fills));
    const done = fills >= o.regions;
    const bodyFill = done ? colors[0] : UNFILLED;
    const partFill = done ? colors[3] : UNFILLED;
    const label = o.label || `${o.name || "Unknown kata"}, ${fills} of ${o.regions} regions colored${o.evolved ? ", evolved" : ""}`;

    const spots = regionLayout(o.regions, random);
    let regionsSVG = "";
    spots.forEach((p, i) => {
      const filled = i < fills;
      const color = colors[1 + ((i * 7 + seed) % 2)];
      regionsSVG += `<circle cx="${p.x.toFixed(1)}" cy="${p.y.toFixed(1)}" r="5.5" fill="${filled ? color : UNFILLED}" stroke="${filled ? "none" : OUTLINE}" stroke-width="1" stroke-dasharray="${filled ? "0" : "2 1.5"}"/>`;
      if (!filled) regionsSVG += `<text x="${p.x.toFixed(1)}" y="${(p.y + 2).toFixed(1)}" font-size="5" text-anchor="middle" fill="${OUTLINE}" font-family="ui-sans-serif, system-ui, sans-serif">${i + 1}</text>`;
    });

    const aura = o.evolved
      ? `<circle cx="60" cy="66" r="56" fill="none" stroke="${colors[2]}" stroke-width="3" stroke-dasharray="6 4" opacity="0.9"/>`
      : "";
    const crown = o.evolved ? `<path d="M46,10 L52,0 L58,10 L64,0 L70,10 L74,2 L74,14 L46,14 Z" fill="${colors[2]}"/>` : "";

    return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 124" width="${o.size}" height="${o.size}" role="img" aria-label="${esc(label)}">` +
      aura +
      `<g clip-path="url(#body-${seed})"><clipPath id="body-${seed}"><path d="${body.path}"/></clipPath>` +
      `<path d="${body.path}" fill="${bodyFill}"/>${regionsSVG}</g>` +
      `<path d="${body.path}" fill="none" stroke="${done ? colors[3] : OUTLINE}" stroke-width="2.5"/>` +
      crest(partFill) + crown + eyes() + mouth() +
      `</svg>`;
  }

  const api = { creature, rng, PALETTES };
  if (typeof module !== "undefined" && module.exports) module.exports = api;
  if (typeof window !== "undefined") window.KataSVG = api;
  if (typeof globalThis !== "undefined") globalThis.KataSVG = api;
})();
