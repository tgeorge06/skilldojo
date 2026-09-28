package quest

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/tgeorge06/skilldojo/internal/account"
	"github.com/tgeorge06/skilldojo/internal/curriculum"
	"github.com/tgeorge06/skilldojo/internal/db"
	"github.com/tgeorge06/skilldojo/internal/kata"
	"github.com/tgeorge06/skilldojo/internal/ost"
	"github.com/tgeorge06/skilldojo/internal/progress"
	"github.com/tgeorge06/skilldojo/internal/sheet"
)

type fakeSources struct {
	missed  int
	history []ost.Summary
}

func (f fakeSources) MissedCount(context.Context, progress.Child, time.Time) (int, error) {
	return f.missed, nil
}
func (f fakeSources) History(context.Context, ost.Child, int) ([]ost.Summary, error) {
	return f.history, nil
}

type env struct {
	store *Store
	prog  *progress.Store
	child progress.Child
	ctx   context.Context
}

func newEnv(t *testing.T, src Sources) *env {
	t.Helper()
	d, err := db.Open(filepath.Join(t.TempDir(), "q.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	ctx := context.Background()
	if _, err := db.Migrate(ctx, d); err != nil {
		t.Fatal(err)
	}
	cur, err := curriculum.Load()
	if err != nil {
		t.Fatal(err)
	}
	roster, err := kata.Load(cur)
	if err != nil {
		t.Fatal(err)
	}
	accounts := account.New(d)
	now := time.Date(2026, 9, 28, 20, 0, 0, 0, time.UTC) // 4 pm in New York
	tok, _ := accounts.CreateLoginToken(ctx, "q@example.com", "America/New_York", now)
	_, acct, err := accounts.RedeemLoginToken(ctx, tok, now)
	if err != nil {
		t.Fatal(err)
	}
	kid, err := accounts.CreateChild(ctx, acct.ID, "Nova", 3, now)
	if err != nil {
		t.Fatal(err)
	}
	prog := progress.New(d, cur, sheet.NewStore())
	prog.AddSink(kata.New(d, roster))
	store := New(d, roster, src)
	prog.AddSink(store)
	return &env{store: store, prog: prog, ctx: ctx, child: progress.Child{AccountID: acct.ID, ChildID: kid.ID, Grade: 3, Timezone: "America/New_York", RoundLen: 10}}
}

func TestTodayPicksThreeAndIsStablePerDay(t *testing.T) {
	e := newEnv(t, fakeSources{missed: 4})
	now := time.Date(2026, 9, 28, 20, 0, 0, 0, time.UTC)
	d, err := e.store.Today(e.ctx, e.child, now)
	if err != nil {
		t.Fatal(err)
	}
	if d.Day != "2026-09-28" || len(d.Quests) != 3 || d.AllDone {
		t.Fatalf("day: %+v", d)
	}
	if d.Quests[0].Kind != "math" || d.Quests[1].Focus != "review" || d.Quests[1].Reason != "4 words are waiting to be rescued" {
		t.Fatalf("picks: %+v", d.Quests)
	}
	again, _ := e.store.Today(e.ctx, e.child, now.Add(2*time.Hour))
	if again.Quests[0].ID != d.Quests[0].ID {
		t.Fatal("the same local day must return the same quests")
	}
	// Just past midnight New York time is a new day with a new set.
	next, _ := e.store.Today(e.ctx, e.child, time.Date(2026, 9, 29, 4, 30, 0, 0, time.UTC))
	if next.Day != "2026-09-29" || next.Quests[0].ID == d.Quests[0].ID {
		t.Fatalf("next day: %+v", next)
	}
}

func TestWeakPracticeTestAreaDrivesTheMathQuest(t *testing.T) {
	hist := []ost.Summary{{Grade: 3, Finished: true, FinishedAt: time.Now(), Percent: 40, Categories: []ost.Tally{
		{Name: "Fractions", Right: 2, Total: 10, Percent: 20}, {Name: "Geometry", Right: 9, Total: 10, Percent: 90},
	}}}
	e := newEnv(t, fakeSources{history: hist})
	d, err := e.store.Today(e.ctx, e.child, time.Date(2026, 9, 28, 20, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if d.Quests[0].Focus != "frac" || d.Quests[0].Reason != "Your practice test says fractions needs work" {
		t.Fatalf("math quest: %+v", d.Quests[0])
	}
}

func TestRoundsCompleteQuestsAndTheThirdReveals(t *testing.T) {
	e := newEnv(t, fakeSources{})
	now := time.Date(2026, 9, 28, 20, 0, 0, 0, time.UTC)
	d, _ := e.store.Today(e.ctx, e.child, now)
	// Finish rounds matching each quest in turn.
	finish := func(id string, q Quest, at time.Time) progress.FinishResponse {
		t.Helper()
		req := progress.StartRequest{RoundID: id, Kind: q.Kind, Grade: 3, Count: q.Count}
		if q.Kind == "math" {
			req.Ops = []string{q.Focus}
			if q.Focus == "tables" {
				req.Table, req.Ordered = q.Table, true
			}
		} else {
			req.Focus = q.Focus
		}
		start, err := e.prog.Start(e.ctx, e.child, req, at)
		if err != nil {
			t.Fatalf("start %s: %v", id, err)
		}
		fin := progress.FinishRequest{RoundID: id}
		if q.Kind == "math" {
			fin.Answers = make([]string, len(start.Questions))
			for i := range fin.Answers {
				fin.Answers[i] = "1"
			}
		} else {
			for range start.Words {
				fin.Guesses = append(fin.Guesses, nil)
			}
		}
		resp, err := e.prog.Finish(e.ctx, e.child, fin, at.Add(time.Minute))
		if err != nil {
			t.Fatalf("finish %s: %v", id, err)
		}
		return resp
	}
	finish("quest-round-1", d.Quests[0], now)
	after, _ := e.store.Today(e.ctx, e.child, now)
	if !after.Quests[0].Done || after.Quests[1].Done || after.AllDone {
		t.Fatalf("after one: %+v", after.Quests)
	}
	// A round that matches no open quest changes nothing.
	finish("quest-round-x", Quest{Kind: "math", Focus: "div", Count: 10}, now)
	mid, _ := e.store.Today(e.ctx, e.child, now)
	if mid.Quests[1].Done || mid.Quests[2].Done {
		t.Fatalf("unrelated round completed a quest: %+v", mid.Quests)
	}
	finish("quest-round-2", d.Quests[1], now)
	resp := finish("quest-round-3", d.Quests[2], now)
	done, _ := e.store.Today(e.ctx, e.child, now)
	if !done.AllDone || done.Reveal == nil {
		t.Fatalf("all three should be done with a reveal: %+v", done)
	}
	var touched []kata.Touched
	if err := json.Unmarshal(resp.Reward.Creatures, &touched); err != nil || len(touched) == 0 || !touched[len(touched)-1].NewlySeen {
		t.Fatalf("the reveal should ride along in the round's creatures: %v %s", err, resp.Reward.Creatures)
	}
	// Repeating a quest's round after the day is done changes nothing.
	finish("quest-round-4", d.Quests[0], now)
	still, _ := e.store.Today(e.ctx, e.child, now)
	if string(still.Reveal) != string(done.Reveal) {
		t.Fatal("the reveal must not change")
	}
}

func TestMatches(t *testing.T) {
	cases := []struct {
		q           Quest
		kind, focus string
		want        bool
	}{
		{Quest{Kind: "math", Focus: "mul"}, "math", "mul", true},
		{Quest{Kind: "math", Focus: "mul"}, "math", "addsub,mul", true},
		{Quest{Kind: "math", Focus: "mul"}, "math", "tables:7", false},
		{Quest{Kind: "math", Focus: "tables"}, "math", "tables:7", true},
		{Quest{Kind: "math", Focus: "tables"}, "math", "mul", false},
		{Quest{Kind: "spelling", Focus: "mixed"}, "spelling", "g3-prefixes", true},
		{Quest{Kind: "spelling", Focus: "review"}, "spelling", "mixed", false},
		{Quest{Kind: "spelling", Focus: "review"}, "math", "review", false},
	}
	for _, c := range cases {
		if got := Matches(c.q, c.kind, c.focus); got != c.want {
			t.Errorf("Matches(%+v, %s, %s) = %v", c.q, c.kind, c.focus, got)
		}
	}
}
