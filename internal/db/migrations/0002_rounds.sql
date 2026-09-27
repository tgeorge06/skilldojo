-- Practice rounds and per-skill mastery for signed-in children.
-- rounds.id is chosen by the client so a retried finish is a duplicate
-- key, not a double reward. Every table carries account_id so a query can
-- fence on it without joining.

CREATE TABLE rounds (
    id          TEXT PRIMARY KEY,
    account_id  INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    child_id    INTEGER NOT NULL REFERENCES children(id) ON DELETE CASCADE,
    kind        TEXT NOT NULL CHECK (kind IN ('math', 'spelling')),
    focus       TEXT NOT NULL,
    grade       INTEGER NOT NULL CHECK (grade BETWEEN 1 AND 5),
    child_grade INTEGER NOT NULL CHECK (child_grade BETWEEN 1 AND 5),
    sheet_id    TEXT,
    started_at  TEXT NOT NULL,
    finished_at TEXT,
    correct     INTEGER,
    total       INTEGER,
    result_json TEXT
);
CREATE INDEX rounds_child_finished ON rounds(child_id, finished_at);

CREATE TABLE round_items (
    round_id   TEXT NOT NULL REFERENCES rounds(id) ON DELETE CASCADE,
    idx        INTEGER NOT NULL,
    account_id INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    child_id   INTEGER NOT NULL REFERENCES children(id) ON DELETE CASCADE,
    item_key   TEXT NOT NULL,
    skill_id   TEXT NOT NULL,
    grade      INTEGER NOT NULL,
    correct    INTEGER,
    answered_at TEXT,
    PRIMARY KEY (round_id, idx)
);
CREATE INDEX round_items_child_key ON round_items(child_id, item_key, answered_at);

CREATE TABLE skill_progress (
    account_id   INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    child_id     INTEGER NOT NULL REFERENCES children(id) ON DELETE CASCADE,
    skill_id     TEXT NOT NULL,
    box          INTEGER NOT NULL DEFAULT 0,
    due_on       TEXT NOT NULL,
    reviews_ok   INTEGER NOT NULL DEFAULT 0,
    first_ok_on  TEXT,
    last_seen_at TEXT NOT NULL,
    rounds       INTEGER NOT NULL DEFAULT 0,
    evolved_on   TEXT,
    PRIMARY KEY (child_id, skill_id)
);
