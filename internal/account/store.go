// Package account owns parent accounts, magic-link tokens, sessions, and
// nickname-only child profiles. Every method that touches a child takes the
// account id as well and puts both in the WHERE clause, so a caller can never
// reach another family's rows even with a guessed child id.
package account

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"
	"unicode"
)

// ErrNotFound is returned when a row does not exist or is not visible to the
// given account.
var ErrNotFound = errors.New("account: not found")

// ErrInvalidToken covers unknown, expired, and already-used tokens alike so a
// caller cannot distinguish them.
var ErrInvalidToken = errors.New("account: invalid token")

const (
	// LoginTokenTTL is how long a magic link stays valid.
	LoginTokenTTL = 15 * time.Minute
	// SessionTTL is how long a parent stays signed in.
	SessionTTL = 90 * 24 * time.Hour
	// MaxNickname is the nickname length cap in runes.
	MaxNickname = 24
	// MinGrade and MaxGrade bound a child's grade.
	MinGrade, MaxGrade = 1, 5
)

// Account is a parent.
type Account struct {
	ID        int64
	Email     string
	Timezone  string
	Plan      string
	CreatedAt time.Time
}

// Child is a nickname-only profile under an account.
type Child struct {
	ID        int64
	AccountID int64
	Nickname  string
	Grade     int
	CreatedAt time.Time
}

// Session is a signed-in parent, optionally with a selected child.
type Session struct {
	ID            int64
	AccountID     int64
	ActiveChildID int64 // 0 when none
	ExpiresAt     time.Time
}

// Store is the data access layer.
type Store struct {
	db *sql.DB
}

// New wraps an opened, migrated database.
func New(db *sql.DB) *Store { return &Store{db: db} }

// NormalizeEmail lower-cases and validates an address. It is deliberately
// strict about shape and lenient about domains.
func NormalizeEmail(raw string) (string, error) {
	e := strings.ToLower(strings.TrimSpace(raw))
	if e == "" || len(e) > 254 {
		return "", errors.New("account: email is required")
	}
	addr, err := mail.ParseAddress(e)
	if err != nil || addr.Address != e || !strings.Contains(e, "@") {
		return "", errors.New("account: email looks wrong")
	}
	return e, nil
}

// ValidateNickname trims, collapses spaces, and rejects anything but
// letters, digits, spaces, apostrophes, and hyphens. Returns the cleaned value.
func ValidateNickname(raw string) (string, error) {
	n := strings.Join(strings.Fields(strings.TrimSpace(raw)), " ")
	if n == "" {
		return "", errors.New("account: nickname is required")
	}
	if len([]rune(n)) > MaxNickname {
		return "", fmt.Errorf("account: nickname must be %d characters or fewer", MaxNickname)
	}
	for _, r := range n {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' || r == '\'' || r == '-') {
			return "", errors.New("account: nickname can use letters, numbers, spaces, apostrophes, and hyphens")
		}
	}
	return n, nil
}

// ValidateTimezone returns tz if it loads, else "UTC".
func ValidateTimezone(tz string) string {
	tz = strings.TrimSpace(tz)
	if tz == "" || len(tz) > 64 {
		return "UTC"
	}
	if _, err := time.LoadLocation(tz); err != nil {
		return "UTC"
	}
	return tz
}

func newToken() (plain, hash string, err error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", "", err
	}
	plain = base64.RawURLEncoding.EncodeToString(b[:])
	return plain, hashToken(plain), nil
}

func hashToken(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}

// tsLayout is fixed width so "expires_at > ?" compares chronologically in
// SQL; RFC3339Nano drops trailing zeros and breaks lexical order.
const tsLayout = "2006-01-02T15:04:05.000000000Z"

func ts(t time.Time) string { return t.UTC().Format(tsLayout) }

func parseTS(s string) time.Time {
	t, _ := time.Parse(time.RFC3339Nano, s)
	return t
}

// CreateLoginToken mints a single-use magic-link challenge for an email
// address and returns the plaintext to embed in the email. Only its hash is
// stored, and no account row exists until the token is redeemed.
func (s *Store) CreateLoginToken(ctx context.Context, email, timezone string, now time.Time) (string, error) {
	email, err := NormalizeEmail(email)
	if err != nil {
		return "", err
	}
	plain, hash, err := newToken()
	if err != nil {
		return "", err
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO login_tokens (email, timezone, token_hash, expires_at) VALUES (?, ?, ?, ?)`,
		email, ValidateTimezone(timezone), hash, ts(now.Add(LoginTokenTTL)))
	return plain, err
}

// PeekLoginToken returns the email a live token belongs to without
// consuming it, so a confirmation page can say who is about to sign in.
func (s *Store) PeekLoginToken(ctx context.Context, plain string, now time.Time) (string, error) {
	if plain == "" || len(plain) > 128 {
		return "", ErrInvalidToken
	}
	var email string
	err := s.db.QueryRowContext(ctx,
		`SELECT email FROM login_tokens WHERE token_hash = ? AND used_at IS NULL AND expires_at > ?`,
		hashToken(plain), ts(now)).Scan(&email)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrInvalidToken
	}
	return email, err
}

// RedeemLoginToken consumes a token, creates the account on first sight (or
// refreshes its timezone, now that mailbox ownership is proven), and starts
// a session, all in one transaction so a failure never burns the link
// without signing the parent in. Returns the session cookie value.
func (s *Store) RedeemLoginToken(ctx context.Context, plain string, now time.Time) (string, Account, error) {
	if plain == "" || len(plain) > 128 {
		return "", Account{}, ErrInvalidToken
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", Account{}, err
	}
	defer tx.Rollback()

	var email, timezone string
	err = tx.QueryRowContext(ctx,
		`UPDATE login_tokens SET used_at = ?
		 WHERE token_hash = ? AND used_at IS NULL AND expires_at > ?
		 RETURNING email, timezone`,
		ts(now), hashToken(plain), ts(now)).Scan(&email, &timezone)
	if errors.Is(err, sql.ErrNoRows) {
		return "", Account{}, ErrInvalidToken
	}
	if err != nil {
		return "", Account{}, err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO accounts (email, timezone, plan, created_at) VALUES (?, ?, 'trial', ?)
		 ON CONFLICT(email) DO UPDATE SET timezone = excluded.timezone`,
		email, timezone, ts(now)); err != nil {
		return "", Account{}, err
	}
	var acct Account
	var created string
	if err := tx.QueryRowContext(ctx,
		`SELECT id, email, timezone, plan, created_at FROM accounts WHERE email = ?`, email).
		Scan(&acct.ID, &acct.Email, &acct.Timezone, &acct.Plan, &created); err != nil {
		return "", Account{}, err
	}
	acct.CreatedAt = parseTS(created)

	sessionPlain, sessionHash, err := newToken()
	if err != nil {
		return "", Account{}, err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO sessions (account_id, token_hash, created_at, expires_at) VALUES (?, ?, ?, ?)`,
		acct.ID, sessionHash, ts(now), ts(now.Add(SessionTTL))); err != nil {
		return "", Account{}, err
	}
	if err := tx.Commit(); err != nil {
		return "", Account{}, err
	}
	return sessionPlain, acct, nil
}

// AccountByID loads one account.
func (s *Store) AccountByID(ctx context.Context, id int64) (Account, error) {
	var a Account
	var created string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, email, timezone, plan, created_at FROM accounts WHERE id = ?`, id).
		Scan(&a.ID, &a.Email, &a.Timezone, &a.Plan, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return Account{}, ErrNotFound
	}
	a.CreatedAt = parseTS(created)
	return a, err
}

// CreateSession starts a signed-in session for an existing account and
// returns the cookie value. Sign-in goes through RedeemLoginToken; this is
// for tests and future flows that already proved identity.
func (s *Store) CreateSession(ctx context.Context, accountID int64, now time.Time) (string, error) {
	plain, hash, err := newToken()
	if err != nil {
		return "", err
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO sessions (account_id, token_hash, created_at, expires_at) VALUES (?, ?, ?, ?)`,
		accountID, hash, ts(now), ts(now.Add(SessionTTL)))
	return plain, err
}

// SessionByToken resolves a cookie value to a live session.
func (s *Store) SessionByToken(ctx context.Context, plain string, now time.Time) (Session, error) {
	if plain == "" || len(plain) > 128 {
		return Session{}, ErrNotFound
	}
	var sess Session
	var child sql.NullInt64
	var expires string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, account_id, active_child_id, expires_at FROM sessions
		 WHERE token_hash = ? AND expires_at > ?`, hashToken(plain), ts(now)).
		Scan(&sess.ID, &sess.AccountID, &child, &expires)
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, ErrNotFound
	}
	if err != nil {
		return Session{}, err
	}
	sess.ActiveChildID = child.Int64
	sess.ExpiresAt = parseTS(expires)
	return sess, nil
}

// DeleteSession signs out one session.
func (s *Store) DeleteSession(ctx context.Context, plain string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = ?`, hashToken(plain))
	return err
}

// SetActiveChild selects which child is training. The child must belong to
// the session's account; otherwise ErrNotFound.
func (s *Store) SetActiveChild(ctx context.Context, sessionID, accountID, childID int64) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE sessions SET active_child_id = ?
		 WHERE id = ? AND account_id = ?
		   AND EXISTS (SELECT 1 FROM children WHERE id = ? AND account_id = ? AND deleted_at IS NULL)`,
		childID, sessionID, accountID, childID, accountID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// Children lists an account's live profiles, oldest first.
func (s *Store) Children(ctx context.Context, accountID int64) ([]Child, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, account_id, nickname, grade, created_at FROM children
		 WHERE account_id = ? AND deleted_at IS NULL ORDER BY id`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Child
	for rows.Next() {
		var c Child
		var created string
		if err := rows.Scan(&c.ID, &c.AccountID, &c.Nickname, &c.Grade, &created); err != nil {
			return nil, err
		}
		c.CreatedAt = parseTS(created)
		out = append(out, c)
	}
	return out, rows.Err()
}

// Child loads one live profile visible to the account.
func (s *Store) Child(ctx context.Context, accountID, childID int64) (Child, error) {
	var c Child
	var created string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, account_id, nickname, grade, created_at FROM children
		 WHERE id = ? AND account_id = ? AND deleted_at IS NULL`, childID, accountID).
		Scan(&c.ID, &c.AccountID, &c.Nickname, &c.Grade, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return Child{}, ErrNotFound
	}
	c.CreatedAt = parseTS(created)
	return c, err
}

// CreateChild adds a profile. Nickname is validated here, grade must be 1-5.
func (s *Store) CreateChild(ctx context.Context, accountID int64, nickname string, grade int, now time.Time) (Child, error) {
	nick, err := ValidateNickname(nickname)
	if err != nil {
		return Child{}, err
	}
	if grade < MinGrade || grade > MaxGrade {
		return Child{}, errors.New("account: grade must be between 1 and 5")
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO children (account_id, nickname, grade, created_at) VALUES (?, ?, ?, ?)`,
		accountID, nick, grade, ts(now))
	if err != nil {
		return Child{}, err
	}
	id, _ := res.LastInsertId()
	return Child{ID: id, AccountID: accountID, Nickname: nick, Grade: grade, CreatedAt: now}, nil
}

// UpdateChild renames or regrades a profile the account owns.
func (s *Store) UpdateChild(ctx context.Context, accountID, childID int64, nickname string, grade int) error {
	nick, err := ValidateNickname(nickname)
	if err != nil {
		return err
	}
	if grade < MinGrade || grade > MaxGrade {
		return errors.New("account: grade must be between 1 and 5")
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE children SET nickname = ?, grade = ? WHERE id = ? AND account_id = ? AND deleted_at IS NULL`,
		nick, grade, childID, accountID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteChild soft-deletes a profile and clears it from any session that
// had it selected.
func (s *Store) DeleteChild(ctx context.Context, accountID, childID int64, now time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx,
		`UPDATE children SET deleted_at = ? WHERE id = ? AND account_id = ? AND deleted_at IS NULL`,
		ts(now), childID, accountID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE sessions SET active_child_id = NULL WHERE active_child_id = ? AND account_id = ?`,
		childID, accountID); err != nil {
		return err
	}
	return tx.Commit()
}

// PurgeExpired removes dead tokens and sessions. Safe to call on a timer.
func (s *Store) PurgeExpired(ctx context.Context, now time.Time) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM login_tokens WHERE expires_at <= ? OR used_at IS NOT NULL`, ts(now.Add(-time.Hour))); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at <= ?`, ts(now))
	return err
}
