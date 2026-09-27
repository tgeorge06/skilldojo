-- Parent accounts, magic-link tokens, sessions, and nickname-only children.
-- Child rows never hold PII beyond a nickname; the parent owns the account.

CREATE TABLE accounts (
    id            INTEGER PRIMARY KEY,
    email         TEXT NOT NULL UNIQUE,
    timezone      TEXT NOT NULL DEFAULT 'UTC',
    plan          TEXT NOT NULL DEFAULT 'trial',
    trial_ends_at TEXT,
    created_at    TEXT NOT NULL
);

-- A login token is a challenge for an email address. The account row is
-- created (or its timezone refreshed) only when the token is redeemed, so an
-- unverified request can never create or alter an account.
CREATE TABLE login_tokens (
    id         INTEGER PRIMARY KEY,
    email      TEXT NOT NULL,
    timezone   TEXT NOT NULL DEFAULT 'UTC',
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TEXT NOT NULL,
    used_at    TEXT
);
CREATE INDEX login_tokens_expires ON login_tokens(expires_at);

CREATE TABLE children (
    id         INTEGER PRIMARY KEY,
    account_id INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    nickname   TEXT NOT NULL,
    grade      INTEGER NOT NULL CHECK (grade BETWEEN 1 AND 5),
    created_at TEXT NOT NULL,
    deleted_at TEXT
);
CREATE INDEX children_account ON children(account_id);

CREATE TABLE sessions (
    id              INTEGER PRIMARY KEY,
    account_id      INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    token_hash      TEXT NOT NULL UNIQUE,
    active_child_id INTEGER REFERENCES children(id) ON DELETE SET NULL,
    created_at      TEXT NOT NULL,
    expires_at      TEXT NOT NULL
);
CREATE INDEX sessions_account ON sessions(account_id);
