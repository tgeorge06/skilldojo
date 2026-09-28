package account

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/tgeorge06/skilldojo/internal/db"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	d, err := db.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	if _, err := db.Migrate(context.Background(), d); err != nil {
		t.Fatal(err)
	}
	return New(d)
}

var now = time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)

func TestNormalizeEmail(t *testing.T) {
	good := map[string]string{" Parent@Example.com ": "parent@example.com", "a.b+c@d.io": "a.b+c@d.io"}
	for in, want := range good {
		if got, err := NormalizeEmail(in); err != nil || got != want {
			t.Errorf("NormalizeEmail(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "nope", "a@", "@b.com", "Name <a@b.com>", "a@b.com, c@d.com", "a b@c.com"} {
		if _, err := NormalizeEmail(bad); err == nil {
			t.Errorf("NormalizeEmail(%q) accepted", bad)
		}
	}
}

func TestValidateNickname(t *testing.T) {
	if got, err := ValidateNickname("  Nova   Bee "); err != nil || got != "Nova Bee" {
		t.Fatalf("got %q, %v", got, err)
	}
	if got, err := ValidateNickname("Émilie O'Neil-Rose"); err != nil || got != "Émilie O'Neil-Rose" {
		t.Fatalf("got %q, %v", got, err)
	}
	if got, err := ValidateNickname("a\nb"); err != nil || got != "a b" {
		t.Fatalf("newline should collapse to a space, got %q, %v", got, err)
	}
	for _, bad := range []string{"", "   ", "<script>", "nova@home", "x&y", "1234567890123456789012345"} {
		if _, err := ValidateNickname(bad); err == nil {
			t.Errorf("ValidateNickname(%q) accepted", bad)
		}
	}
}

func TestValidateTimezone(t *testing.T) {
	if got := ValidateTimezone("America/New_York"); got != "America/New_York" {
		t.Fatal(got)
	}
	for _, bad := range []string{"", "Mars/Olympus", "../../etc/passwd"} {
		if got := ValidateTimezone(bad); got != "UTC" {
			t.Errorf("ValidateTimezone(%q) = %q", bad, got)
		}
	}
}

func TestLoginTokenLifecycle(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	plain, err := s.CreateLoginToken(ctx, "Parent@Example.com", "America/Chicago", now)
	if err != nil || len(plain) < 40 {
		t.Fatalf("token %q, %v", plain, err)
	}
	// Requesting a link creates no account.
	if _, err := s.AccountByID(ctx, 1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("account created before redemption: %v", err)
	}
	if email, err := s.PeekLoginToken(ctx, plain, now); err != nil || email != "parent@example.com" {
		t.Fatalf("peek = %q, %v", email, err)
	}
	if _, _, err := s.RedeemLoginToken(ctx, plain, now.Add(LoginTokenTTL+time.Second)); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expired token accepted: %v", err)
	}
	sess, acct, err := s.RedeemLoginToken(ctx, plain, now.Add(time.Minute))
	if err != nil || acct.Email != "parent@example.com" || acct.Timezone != "America/Chicago" || acct.Plan != "trial" {
		t.Fatalf("redeem = %+v, %v", acct, err)
	}
	if got, err := s.SessionByToken(ctx, sess, now.Add(time.Minute)); err != nil || got.AccountID != acct.ID {
		t.Fatalf("session after redeem = %+v, %v", got, err)
	}
	if _, _, err := s.RedeemLoginToken(ctx, plain, now.Add(time.Minute)); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("token reusable: %v", err)
	}
	if _, err := s.PeekLoginToken(ctx, plain, now.Add(time.Minute)); !errors.Is(err, ErrInvalidToken) {
		t.Fatal("peek of used token succeeded")
	}
	for _, bad := range []string{"", "nope", plain + "x"} {
		if _, _, err := s.RedeemLoginToken(ctx, bad, now); !errors.Is(err, ErrInvalidToken) {
			t.Errorf("RedeemLoginToken(%q) = %v", bad, err)
		}
	}

	// A second sign-in for the same address reuses the account and refreshes
	// the timezone, because ownership is proven by redemption.
	plain2, _ := s.CreateLoginToken(ctx, "parent@example.com", "Europe/Paris", now)
	_, again, err := s.RedeemLoginToken(ctx, plain2, now.Add(time.Minute))
	if err != nil || again.ID != acct.ID || again.Timezone != "Europe/Paris" {
		t.Fatalf("second redeem = %+v, %v", again, err)
	}
	// An unredeemed request with a bogus timezone changes nothing.
	_, _ = s.CreateLoginToken(ctx, "parent@example.com", "Mars/Olympus", now)
	got, _ := s.AccountByID(ctx, acct.ID)
	if got.Timezone != "Europe/Paris" {
		t.Fatalf("unredeemed request altered timezone: %q", got.Timezone)
	}
}

// ensureAccount is the test shorthand for a redeemed sign-in.
func ensureAccount(t *testing.T, s *Store, email string) Account {
	t.Helper()
	plain, err := s.CreateLoginToken(context.Background(), email, "UTC", now)
	if err != nil {
		t.Fatal(err)
	}
	_, acct, err := s.RedeemLoginToken(context.Background(), plain, now)
	if err != nil {
		t.Fatal(err)
	}
	return acct
}

func TestSessionsAndChildrenAreFencedByAccount(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	a := ensureAccount(t, s, "a@example.com")
	b := ensureAccount(t, s, "b@example.com")

	tokA, _ := s.CreateSession(ctx, a.ID, now)
	sessA, err := s.SessionByToken(ctx, tokA, now)
	if err != nil || sessA.AccountID != a.ID || sessA.ActiveChildID != 0 {
		t.Fatalf("session = %+v, %v", sessA, err)
	}
	if _, err := s.SessionByToken(ctx, tokA, now.Add(SessionTTL+time.Second)); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expired session accepted: %v", err)
	}

	childA, err := s.CreateChild(ctx, a.ID, "Nova", 2, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateChild(ctx, a.ID, "Nova", 9, now); err == nil {
		t.Fatal("grade 9 accepted")
	}

	// Account B cannot see, edit, delete, or select A's child.
	if _, err := s.Child(ctx, b.ID, childA.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-account Child = %v", err)
	}
	if err := s.UpdateChild(ctx, b.ID, childA.ID, "Hacked", 1, 10); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-account UpdateChild = %v", err)
	}
	if err := s.DeleteChild(ctx, b.ID, childA.ID, now); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-account DeleteChild = %v", err)
	}
	tokB, _ := s.CreateSession(ctx, b.ID, now)
	sessB, _ := s.SessionByToken(ctx, tokB, now)
	if err := s.SetActiveChild(ctx, sessB.ID, b.ID, childA.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-account SetActiveChild = %v", err)
	}
	// Nor can B use A's session id with B's account.
	if err := s.SetActiveChild(ctx, sessA.ID, b.ID, childA.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("session id with wrong account = %v", err)
	}
	if kids, _ := s.Children(ctx, b.ID); len(kids) != 0 {
		t.Fatalf("B sees %d children", len(kids))
	}

	// The owner can.
	if err := s.SetActiveChild(ctx, sessA.ID, a.ID, childA.ID); err != nil {
		t.Fatal(err)
	}
	sessA, _ = s.SessionByToken(ctx, tokA, now)
	if sessA.ActiveChildID != childA.ID {
		t.Fatalf("active child = %d", sessA.ActiveChildID)
	}
	if err := s.UpdateChild(ctx, a.ID, childA.ID, "Nova B", 3, 7); err == nil {
		t.Fatal("round length 7 should be rejected")
	}
	if err := s.UpdateChild(ctx, a.ID, childA.ID, "Nova B", 3, 20); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Child(ctx, a.ID, childA.ID)
	if got.Nickname != "Nova B" || got.Grade != 3 || got.RoundLen != 20 {
		t.Fatalf("after update: %+v", got)
	}

	// Deleting clears the selection and hides the row.
	if err := s.DeleteChild(ctx, a.ID, childA.ID, now); err != nil {
		t.Fatal(err)
	}
	sessA, _ = s.SessionByToken(ctx, tokA, now)
	if sessA.ActiveChildID != 0 {
		t.Fatalf("active child survived delete: %d", sessA.ActiveChildID)
	}
	if _, err := s.Child(ctx, a.ID, childA.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted child visible: %v", err)
	}
	if err := s.DeleteSession(ctx, tokA); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SessionByToken(ctx, tokA, now); !errors.Is(err, ErrNotFound) {
		t.Fatal("session survived logout")
	}
}

func TestPurgeExpired(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	a := ensureAccount(t, s, "a@example.com")
	_, _ = s.CreateLoginToken(ctx, "a@example.com", "UTC", now)
	tok, _ := s.CreateSession(ctx, a.ID, now)
	if err := s.PurgeExpired(ctx, now.Add(SessionTTL+time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SessionByToken(ctx, tok, now); !errors.Is(err, ErrNotFound) {
		t.Fatal("expired session not purged")
	}
}
