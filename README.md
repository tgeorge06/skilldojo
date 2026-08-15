# SkillDojo 🥋

A tiny practice-sheet app for kids. Pick what to practice, pick your belt
(difficulty), answer the sheet, and get graded — with confetti for a perfect
score. Math is the first dojo; the code treats it as one subject so more can
be added later.

No login, no database, no tracking. Everything lives in memory; answers are
kept server-side so peeking at the network tab doesn't help. 🙂

## Run it

```sh
make run    # http://127.0.0.1:8080
```

That's it — a single Go binary with all assets embedded (`make build` produces
`./skilldojo`). Works fully offline.

## Practice options

- **Add & Subtract** — multi-digit, never negative
- **Multiplication** — times tables up to 12×12
- **Division** — always divides evenly
- **Fractions** — same-denominator add/subtract, answer like `3/4`
  (equivalent fractions are accepted: `2/4` = `1/2`)

Belts: ⬜ White (easy) → 🟨 Yellow → ⬛ Black (hard). Sheets of 10/20/30.

## Development

- `make test` / `make vet` — the generators are unit-tested in
  `internal/sheet`.
- The UI is Alpine.js + Tailwind v4 using the Vector house theme. After
  changing templates or classes: `npm install` once, then `make css` to
  rebuild `static/app.css` (committed, so runtime needs no Node).
