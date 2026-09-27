-- Color by number: math key pages and the weekly mosaic.
CREATE TABLE page_state (
    account_id  INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    child_id    INTEGER NOT NULL REFERENCES children(id) ON DELETE CASCADE,
    page_id     TEXT NOT NULL,
    sheet_id    TEXT NOT NULL,
    grade       INTEGER NOT NULL CHECK (grade BETWEEN 1 AND 5),
    ops         TEXT NOT NULL,
    seed        INTEGER NOT NULL,
    regions     INTEGER NOT NULL,
    filled_mask INTEGER NOT NULL DEFAULT 0,
    attempts    INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL,
    PRIMARY KEY (child_id, page_id)
);

-- One row per child per local week; cells is the count of revealed cells.
CREATE TABLE mosaic_weeks (
    account_id INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    child_id   INTEGER NOT NULL REFERENCES children(id) ON DELETE CASCADE,
    week_key   TEXT NOT NULL,
    image_id   TEXT NOT NULL,
    cells      INTEGER NOT NULL DEFAULT 0,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (child_id, week_key)
);
