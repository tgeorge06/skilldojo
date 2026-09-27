-- Creature (kata) state per child. The roster itself is static data; only
-- discovery state lives here. fills counts colored regions toward "caught".
CREATE TABLE creature_state (
    account_id  INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    child_id    INTEGER NOT NULL REFERENCES children(id) ON DELETE CASCADE,
    creature_id TEXT NOT NULL,
    state       TEXT NOT NULL CHECK (state IN ('seen', 'caught', 'evolved')),
    fills       INTEGER NOT NULL DEFAULT 0,
    seen_at     TEXT NOT NULL,
    caught_at   TEXT,
    evolved_at  TEXT,
    updated_at  TEXT NOT NULL,
    PRIMARY KEY (child_id, creature_id)
);
