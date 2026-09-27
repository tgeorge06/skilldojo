// Color by number: the weekly mosaic, math key pages, and a free-form
// cooldown painter. Mixed into dojo() by app.js; everything but cooldown
// painting needs a signed-in child.
function paintMixin() {
  return {
    mosaic: null,
    mosaicError: "",
    page: null,
    pageId: "",
    pageError: "",
    pageAnswer: "",
    pageSelected: null,
    pageBusy: false,
    lastFill: null,
    cooldown: null, // { seed, palette, regions, colors: [] }
    cooldownColor: 0,

    // ---- weekly mosaic ----
    async loadMosaic() {
      this.mosaicError = "";
      try {
        const res = await fetch("/api/mosaic/week", { headers: { Accept: "application/json" } });
        const data = await res.json().catch(() => ({}));
        if (!res.ok) throw new Error(data.error || "Could not load this week's mosaic.");
        this.mosaic = data;
      } catch (e) {
        this.mosaicError = e.message;
      }
    },
    // The cells to draw: revealed ones show the picture, the rest stay grey.
    mosaicCells(m) {
      if (!m || !m.cells) return [];
      const revealed = new Set((m.order || []).slice(0, m.revealed));
      const out = [];
      for (let i = 0; i < m.cells.length; i += 1) {
        const shown = revealed.has(i);
        out.push({ i, color: shown ? m.palette[Number(m.cells[i])] : "", shown });
      }
      return out;
    },
    mosaicLabel(m) {
      if (!m) return "";
      return `This week's mosaic, ${m.revealed} of ${m.total} tiles revealed`;
    },
    rewardMosaic() {
      if (!this.reward || !this.reward.mosaic) return null;
      try {
        return typeof this.reward.mosaic === "string" ? JSON.parse(this.reward.mosaic) : this.reward.mosaic;
      } catch {
        return null;
      }
    },

    // ---- math key pages ----
    async openPage() {
      this.error = "";
      this.pageError = "";
      this.pageBusy = true;
      try {
        // Keep the candidate id local so a failed start never pairs the old
        // page with a new id.
        const candidate = newRoundId();
        const page = await this.post("/api/paint/page", { page_id: candidate, ops: this.ops, grade: this.grade });
        this.pageId = candidate;
        this.page = page;
        this.pageSelected = null;
        this.pageAnswer = "";
        this.lastFill = null;
        this.view = "paint-page";
        this.moveToTop("#paint-heading");
      } catch (e) {
        this.error = e.message;
      } finally {
        this.pageBusy = false;
      }
    },
    selectRegion(idx) {
      const r = this.page && this.page.regions[idx];
      if (!r || r.filled) return;
      this.pageSelected = idx;
      this.pageAnswer = "";
      this.lastFill = null;
      requestAnimationFrame(() => document.querySelector("#paint-answer")?.focus());
    },
    async submitFill() {
      if (this.pageSelected === null || !this.pageAnswer.trim() || this.pageBusy) return;
      this.pageBusy = true;
      this.pageError = "";
      try {
        const res = await this.post("/api/paint/fill", { page_id: this.pageId, idx: this.pageSelected, answer: this.pageAnswer.trim() });
        this.page = res.page;
        this.lastFill = { right: res.right, count: (res.filled || []).length };
        if (res.right) {
          this.pageSelected = null;
          this.pageAnswer = "";
          if (res.page.done) confettiBurst();
        } else {
          this.pageAnswer = "";
        }
      } catch (e) {
        this.pageError = e.message;
      } finally {
        this.pageBusy = false;
      }
    },
    // Each answer owns a stable color derived from the answer itself, so a
    // region never changes color when another answer is solved later.
    pagePalette() {
      return ["#ef476f", "#ffd166", "#06d6a0", "#3a86ff", "#7b5cff", "#f4a261", "#8ecae6", "#8fd694",
        "#ffb4a2", "#c7b8ff", "#a0f0e0", "#f2e94e", "#e76f51", "#2a9d8f", "#e9c46a", "#264653",
        "#b5179e", "#4cc9f0", "#90be6d", "#f8961e", "#577590", "#f9c74f", "#43aa8b", "#9d4edd"];
    },
    answerColor(answer) {
      let h = 0;
      for (const ch of String(answer)) h = (h * 31 + ch.charCodeAt(0)) >>> 0;
      const palette = this.pagePalette();
      return palette[h % palette.length];
    },
    pageLegend() {
      if (!this.page) return [];
      const seen = [];
      for (const r of this.page.regions) {
        if (r.filled && r.answer && !seen.includes(r.answer)) seen.push(r.answer);
      }
      return seen.map((answer) => ({ answer, color: this.answerColor(answer) }));
    },
    regionColor(r) {
      return r.filled && r.answer ? this.answerColor(r.answer) : "";
    },
    // Regions are laid out as a mandala of wedges around a center, seeded
    // by the page so every page looks a little different.
    regionShape(idx, total, seed) {
      const ring = idx < 6 ? 0 : idx < 14 ? 1 : 2;
      const start = ring === 0 ? 0 : ring === 1 ? 6 : 14;
      const count = ring === 0 ? Math.min(6, total) : ring === 1 ? Math.min(8, total - 6) : total - 14;
      const inner = [18, 48, 80][ring];
      const outer = [46, 78, 110][ring];
      const rot = ((seed % 360) * Math.PI) / 180;
      const a0 = rot + ((idx - start) / count) * Math.PI * 2;
      const a1 = rot + ((idx - start + 1) / count) * Math.PI * 2;
      const p = (r, a) => `${(120 + r * Math.cos(a)).toFixed(1)},${(120 + r * Math.sin(a)).toFixed(1)}`;
      const path = `M${p(inner, a0)} L${p(outer, a0)} A${outer},${outer} 0 0 1 ${p(outer, a1)} L${p(inner, a1)} A${inner},${inner} 0 0 0 ${p(inner, a0)} Z`;
      const mid = (a0 + a1) / 2;
      const rm = (inner + outer) / 2;
      return { path, x: 120 + rm * Math.cos(mid), y: 120 + rm * Math.sin(mid) };
    },

    // The page is rendered as one SVG string (a <template> inside <svg> is
    // not a real template, so x-for cannot be used there). Every value is
    // numeric or escaped before it reaches the markup.
    pageSVG() {
      if (!this.page) return "";
      const esc = (v) => String(v).replace(/[&<>"']/g, (ch) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[ch]));
      let out = "";
      for (const r of this.page.regions) {
        const shape = this.regionShape(r.idx, this.page.total, this.page.seed);
        const selected = this.pageSelected === r.idx;
        const fill = r.filled ? this.regionColor(r) : "#f8fafc";
        const label = `region ${r.idx + 1}, ${esc(r.prompt)}, ${r.filled ? "colored" : "unfilled"}`;
        out += `<g data-region="${r.idx}" role="button" tabindex="0" aria-label="${label}" aria-pressed="${selected}" style="cursor:pointer">` +
          `<path d="${shape.path}" fill="${esc(fill)}" stroke="${selected ? "#7b5cff" : "#94a3b8"}" stroke-width="${selected ? 3 : 1.5}"/>` +
          `<text x="${shape.x.toFixed(1)}" y="${(shape.y + 3).toFixed(1)}" text-anchor="middle" font-size="9" font-weight="700" fill="${r.filled ? "#1e293b" : "#334155"}" font-family="ui-sans-serif, system-ui, sans-serif">${esc(r.filled ? r.answer : r.prompt)}</text>` +
          `</g>`;
      }
      return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 240 240" class="mx-auto block w-full max-w-md" role="group" aria-label="Color by number page">${out}</svg>`;
    },
    // Delegated click/keyboard handling for the region groups.
    pickRegion(event) {
      const target = event.target.closest("[data-region]");
      if (!target) return;
      if (event.type === "keydown" && event.key !== "Enter" && event.key !== " ") return;
      if (event.type === "keydown") event.preventDefault();
      this.selectRegion(Number(target.dataset.region));
    },

    // ---- cooldown painting (no server, no questions) ----
    startCooldown() {
      if (!this.child) return; // signed-in feature; anonymous play is unchanged
      const list = this.rewardCreatures();
      const t = list[0];
      this.cooldown = t
        ? { seed: t.seed, palette: t.palette, regions: t.regions, colors: Array(t.regions).fill("") }
        : { seed: 11, palette: "meadow", regions: 20, colors: Array(20).fill("") };
      this.cooldownColor = 0;
      this.view = "cooldown";
      this.moveToTop("#cooldown-heading");
    },
    cooldownPalette() {
      return ["#ef476f", "#ffd166", "#06d6a0", "#3a86ff", "#7b5cff", "#f4a261", "#8ecae6", "#1e293b"];
    },
    cooldownSVG() {
      if (!this.cooldown || typeof KataSVG === "undefined") return "";
      return KataSVG.creature(this.cooldown.seed, {
        palette: this.cooldown.palette, regions: this.cooldown.regions, fills: 0, name: "Your painting", size: 240,
        regionColors: this.cooldown.colors,
      });
    },
    // Clicks land on the SVG; find the region circle under the pointer.
    paintAt(event) {
      const target = event.target.closest("[data-region]");
      if (!target || !this.cooldown) return;
      const i = Number(target.dataset.region);
      this.cooldown.colors[i] = this.cooldownPalette()[this.cooldownColor];
      this.cooldown = { ...this.cooldown, colors: [...this.cooldown.colors] };
    },
  };
}
