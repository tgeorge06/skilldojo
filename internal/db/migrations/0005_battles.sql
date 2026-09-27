-- Turn-based quiz battles. A credit is earned by a finished round of five
-- attempted items and spent to start one battle. State lives server-side
-- so a refresh resumes and answers are never in the client.
CREATE TABLE battle_credits (
    round_id   TEXT PRIMARY KEY REFERENCES rounds(id) ON DELETE CASCADE,
    account_id INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    child_id   INTEGER NOT NULL REFERENCES children(id) ON DELETE CASCADE,
    earned_at  TEXT NOT NULL,
    spent_at   TEXT,
    battle_id  TEXT
);
CREATE INDEX battle_credits_child ON battle_credits(child_id, spent_at);

CREATE TABLE battles (
    id          TEXT PRIMARY KEY,
    account_id  INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    child_id    INTEGER NOT NULL REFERENCES children(id) ON DELETE CASCADE,
    state_json  TEXT NOT NULL,
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL,
    finished_at TEXT
);
CREATE INDEX battles_child ON battles(child_id, finished_at);
