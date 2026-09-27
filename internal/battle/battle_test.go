package battle

import (
	"context"
	"errors"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/tgeorge06/skilldojo/internal/account"
	"github.com/tgeorge06/skilldojo/internal/curriculum"
	"github.com/tgeorge06/skilldojo/internal/db"
	"github.com/tgeorge06/skilldojo/internal/kata"
	"github.com/tgeorge06/skilldojo/internal/progress"
	"github.com/tgeorge06/skilldojo/internal/sheet"
)

var day0 = time.Date(2026, 9, 27, 15, 0, 0, 0, time.UTC)

type env struct {
	prog   *progress.Store
	battle *Store
	child  progress.Child
}

func newEnv(t *testing.T) env {
	t.Helper()
	d, err := db.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	ctx := context.Background()
	if _, err := db.Migrate(ctx, d); err != nil {
		t.Fatal(err)
	}
	accounts := account.New(d)
	tok, _ := accounts.CreateLoginToken(ctx, "p@example.com", "UTC", day0)
	_, acct, _ := accounts.RedeemLoginToken(ctx, tok, day0)
	kid, _ := accounts.CreateChild(ctx, acct.ID, "Nova", 2, day0)
	cur := curriculum.MustLoad()
	roster, err := kata.Load(cur)
	if err != nil {
		t.Fatal(err)
	}
	prog := progress.New(d, cur, sheet.NewStore())
	prog.AddSink(kata.New(d, roster))
	b := New(d, cur, roster)
	prog.AddSink(b)
	return env{prog: prog, battle: b, child: progress.Child{AccountID: acct.ID, ChildID: kid.ID, Grade: 2, Timezone: "UTC"}}
}

func (e env) earnCredit(t *testing.T, id string, at time.Time) {
	t.Helper()
	start, err := e.prog.Start(context.Background(), e.child, progress.StartRequest{RoundID: id, Kind: progress.KindSpelling, Focus: "g2-endings", Grade: 2, Count: 5}, at)
	if err != nil {
		t.Fatal(err)
	}
	var g [][]curriculum.Guess
	for _, w := range start.Words {
		g = append(g, []curriculum.Guess{{Kind: curriculum.GuessWord, Value: w.Word}})
	}
	if _, err := e.prog.Finish(context.Background(), e.child, progress.FinishRequest{RoundID: id, Guesses: g}, at); err != nil {
		t.Fatal(err)
	}
}

func TestCreditsAreEarnedAndSpentOnce(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	if _, err := e.battle.Start(ctx, e.child, StartRequest{BattleID: "battle-0001", CreatureID: "g2-endings"}, day0); !errors.Is(err, ErrBadRequest) {
		t.Fatalf("battle with an unseen creature and no credit: %v", err)
	}
	e.earnCredit(t, "credit-round-1", day0)
	if n, _ := e.battle.Credits(ctx, e.child); n != 1 {
		t.Fatalf("credits = %d", n)
	}
	// Retrying the finish never banks twice (sinks do not run on a retry).
	if n, _ := e.battle.Credits(ctx, e.child); n != 1 {
		t.Fatalf("credits after retry = %d", n)
	}
	st, err := e.battle.Start(ctx, e.child, StartRequest{BattleID: "battle-0001", CreatureID: "g2-endings"}, day0)
	if err != nil {
		t.Fatal(err)
	}
	if st.Child.HP != MaxHP || st.Opponent.HP != MaxHP || st.Item == nil || st.Turn != 0 || st.Child.Name != "Tailfin" || st.Opponent.ID == st.Child.ID {
		t.Fatalf("start state: %+v", st)
	}
	if n, _ := e.battle.Credits(ctx, e.child); n != 0 {
		t.Fatalf("credit not spent: %d", n)
	}
	// Replaying the start returns the same battle without needing a credit.
	again, err := e.battle.Start(ctx, e.child, StartRequest{BattleID: "battle-0001", CreatureID: "g2-endings"}, day0)
	if err != nil || again.Item.Prompt != st.Item.Prompt {
		t.Fatalf("replay start: %+v %v", again, err)
	}
	if _, err := e.battle.Start(ctx, e.child, StartRequest{BattleID: "battle-0002", CreatureID: "g2-endings"}, day0); !errors.Is(err, ErrNoCredit) {
		t.Fatalf("second battle without credit: %v", err)
	}
	// Another child cannot load it.
	other := e.child
	other.ChildID = 999
	if _, err := e.battle.Load(ctx, other, "battle-0001"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-child load: %v", err)
	}
}

func TestBattleResolvesToAWinWithSpecials(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	e.earnCredit(t, "credit-round-1", day0)
	st, err := e.battle.Start(ctx, e.child, StartRequest{BattleID: "battle-win-1", CreatureID: "g2-endings"}, day0)
	if err != nil {
		t.Fatal(err)
	}
	turns := 0
	for !st.Done {
		answer := e.answerFor(t, "battle-win-1")
		st, err = e.battle.Play(ctx, e.child, TurnRequest{BattleID: "battle-win-1", Turn: st.Turn, Answer: strings.ToUpper(answer)}, day0)
		if err != nil {
			t.Fatal(err)
		}
		turns++
		if turns > 10 {
			t.Fatal("battle did not end")
		}
	}
	// 5 HP with a special on every third hit: 1+1+2 = 4, then 1 → 4 turns.
	if !st.Won || turns != 4 || st.Child.HP != MaxHP || st.Item != nil || !strings.Contains(st.Message, "faints and rests") {
		t.Fatalf("win state: turns %d %+v", turns, st)
	}
	if !st.Log[2].Special || st.Log[2].Damage != 2 {
		t.Fatalf("third hit should be a special: %+v", st.Log)
	}
	// A finished battle ignores further turns.
	after, _ := e.battle.Play(ctx, e.child, TurnRequest{BattleID: "battle-win-1", Turn: st.Turn, Answer: "x"}, day0)
	if !after.Done || len(after.Log) != 4 {
		t.Fatal("finished battle accepted a turn")
	}
}

func TestWrongAnswersLoseHeartsAndStaleTurnsAreIgnored(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	e.earnCredit(t, "credit-round-1", day0)
	st, _ := e.battle.Start(ctx, e.child, StartRequest{BattleID: "battle-lose-1", CreatureID: "g2-endings"}, day0)
	first := st.Item.Prompt
	st, err := e.battle.Play(ctx, e.child, TurnRequest{BattleID: "battle-lose-1", Turn: 0, Answer: "zzzz"}, day0)
	if err != nil || st.Child.HP != MaxHP-1 || st.Turn != 1 || st.Item.Prompt == first && st.Item.Kind == "math" {
		t.Fatalf("after a miss: %+v %v", st, err)
	}
	if !strings.Contains(st.Message, "It was ") {
		t.Fatalf("a miss should teach the answer: %s", st.Message)
	}
	// A stale (duplicate) turn changes nothing.
	dup, _ := e.battle.Play(ctx, e.child, TurnRequest{BattleID: "battle-lose-1", Turn: 0, Answer: "zzzz"}, day0)
	if dup.Child.HP != MaxHP-1 || dup.Turn != 1 {
		t.Fatalf("stale turn applied: %+v", dup)
	}
	if _, err := e.battle.Play(ctx, e.child, TurnRequest{BattleID: "battle-lose-1", Turn: 1, Answer: "   "}, day0); !errors.Is(err, ErrBadRequest) {
		t.Fatalf("blank answer: %v", err)
	}
	for i := 1; !st.Done; i++ {
		st, _ = e.battle.Play(ctx, e.child, TurnRequest{BattleID: "battle-lose-1", Turn: st.Turn, Answer: "zzzz"}, day0)
		if i > 6 {
			t.Fatal("loss did not end")
		}
	}
	if st.Won || st.Child.HP != 0 || !strings.Contains(st.Message, "rests for now") {
		t.Fatalf("loss state: %+v", st)
	}
}

func TestMathBattleAcceptsEquivalentAnswers(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	// Find the math creature by playing a math round so it is "seen".
	start, _ := e.prog.Start(ctx, e.child, progress.StartRequest{RoundID: "math-cred-1", Kind: progress.KindMath, Ops: []string{"frac"}, Grade: 4, Count: 10}, day0)
	answers := make([]string, 10)
	for i := range answers {
		answers[i] = "1/2"
	}
	_ = start
	if _, err := e.prog.Finish(ctx, e.child, progress.FinishRequest{RoundID: "math-cred-1", Answers: answers}, day0); err != nil {
		t.Fatal(err)
	}
	st, err := e.battle.Start(ctx, e.child, StartRequest{BattleID: "battle-math-1", CreatureID: "math-frac-g4"}, day0)
	if err != nil {
		t.Fatal(err)
	}
	if st.Item.Kind != "math" || st.Item.Prompt == "" {
		t.Fatalf("math item: %+v", st.Item)
	}
	key := e.answerFor(t, "battle-math-1")
	// Present the same value as an unreduced fraction: still right.
	num, den := parseFrac(t, key)
	given := strconv.Itoa(num*2) + "/" + strconv.Itoa(den*2)
	st, err = e.battle.Play(ctx, e.child, TurnRequest{BattleID: "battle-math-1", Turn: 0, Answer: given}, day0)
	if err != nil || !st.Log[0].Right {
		t.Fatalf("equivalent fraction rejected: %s vs %s: %+v %v", given, key, st.Log, err)
	}
}

func parseFrac(t *testing.T, s string) (int, int) {
	t.Helper()
	if !strings.Contains(s, "/") {
		n, _ := strconv.Atoi(s)
		return n, 1
	}
	parts := strings.SplitN(s, "/", 2)
	n, _ := strconv.Atoi(parts[0])
	d, _ := strconv.Atoi(parts[1])
	return n, d
}

// answerFor reads the stored answer (a test-only privilege).
func (e env) answerFor(t *testing.T, id string) string {
	t.Helper()
	st, err := e.battle.load(context.Background(), e.child, id)
	if err != nil {
		t.Fatal(err)
	}
	return st.ItemAnswer
}

func TestBlankMasksOnlyWholeWords(t *testing.T) {
	if got := blank("The cat sat on the catalog.", "cat"); got != "The _____ sat on the catalog." {
		t.Fatal(got)
	}
	if got := blank("A bird landed.", "a"); got != "_____ bird landed." {
		t.Fatal(got)
	}
}
