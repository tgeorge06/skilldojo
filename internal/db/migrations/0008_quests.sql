-- Daily quests: three per child per local day, picked by the server from
-- review words, practice-test weak areas, and a rotation. A quest is done
-- when any matching round finishes; finishing all three reveals a kata.
CREATE TABLE quests (
    id          INTEGER PRIMARY KEY,
    account_id  INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    child_id    INTEGER NOT NULL REFERENCES children(id) ON DELETE CASCADE,
    day         TEXT NOT NULL,            -- YYYY-MM-DD in the family's timezone
    idx         INTEGER NOT NULL,         -- 0, 1, 2
    kind        TEXT NOT NULL CHECK (kind IN ('math', 'spelling')),
    focus       TEXT NOT NULL,            -- math op id or spelling focus
    tbl         INTEGER NOT NULL DEFAULT 0,
    grade       INTEGER NOT NULL,
    count       INTEGER NOT NULL,
    label       TEXT NOT NULL,
    reason      TEXT NOT NULL,
    round_id    TEXT,
    done_at     TEXT,
    reveal_json TEXT,                     -- set on the quest that finished the day
    UNIQUE (child_id, day, idx)
);
CREATE INDEX quests_child_day ON quests(child_id, day);
