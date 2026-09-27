-- Practice test attempts. Items (with keys) are stored with the attempt so
-- a saved test resumes exactly as it was, even after templates change.
-- Answers are saved as the child goes; finished_at marks a graded attempt.
CREATE TABLE ost_attempts (
    id           TEXT PRIMARY KEY,
    account_id   INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    child_id     INTEGER NOT NULL REFERENCES children(id) ON DELETE CASCADE,
    subject      TEXT NOT NULL,
    grade        INTEGER NOT NULL,
    seed         INTEGER NOT NULL,
    items_json   TEXT NOT NULL,
    answers_json TEXT NOT NULL DEFAULT '{}',
    started_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL,
    finished_at  TEXT,
    score        INTEGER NOT NULL DEFAULT 0,
    total        INTEGER NOT NULL DEFAULT 0,
    percent      INTEGER NOT NULL DEFAULT 0,
    level        TEXT NOT NULL DEFAULT '',
    report_json  TEXT NOT NULL DEFAULT '{}'
);
CREATE INDEX ost_attempts_child ON ost_attempts(child_id, finished_at, started_at);
