# Game Layer Implementation Plan

Status: approved 2026-09-27 (all five decisions in section 11 confirmed: migrations at boot, curriculum as JSON, games behind parent login, Leitner intervals as written, Playwright render gate). Covers the two games agreed in ideation, the
"kata" creature collection with its index and battles, and color by number
(creature reveals, math key pages, weekly mosaic), plus the account and
activity foundation they need. Weekly recap email and Stripe are referenced
where they constrain the schema but are planned separately.

Guiding rules, restated so every PR can be checked against them:

- The free app stays exactly as it is today. Nothing that works without a
  login today ever moves behind one.
- Learning is primary. Rewards are computed and shown only when a session
  ends, never mid-sheet or mid-round.
- The server decides every reward. A client can never post "I got this
  right"; it posts what it did and the server grades it.
- Child data is a nickname and per-round events. No other child PII, ever.
- One process: Go binary, SQLite on a volume, in-process cron. No extra
  hosts.

---

## 0. What exists today (verified)

- `main.go` has one struct, `server{store *sheet.Store, tmpl}`, four routes
  (`GET /static/`, `GET /{$}`, `POST /api/sheet`, `POST /api/grade`), a
  strict JSON decoder (`decodeJSON`, 64 KiB cap, unknown fields rejected,
  trailing data rejected), and no cookies, sessions, or database.
  `go.mod` has zero dependencies.
- Math is already server-graded: `sheet.Store` holds the answer key in
  memory, `Grade` consumes the sheet. That is the pattern the game layer
  extends.
- Spelling is fully client-side. `static/words.js` declares bare globals
  (`SPELLING_SKILLS`, `SPELLING_WORDS`, `SIGHT_WORDS`, skill ids like
  `g4-roots`), and `dojo()` in `static/app.js` runs the whole round. The
  server never sees a spelling result. This is the one structural change
  the games force (see PR 1).
- Session-complete hooks: math at `app.js:129-131` (`view = "math-results"`),
  spelling at `app.js:290-295` inside `nextSpellingWord`
  (`view = "spelling-results"`).
- Tests: Go table tests in `internal/sheet`, and a `node --test` harness
  (`tests/spelling.test.js`) that runs `words.js`, `manifest.js` and
  `app.js` inside a `vm` context with fakes. CI runs both plus a committed
  CSS sync check.

---

## 1. Architecture decisions

| Decision | Choice | Why |
|---|---|---|
| SQLite driver | `modernc.org/sqlite` (pure Go) | No cgo, so `go build` and the Fly image stay trivial. Single writer is fine: one process, WAL mode, `busy_timeout`. |
| Migrations | Embedded SQL files under `internal/db/migrations/`, applied at boot inside a transaction with a `schema_version` table | Hobby-scale app with one instance. **Decision to confirm:** this breaks Tyson's "migrations never auto-applied" rule for Vector services. Alternative is a `skilldojo migrate` subcommand run by hand before deploy. |
| Curriculum source of truth | Move to `internal/curriculum/spelling.json`; a script generates `static/words.js` from it (same pattern as `manifest.js`); Go embeds the JSON | The server must know every word to grade spelling rounds and to bind creatures to skills. One source, two consumers, a CI check that the generated file is in sync (same as the CSS check). |
| Spelling verification | Client still plays the round locally for instant feedback. At round end it posts the ordered guess sequence. The server replays the guesses against its own copy of the word and decides won/lost | Deterministic, cheap, and makes a forged result impossible without changing the kid's experience. Lesson applied: *a fallback to client-posted data is a forgery path unless bounded by something the client cannot produce*. |
| Sessions | Random 32-byte token, stored as SHA-256 hash, HttpOnly Secure SameSite=Lax cookie, 90-day expiry | Standard. Lesson applied: tokens are high-entropy and hashed at rest; `SESSION_SECRET`/`RESEND_API_KEY` refuse empty or placeholder values at boot. |
| Active child | Signed value in the session row (`sessions.active_child_id`), not a cookie the client writes | Every write handler re-binds `child_id` to `account_id` in the query anyway; this just picks the default. |
| Client structure | Keep the no-bundler setup. Split `app.js` into `app.js` (core), `kata.js`, `paint.js`, each exporting a mixin merged in `dojo()` via `Object.assign` | `app.js` is 400 lines today and would triple. Files load in order like `words.js` does now. The `vm` test harness loads them the same way. |
| Entitlement | Server-side `accounts.plan` and `trial_ends_at`. Until Stripe ships, every account is `trial` with no end date | Lesson applied: *a client-side-only lockout is not a lockout*. Auth fails closed. Entitlement failure on a kids app fails open to read-only (you can look at your index, you cannot earn), decided here so it is not improvised later. |
| UI render gate | Add a Playwright smoke test (`make ui-test`) that boots the binary, loads every view, and fails on any `pageerror` | Lesson applied (Used 8): *a template binding to a helper that does not exist passes every unit test*. The games add four new Alpine surfaces; the `vm` harness cannot see template bindings. |

---

## 2. Data model

All tables carry `account_id` alongside `child_id`, and every store method
takes both and puts both in `WHERE`. Lessons applied: *session-bound rows
need the user in the FK*; *a repository method must carry its own tenant
fence even when today's caller pre-scopes the ids*.

```sql
accounts        (id, email UNIQUE, timezone, plan, trial_ends_at, created_at)
login_tokens    (id, account_id, token_hash UNIQUE, expires_at, used_at)
sessions        (id, account_id, token_hash UNIQUE, active_child_id, expires_at)
children        (id, account_id, nickname, grade, created_at, deleted_at)

rounds          (id TEXT PRIMARY KEY,        -- client-generated UUID, idempotency key
                 account_id, child_id, kind, -- 'math' | 'spelling'
                 skill_id, grade, started_at, finished_at,
                 correct, total)
round_items     (round_id, idx, item_key, skill_id, correct,
                 PRIMARY KEY (round_id, idx))

skill_progress  (account_id, child_id, skill_id, box, due_on, reviews_ok,
                 first_ok_on, last_seen_at, PRIMARY KEY (child_id, skill_id))
creature_state  (account_id, child_id, creature_id, state, fills, updated_at,
                 PRIMARY KEY (child_id, creature_id))
page_state      (account_id, child_id, page_id, filled_mask, updated_at,
                 PRIMARY KEY (child_id, page_id))
mosaic_weeks    (account_id, child_id, week_key, cells, PRIMARY KEY (child_id, week_key))
battles         (id, account_id, child_id, state_json, created_at, finished_at)
recap_sends     (account_id, week_key, claimed_at, sent_at, PRIMARY KEY (account_id, week_key))
```

Notes:

- `rounds.id` is client-generated and `PRIMARY KEY`, so an iPad retry of
  the finish call is a duplicate-key that returns the original reward.
  Lesson applied: *an idempotency key needs a UNIQUE index, not just a
  pre-check*.
- `children.deleted_at` plus `ON DELETE CASCADE` from every child-keyed
  table, written in the same migration that creates the FK. Lesson applied:
  *a new FK column joins the referenced entity's delete path in the same
  commit*.
- `week_key` and `due_on` are local calendar dates in the account's
  timezone, stored as `YYYY-MM-DD` text. Lesson applied: *order "days" by
  the local date, not by local midnight's UTC instant*.
- `recap_sends` is the atomic claim for the email job (principle:
  *atomic-claims*). Created now so the recap PR does not touch the schema
  of anything else.

---

## 3. Server API surface

Every new endpoint uses the existing `decodeJSON` (cap, unknown fields,
trailing data). Lesson applied: *`json.Decoder.Decode` once is not strict
body parsing*; the cap per endpoint is listed.

| Route | Cap | Purpose |
|---|---|---|
| `POST /auth/magic` | 1 KiB | Send login link. Always 200, never reveals whether the email exists. |
| `GET /auth/verify?t=` | | Consume token (single use, 15 min), create session, redirect to `/family`. |
| `POST /auth/logout` | | Delete session row. |
| `GET /family` | | Parent dashboard: children, index as mastery map, plan status. Server-rendered. |
| `POST /api/children` | 1 KiB | Create nickname + grade. Nickname is trimmed, 1..24 chars, letters/digits/spaces only. |
| `PATCH /api/children/{id}` / `DELETE` | 1 KiB | Rename, change grade, soft-delete. |
| `POST /api/children/{id}/select` | | Sets `sessions.active_child_id`. |
| `POST /api/round/start` | 2 KiB | `{round_id, kind, skill_id, grade, count}`. For spelling, server picks the words (same three strategies as `selectSpellingWords`, moved to Go) and returns them; for math it wraps the existing `/api/sheet`. |
| `POST /api/round/finish` | 16 KiB | Math: `{round_id, answers}`. Spelling: `{round_id, guesses: [[letters...], ...]}`. Server grades, writes `rounds` + `round_items`, updates `skill_progress`, computes rewards, returns the reward payload. Idempotent on `round_id`. |
| `GET /api/kata/index` | | Roster merged with `creature_state` for the active child. |
| `POST /api/battle/start` | 1 KiB | Requires an unspent battle credit from a finished round. |
| `POST /api/battle/turn` | 2 KiB | `{battle_id, answer}`. Server grades, applies damage, returns new state. |
| `GET /api/paint/page/{id}` / `POST /api/paint/fill` | 4 KiB | Math key pages. |
| `GET /api/mosaic/week` | | Current week's cell count and image id. |

Without a session, every `/api/round/*` call behaves exactly like today's
`/api/sheet` + `/api/grade` for math and is not needed for spelling; the
client keeps its current code path. That is how the free tier stays
untouched.

Reward payload returned by `/api/round/finish` (the only place rewards are
computed):

```json
{
  "score": {"correct": 8, "total": 10},
  "creature": {"id": "g2-vowel-teams", "newly_seen": false, "fills_added": 6,
               "fills": 14, "regions": 20, "caught": false, "evolved": false},
  "mosaic": {"cells_added": 8, "cells": 131, "total": 400},
  "battle_credit": true,
  "review": {"words_due": 3}
}
```

Reward arithmetic (server-side, one function, table-tested):

- `fills_added = correct * weight`, where weight is 1 for the child's own
  grade, 2 for a grade above, and 2 for any item the child has missed
  before (`round_items` history). Grades below the child's grade earn 0
  fills but still count for the mosaic. This blocks easy-mode farming
  without punishing review.
- `mosaic cells_added = correct`, uncapped, per local week.
- `battle_credit` is granted once per finished round with `total >= 5`.

---

## 4. Mastery model (drives evolution and "words you keep missing")

Leitner boxes per `(child, skill)`:

- Box 0..4, review intervals `[1, 3, 7, 14, 30]` days.
- A finished round in the skill with `correct/total >= 0.8` on or after
  `due_on` moves box up one and sets `due_on = today + interval[box]`.
  Below 0.8 moves it down one (floor 0).
- `reviews_ok` counts successful reviews on or after `due_on`.
  `first_ok_on` is the local date of the first one.
- **Evolved** when `box >= 3` and `reviews_ok >= 2` and
  `today - first_ok_on >= 14 days`. Cannot be rushed by playing more today.
- "Words you keep missing" is a query over `round_items`: items with two or
  more `correct = 0` in the last 30 days for this child, excluding items
  that were correct in their most recent appearance.

---

## 5. Creature roster and index

Roster is static data, `internal/curriculum/kata.json`, embedded in Go and generated
into `static/kata-roster.js` like the words file. One creature per skill
per grade:

- 26 spelling focuses (ids already exist: `g1-short-vowels` ... `g5-...`)
- 5 sight-word entries (`sight-g1` ... `sight-g5`)
- 20 math entries (`math-addsub-g1` ... `math-frac-g5`)

51 creatures. Each entry: `id`, `skill_id`, `grade`, `name`, `hint_vague`,
`hint_specific`, `seed`, `regions` (20 for grade 1-2, 24 for 3-4, 30 for
5). Names are a content task (51 original names, nothing Pokémon-adjacent);
`hint_specific` is derived from the existing skill `label` and grade so it
never drifts from the curriculum. A roster test asserts every skill in
`SPELLING_SKILLS` and every math op has exactly one creature per grade, and
that no hint contains any word from that skill's bank. Lesson applied
(skilldojo, Used 0): *content that gets masked must be validated for
exactly one occurrence*; here the rule is "zero occurrences in hints".

Index states per child, computed server-side from `creature_state` and
`skill_progress`:

| State | Rule | Shown as |
|---|---|---|
| unknown | no rounds in skill | outline silhouette, `hint_vague` |
| seen | at least one round started | outline, `hint_specific`, Train here |
| caught | `fills >= regions` | full color, Train here |
| evolved | mastery rule in section 4 | evolved form, badge |

Index UI: one grid per grade with a completion bar, entries beyond the
child's grade shown "up the mountain" and playable, plus one personal entry
"Hiding in the words you keep missing" whose Train here starts a review
round of exactly those items. "Train here" sets `subject`, `spellingFocus`
or `ops`, and `grade` on the existing setup state and calls
`startTraining()`; no new game path.

Procedural creatures: `static/kata-svg.js`, a pure function
`creatureSVG(seed, fills, evolved) -> string`. Parts: 5 bodies, 4 eye sets,
4 crests, 3 patterns, 6 palettes keyed by grade. Every part is a region with
a small number label; regions fill in a fixed order as `fills` grows;
unfilled regions render as a light outline. The creature is a single
`role="img"` with a computed label ("Vowel Team kata, 14 of 20 regions
colored"). Lesson applied: *a composite visualization is a group of
labelled cells, not one image* applies to the pages and mosaic, and the
companion rule that a procedural creature is one image with one label. The
generator is pure and gets a snapshot test over a handful of seeds.

---

## 6. Color by number

### 6a. Creature reveal (ships with the index)

Already covered: the creature's regions are the color-by-number page.
Nothing extra to build beyond the reward payload animating `fills_added`
regions at session end.

### 6b. Math key pages

- A page is an SVG with `N` regions and a legend mapping answers to colors.
  Page art for v1 is geometric (tiles, mandalas, simple scenes built from
  primitives) generated from a seed, so there is no illustration cost.
- `GET /api/paint/page/{id}` generates `N` problems via
  `sheet.Generate(ops, grade, N)` and stores the sheet exactly like today,
  keyed by the page. Each region carries a problem prompt.
- `POST /api/paint/fill` `{page_id, idx, answer}` grades one problem
  against the stored sheet, and on success sets that bit in
  `page_state.filled_mask` and fills every region whose stored answer is
  equal. Wrong answers are not recorded against mastery (this mode is calm
  by design) but are counted in `rounds` for the recap.
- Page size is one session: 12 regions at grade 1, up to 24 at grade 5.
- Accessibility: regions are `role="group"` with per-region labels
  ("region 7, 6 times 8, unfilled"); the palette is a set of
  `aria-pressed` buttons. Lesson applied: *toggle-button UIs need
  aria-pressed / role=alert from the start* (skilldojo, Used 1).

### 6c. Weekly mosaic

- 20 by 20 grid, 400 cells, one image per week chosen from a small set of
  hand-made pixel arts stored as 400-char palette-index strings in
  `internal/curriculum/mosaics.json`. Cell reveal order is a seeded shuffle of the
  week key, so the picture emerges scattered rather than top-down.
- Storage is one integer per child per week (`mosaic_weeks.cells`).
  Rendering is a CSS grid of 400 divs in a `role="group"` labelled
  "Week mosaic, 131 of 400 cells".
- The recap PR renders the same data server-side to a PNG with
  `image/png` from the standard library, so the finished mosaic can go in
  the email with no dependency.

### 6d. Cooldown painting

After a lost round (six lanterns out), the results view offers a
free-form palette on the current creature's outline. No questions, no
state saved. Client-only, so it lands in whichever PR touches the results
view first.

---

## 7. Battles (last, and gated on the pilot)

- `POST /api/battle/start` spends the round's battle credit, picks an
  opponent creature from the same grade, and stores
  `{child_creature, opponent, hp, turn, skill_id}` in `battles.state_json`.
- Each turn the server generates one item from the child's creature's
  skill (math via `sheet.Generate(…, 1)`, spelling as "fill the missing
  letters" from the word bank) and returns it. The client posts the answer;
  the server grades and applies damage. Streak of three charges a special.
  No timers anywhere.
- Opponents faint and rest. Losing costs nothing but the battle.
- Battle state is server-side, so a refresh resumes. Client continuations
  (animations, audio) capture the turn object and bail if it is stale.
  Lesson applied (skilldojo, Used 0): *a rejected media/async callback
  outlives the state it was written for*.
- Build this only if the two-week pilot (section 10) shows the index and
  reveals pulling practice toward avoided skills.

---

## 8. PR sequence

Each PR is shippable and leaves the free app unchanged. Risk tier per
the ship flow; Aikido runs on PR 2, 3 and 6 (auth, input parsing, deps).

| PR | Scope | Tier | Key tests |
|---|---|---|---|
| **1. Curriculum as data** | `internal/curriculum/spelling.json` as source; script generates `static/words.js`; Go embeds JSON; `internal/curriculum` package with `Words(grade)`, `Skill(id)`, `Replay(word, guesses) (won, mistakes)`; CI sync check | 1 | Replay table test mirrors the JS win/lose tests; generated-file sync check |
| **2. Accounts and deploy** | `modernc.org/sqlite`, migrations, `accounts/login_tokens/sessions/children`, magic link via Resend, `/family` dashboard, child CRUD, `fly.toml` + volume + Litestream, secrets refuse empty | 2 + Aikido | Token single-use and expiry; session cookie flags; tenant fence test that child B under account A is unreachable from account B; nickname validation; Playwright smoke |
| **3. Rounds and mastery** | `rounds/round_items/skill_progress`, `/api/round/start|finish`, spelling replay grading, Leitner update, reward arithmetic (returns fills into a void until PR 4), client posts guess sequences when logged in | 2 + Aikido | Idempotent finish; reward table (own grade, grade above, previously missed, grade below); Leitner transitions incl. the 14-day floor; free-tier path untouched (no cookie means old endpoints, asserted in Go test) |
| **4. Kata index and reveals** | Roster JSON + generator, `creature_state`, `/api/kata/index`, `kata.js` + `kata-svg.js`, index view, Train here, session-end reveal animation, parent mastery map on `/family` | 2 | Roster coverage test; hint leak test; SVG snapshot; state transitions; Playwright renders index and results reveal |
| **5. Color by number** | Math key pages, `page_state`, mosaic data + `mosaic_weeks`, `paint.js`, cooldown painting | 2 | Fill grading and equal-answer propagation; mosaic week keys across a timezone change; a11y labels present |
| **6. Battles** | `battles`, start/turn endpoints, battle view | 2 + Aikido | State machine table test; stale-turn guard; credit spent exactly once |
| 7. Weekly recap email | claim row, cron, Resend template, mosaic PNG | later | |
| 8. Stripe | Checkout, portal, webhook, `plan` transitions | later | |

Rough effort, in focused sessions: PR 1 one, PR 2 three, PR 3 two, PR 4
three (one of them is content: names, hints, palettes), PR 5 two, PR 6 two.

---

## 9. Client changes in detail

- `dojo()` becomes `Object.assign({}, core(), kataMixin(), paintMixin())`.
  State added to core: `session` (`null` or `{child: {id, nickname, grade}}`),
  `roundId`, `guessLog` (per word, the ordered guesses), `reward`.
- `startSpelling`/`startSheet` generate `roundId` (`crypto.randomUUID()`)
  and, when `session` is set, call `/api/round/start` instead of picking
  words locally. `guessLetter` and `guessWholeWord` append to `guessLog`.
  The two session-complete hooks call `/api/round/finish` and store
  `reward`; the results views render it. Without a session every path is
  byte-for-byte today's behaviour.
- New views: `"kata-index"`, `"paint-page"`, `"battle"`. Setup gains a
  small "who's training" strip when logged in.
- Nickname is seeded into Alpine through a `data-nickname` attribute on the
  root, never interpolated into `x-data`. Lesson applied: *free text seeded
  into an Alpine x-data expression is HTML-escaped, not JS-escaped*.
- Every purchase, settings and account surface lives under `/family`
  (server-rendered, parent only). The kid-facing app shows nickname, index,
  and rewards, nothing about money.

---

## 10. Pilot and success criteria

After PR 4 ships to skilldojo.io with one account and one child profile:

- Two weeks of normal use.
- Watch two numbers from `rounds`: share of rounds in skills that were
  previously untouched (does the index pull her toward avoided skills), and
  rounds per week in plain mode with no reward on offer compared with the
  two weeks before (does intrinsic play survive).
- Build PR 6 only if the first rises and the second does not collapse.

---

## 11. Decisions to confirm before PR 1

1. Migrations at boot versus a manual `migrate` subcommand.
2. Curriculum moves to JSON with generated `words.js` (the alternative,
   parsing JS in Go, is worse).
3. Games require a parent login. Free tier gains nothing and loses nothing.
4. Leitner intervals and the 14-day evolution floor as written.
5. Playwright joins devDependencies for the render gate.
