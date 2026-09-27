package ost

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/tgeorge06/skilldojo/internal/account"
	"github.com/tgeorge06/skilldojo/internal/db"
)

func TestAnalyzeFlagsWeakCategoriesAndWeightsRecentAttempts(t *testing.T) {
	// Newest first, as History returns. Fractions were weak, then improved.
	mk := func(fr, md int) Summary {
		return Summary{Finished: true, Percent: (fr + md) * 5, Categories: []Tally{
			{Name: "Fractions", Right: fr, Total: 10, Percent: fr * 10},
			{Name: "Multiplication and Division", Right: md, Total: 10, Percent: md * 10},
		}, Standards: []Tally{{Name: "4.NF.1", Right: fr / 2, Total: 5}, {Name: "4.NF.3", Right: fr - fr/2, Total: 5}, {Name: "4.OA.1", Right: md, Total: 10}}}
	}
	an := Analyze([]Summary{mk(10, 4), mk(6, 4), mk(2, 5), Summary{Finished: false}}, 4)
	if an.Finished != 3 || an.Best != 70 || len(an.Trend) != 3 || an.Trend[0] != 35 {
		t.Fatalf("summary: %+v", an)
	}
	if len(an.Focus) != 1 || an.Focus[0].Category != "Multiplication and Division" {
		t.Fatalf("focus should be tables only (fractions improved): %+v", an.Focus)
	}
	if an.Focus[0].Practice.Path != "/#math/tables" || len(an.Focus[0].Standards) != 1 || an.Focus[0].Standards[0].Name != "4.OA.1" {
		t.Fatalf("focus detail: %+v", an.Focus[0])
	}
	if len(an.Strong) != 1 || an.Strong[0] != "Fractions" {
		t.Fatalf("strong: %v", an.Strong)
	}
	for _, c := range an.Categories {
		if c.Category == "Fractions" && (len(c.Points) != 3 || c.Points[2] != 100) {
			t.Fatalf("fraction trend: %+v", c)
		}
	}
	if empty := Analyze(nil, 3); empty.Finished != 0 || empty.Latest != nil {
		t.Fatal("empty history should be empty analysis")
	}
}

func TestStoreRoundTrip(t *testing.T) {
	d, err := db.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	ctx := context.Background()
	if _, err := db.Migrate(ctx, d); err != nil {
		t.Fatal(err)
	}
	accounts := account.New(d)
	now := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	_, acct, err := accounts.RedeemLoginToken(ctx, mustToken(t, accounts, now), now)
	if err != nil {
		t.Fatal(err)
	}
	kid, err := accounts.CreateChild(ctx, acct.ID, "Nova", 4, now)
	if err != nil {
		t.Fatal(err)
	}
	child := Child{AccountID: acct.ID, ChildID: kid.ID, Grade: 4}
	s := New(d)

	a, err := s.Start(ctx, child, 4, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Items) != TestLength || a.Report != nil {
		t.Fatalf("fresh attempt: %d items, report %v", len(a.Items), a.Report)
	}
	// Answer with the key, read from storage, to prove grading uses the stored items.
	rw, err := s.load(ctx, child, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	for i, it := range rw.items {
		if i%4 == 0 { // skip a quarter
			continue
		}
		ans := Answer{Choices: it.Answer}
		if it.Type == TypeNumber {
			ans = Answer{Text: it.Numeric}
		}
		if err := s.SaveAnswer(ctx, child, a.ID, it.ID, ans, now); err != nil {
			t.Fatalf("save %s: %v", it.ID, err)
		}
	}
	// Clearing an answer removes it.
	if err := s.SaveAnswer(ctx, child, a.ID, rw.items[1].ID, Answer{}, now); err != nil {
		t.Fatal(err)
	}
	resumed, err := s.Start(ctx, child, 4, now.Add(time.Hour))
	if err != nil || resumed.ID != a.ID || len(resumed.Answers) != 29 {
		t.Fatalf("resume: %v id %s answers %d", err, resumed.ID, len(resumed.Answers))
	}
	// Another grade is a separate attempt in progress.
	g5, err := s.Start(ctx, child, 5, now)
	if err != nil || g5.ID == a.ID || g5.Grade != 5 {
		t.Fatalf("grade 5 attempt: %v %+v", err, g5)
	}

	done, err := s.Submit(ctx, child, a.ID, now.Add(2*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if done.Report == nil || done.Report.Score != 29 || done.Report.Percent != 72 || done.Report.Level != "Accomplished" {
		t.Fatalf("report: %+v", done.Report)
	}
	if len(done.Report.Categories) != 3 || len(done.Report.Items) != TestLength {
		t.Fatalf("report shape: %d cats %d items", len(done.Report.Categories), len(done.Report.Items))
	}
	if err := s.SaveAnswer(ctx, child, a.ID, rw.items[0].ID, Answer{Text: "1"}, now); err != ErrFinished {
		t.Fatalf("save after submit: %v", err)
	}
	hist, err := s.History(ctx, child, 10)
	if err != nil || len(hist) != 2 || hist[0].Finished || !hist[1].Finished || hist[1].Percent != 72 {
		t.Fatalf("history: %v %+v", err, hist)
	}
	if _, err := s.Attempt(ctx, Child{AccountID: acct.ID + 1, ChildID: kid.ID}, a.ID); err != ErrNotFound {
		t.Fatalf("other account: %v", err)
	}
}

func mustToken(t *testing.T, s *account.Store, now time.Time) string {
	t.Helper()
	tok, err := s.CreateLoginToken(context.Background(), "p@example.com", "America/New_York", now)
	if err != nil {
		t.Fatal(err)
	}
	return tok
}
