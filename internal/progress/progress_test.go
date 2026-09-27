package progress

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tgeorge06/skilldojo/internal/account"
	"github.com/tgeorge06/skilldojo/internal/curriculum"
	"github.com/tgeorge06/skilldojo/internal/db"
	"github.com/tgeorge06/skilldojo/internal/sheet"
)

var day0 = time.Date(2026, 9, 27, 15, 0, 0, 0, time.UTC)

func newStore(t *testing.T) (*Store, Child) {
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
	tok, _ := accounts.CreateLoginToken(ctx, "p@example.com", "America/New_York", day0)
	_, acct, err := accounts.RedeemLoginToken(ctx, tok, day0)
	if err != nil {
		t.Fatal(err)
	}
	kid, err := accounts.CreateChild(ctx, acct.ID, "Nova", 2, day0)
	if err != nil {
		t.Fatal(err)
	}
	return New(d, curriculum.MustLoad(), sheet.NewStore()),
		Child{AccountID: acct.ID, ChildID: kid.ID, Grade: kid.Grade, Timezone: "America/New_York"}
}

func winAll(words []curriculum.Word) [][]curriculum.Guess {
	var out [][]curriculum.Guess
	for _, w := range words {
		out = append(out, []curriculum.Guess{{Kind: curriculum.GuessWord, Value: w.Word}})
	}
	return out
}

func loseAll(words []curriculum.Word) [][]curriculum.Guess {
	var out [][]curriculum.Guess
	for range words {
		var g []curriculum.Guess
		for i := 0; i < 6; i++ {
			g = append(g, curriculum.Guess{Kind: curriculum.GuessWord, Value: "zzzz"})
		}
		out = append(out, g)
	}
	return out
}

func TestSpellingRoundLifecycle(t *testing.T) {
	s, kid := newStore(t)
	ctx := context.Background()

	start, err := s.Start(ctx, kid, StartRequest{RoundID: "round-0001", Kind: KindSpelling, Focus: "g2-endings", Grade: 2, Count: 10}, day0)
	if err != nil {
		t.Fatal(err)
	}
	if len(start.Words) != 10 {
		t.Fatalf("got %d words", len(start.Words))
	}
	for _, w := range start.Words[:5] {
		if w.Skill != "g2-endings" {
			t.Fatalf("focused session should lead with the focus skill, got %s", w.Skill)
		}
	}
	// Replaying start returns the same words in the same order.
	again, err := s.Start(ctx, kid, StartRequest{RoundID: "round-0001", Kind: KindSpelling, Focus: "g2-endings", Grade: 2, Count: 10}, day0)
	if err != nil || len(again.Words) != 10 || again.Words[3].Word != start.Words[3].Word {
		t.Fatalf("restart differs: %v %v", err, again.Words)
	}

	// The client claims wins on words it lost: the server replays and disagrees.
	guesses := winAll(start.Words)
	guesses[0] = loseAll(start.Words[:1])[0]
	guesses[1] = []curriculum.Guess{{Kind: curriculum.GuessWord, Value: strings.ToUpper(start.Words[1].Word)}} // browser would accept; server charges
	fin, err := s.Finish(ctx, kid, FinishRequest{RoundID: "round-0001", Guesses: guesses}, day0)
	if err != nil {
		t.Fatal(err)
	}
	if fin.Score != 8 || fin.Total != 10 || fin.Percent != 80 || len(fin.WordResults) != 10 {
		t.Fatalf("finish = %+v", fin)
	}
	if fin.WordResults[0].Won || fin.WordResults[1].Won || !fin.WordResults[2].Won {
		t.Fatalf("word results %+v", fin.WordResults[:3])
	}
	// Reward: own-grade words earn 1 each, mosaic counts every correct one.
	if fin.Reward.Fills != 8 || fin.Reward.MosaicCells != 8 || !fin.Reward.BattleCredit || len(fin.Reward.Evolved) != 0 {
		t.Fatalf("reward = %+v", fin.Reward)
	}
	if fin.Reward.FillsBySkill["g2-endings"] != 3 { // five focus words, first two lost
		t.Fatalf("fills by skill = %v", fin.Reward.FillsBySkill)
	}

	// A retry returns the identical stored response and grants nothing new.
	retry, err := s.Finish(ctx, kid, FinishRequest{RoundID: "round-0001", Guesses: winAll(start.Words)}, day0.Add(time.Minute))
	if err != nil || retry.Score != 8 || retry.Reward.Fills != 8 {
		t.Fatalf("retry = %+v, %v", retry, err)
	}
	if _, err := s.Start(ctx, kid, StartRequest{RoundID: "round-0001", Kind: KindSpelling, Grade: 2, Count: 5}, day0); !errors.Is(err, ErrBadRequest) {
		t.Fatalf("restart of a finished round: %v", err)
	}

	// Another child cannot finish it.
	other := kid
	other.ChildID = 999
	if _, err := s.Finish(ctx, other, FinishRequest{RoundID: "round-0001", Guesses: guesses}, day0); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-child finish: %v", err)
	}
	// A finished round returns its stored result whatever the retry carries.
	if r, err := s.Finish(ctx, kid, FinishRequest{RoundID: "round-0001", Guesses: guesses[:3]}, day0); err != nil || r.Score != 8 {
		t.Fatalf("retry with a different body: %+v, %v", r, err)
	}
}

func TestRewardWeightsByGrade(t *testing.T) {
	s, kid := newStore(t) // child is grade 2
	ctx := context.Background()
	cases := []struct {
		grade     int
		wantFills int
	}{{1, 0}, {2, 5}, {3, 10}}
	for i, tc := range cases {
		id := "grade-round-" + string(rune('a'+i))
		start, err := s.Start(ctx, kid, StartRequest{RoundID: id, Kind: KindSpelling, Focus: FocusMixed, Grade: tc.grade, Count: 5}, day0)
		if err != nil {
			t.Fatal(err)
		}
		fin, err := s.Finish(ctx, kid, FinishRequest{RoundID: id, Guesses: winAll(start.Words)}, day0)
		if err != nil {
			t.Fatal(err)
		}
		if fin.Reward.Fills != tc.wantFills || fin.Reward.MosaicCells != 5 {
			t.Errorf("grade %d: fills %d (want %d), mosaic %d", tc.grade, fin.Reward.Fills, tc.wantFills, fin.Reward.MosaicCells)
		}
	}
}

func TestPreviouslyMissedWordsPayDoubleAndFeedReview(t *testing.T) {
	s, kid := newStore(t)
	ctx := context.Background()
	start, _ := s.Start(ctx, kid, StartRequest{RoundID: "miss-round-1", Kind: KindSpelling, Focus: "g2-endings", Grade: 2, Count: 5}, day0)
	if _, err := s.Finish(ctx, kid, FinishRequest{RoundID: "miss-round-1", Guesses: loseAll(start.Words)}, day0); err != nil {
		t.Fatal(err)
	}
	// One miss is not yet "keeps missing".
	if missed, _ := s.MissedWords(ctx, kid, day0); len(missed) != 0 {
		t.Fatalf("one miss should not count: %v", missed)
	}
	if _, err := s.Start(ctx, kid, StartRequest{RoundID: "review-early", Kind: KindSpelling, Focus: FocusReview, Grade: 2, Count: 5}, day0); !errors.Is(err, ErrBadRequest) {
		t.Fatalf("review with nothing to review: %v", err)
	}
	// Same words again, lost again: now they are review material.
	s2, _ := s.Start(ctx, kid, StartRequest{RoundID: "miss-round-2", Kind: KindSpelling, Focus: "g2-endings", Grade: 2, Count: 5}, day0.Add(time.Hour))
	if _, err := s.Finish(ctx, kid, FinishRequest{RoundID: "miss-round-2", Guesses: loseAll(s2.Words)}, day0.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	missed, _ := s.MissedWords(ctx, kid, day0.Add(2*time.Hour))
	if len(missed) == 0 {
		t.Fatal("two misses should feed review")
	}
	for _, m := range missed {
		if m.Skill != "g2-endings" {
			t.Fatalf("missed word carries its skill: %+v", m)
		}
	}
	review, err := s.Start(ctx, kid, StartRequest{RoundID: "review-round", Kind: KindSpelling, Focus: FocusReview, Grade: 2, Count: 10}, day0.Add(2*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range review.Words {
		found := false
		for _, m := range missed {
			if m.Word == w.Word {
				found = true
			}
		}
		if !found {
			t.Fatalf("review served %q which is not a missed word", w.Word)
		}
	}
	fin, err := s.Finish(ctx, kid, FinishRequest{RoundID: "review-round", Guesses: winAll(review.Words)}, day0.Add(3*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if fin.Reward.Fills != 2*len(review.Words) {
		t.Fatalf("previously missed words should earn 2 each: %+v", fin.Reward)
	}
	if fin.Reward.ReviewDue != 0 {
		t.Fatalf("a correct latest attempt clears the word from review, got %d due", fin.Reward.ReviewDue)
	}
	// Once answered correctly, the same words are ordinary practice again:
	// a single old miss cannot be farmed for double credit.
	again, _ := s.Start(ctx, kid, StartRequest{RoundID: "after-review", Kind: KindSpelling, Focus: "g2-endings", Grade: 2, Count: 5}, day0.Add(4*time.Hour))
	fin2, err := s.Finish(ctx, kid, FinishRequest{RoundID: "after-review", Guesses: winAll(again.Words)}, day0.Add(4*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if fin2.Reward.Fills != 5 {
		t.Fatalf("words corrected since their miss should earn 1 each, got %+v", fin2.Reward)
	}
}

func TestLeitnerScheduleAndEvolution(t *testing.T) {
	s, kid := newStore(t)
	ctx := context.Background()
	skill := "g2-r-controlled"
	play := func(id string, at time.Time, win bool) Reward {
		t.Helper()
		start, err := s.Start(ctx, kid, StartRequest{RoundID: id, Kind: KindSpelling, Focus: skill, Grade: 2, Count: 5}, at)
		if err != nil {
			t.Fatal(err)
		}
		g := winAll(start.Words)
		if !win {
			g = loseAll(start.Words)
		}
		fin, err := s.Finish(ctx, kid, FinishRequest{RoundID: id, Guesses: g}, at)
		if err != nil {
			t.Fatal(err)
		}
		return fin.Reward
	}
	progressFor := func(at time.Time) SkillProgress {
		t.Helper()
		all, err := s.Progress(ctx, kid, at)
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range all {
			if p.SkillID == skill {
				return p
			}
		}
		t.Fatalf("no progress for %s", skill)
		return SkillProgress{}
	}

	play("leitner-r1", day0, true) // box 0 -> 1, due in 3 days
	p := progressFor(day0)
	if p.Box != 1 || p.ReviewsOK != 1 || p.DueOn != "2026-09-30" || p.FirstOKOn != "2026-09-27" {
		t.Fatalf("after first win: %+v", p)
	}
	play("leitner-r2", day0.AddDate(0, 0, 1), true) // early: schedule unchanged
	if p = progressFor(day0); p.Box != 1 || p.ReviewsOK != 1 || p.DueOn != "2026-09-30" {
		t.Fatalf("early success moved the schedule: %+v", p)
	}
	day3 := day0.AddDate(0, 0, 3)
	play("leitner-r3", day3, true) // due: box 2, +7 days
	if p = progressFor(day3); p.Box != 2 || p.ReviewsOK != 2 || p.DueOn != "2026-10-07" {
		t.Fatalf("after second review: %+v", p)
	}
	day10 := day0.AddDate(0, 0, 10)
	r := play("leitner-r4", day10, true) // box 3, 3 reviews, but only 10 days since first ok
	if p = progressFor(day10); p.Box != 3 || p.Evolved || len(r.Evolved) != 0 {
		t.Fatalf("evolved too early: %+v %+v", p, r)
	}
	// Grinding more today changes nothing.
	play("leitner-r5", day10.Add(time.Hour), true)
	if p = progressFor(day10); p.Box != 3 || p.ReviewsOK != 3 || p.Evolved {
		t.Fatalf("grind changed schedule: %+v", p)
	}
	// Two weeks after the first success the same standing counts as evolved
	// by the calendar; the next round records and reports it exactly once.
	day14 := day0.AddDate(0, 0, 14)
	if p = progressFor(day14); !p.Evolved {
		t.Fatalf("should be evolved by the calendar: %+v", p)
	}
	r = play("leitner-r6", day14, true)
	if len(r.Evolved) != 1 || r.Evolved[0] != skill {
		t.Fatalf("calendar crossing should be reported on the next round: %+v", r)
	}
	r = play("leitner-r7", day14.Add(time.Hour), true)
	if len(r.Evolved) != 0 {
		t.Fatalf("evolution reported twice: %+v", r)
	}
	// A bad round drops a box and reschedules, but evolution is permanent.
	day15 := day0.AddDate(0, 0, 15)
	play("leitner-r8", day15, false)
	if p = progressFor(day15); p.Box != 2 || p.DueOn != "2026-10-19" || !p.Evolved {
		t.Fatalf("after a miss: %+v", p)
	}
}

func TestEvolutionIsReportedOnceWhenCrossed(t *testing.T) {
	s, kid := newStore(t)
	ctx := context.Background()
	skill := "g2-two-syllable"
	play := func(id string, at time.Time) Reward {
		start, _ := s.Start(ctx, kid, StartRequest{RoundID: id, Kind: KindSpelling, Focus: skill, Grade: 2, Count: 5}, at)
		fin, err := s.Finish(ctx, kid, FinishRequest{RoundID: id, Guesses: winAll(start.Words)}, at)
		if err != nil {
			t.Fatal(err)
		}
		return fin.Reward
	}
	play("evo-r001", day0)                        // box 1, due day 3
	play("evo-r002", day0.AddDate(0, 0, 3))       // box 2, due day 10
	play("evo-r003", day0.AddDate(0, 0, 10))      // box 3, 3 reviews, 10 days: not yet
	r := play("evo-r004", day0.AddDate(0, 0, 14)) // 14 days since first ok: crosses now
	if len(r.Evolved) != 1 || r.Evolved[0] != skill {
		t.Fatalf("expected evolution to be reported once: %+v", r)
	}
}

func TestMathRoundGradesOnServer(t *testing.T) {
	s, kid := newStore(t)
	ctx := context.Background()
	start, err := s.Start(ctx, kid, StartRequest{RoundID: "math-round-1", Kind: KindMath, Ops: []string{"addsub"}, Grade: 2, Count: 10}, day0)
	if err != nil {
		t.Fatal(err)
	}
	if len(start.Questions) != 10 || start.SheetID == "" {
		t.Fatalf("start = %+v", start)
	}
	again, err := s.Start(ctx, kid, StartRequest{RoundID: "math-round-1", Kind: KindMath, Ops: []string{"addsub"}, Grade: 2, Count: 10}, day0)
	if err != nil || again.SheetID != start.SheetID {
		t.Fatalf("restart should return the same sheet: %v", err)
	}
	answers := make([]string, 10) // all blank: zero correct
	fin, err := s.Finish(ctx, kid, FinishRequest{RoundID: "math-round-1", Answers: answers}, day0)
	if err != nil {
		t.Fatal(err)
	}
	if fin.Score != 0 || fin.Total != 10 || len(fin.Results) != 10 || fin.Reward.Fills != 0 || fin.Reward.MosaicCells != 0 || fin.Reward.BattleCredit {
		t.Fatalf("blank answers must not earn a battle credit: %+v", fin)
	}
	all, _ := s.Progress(ctx, kid, day0)
	if len(all) != 1 || all[0].SkillID != "math-addsub-g2" || all[0].Box != 0 {
		t.Fatalf("progress = %+v", all)
	}
	if _, err := s.Finish(ctx, kid, FinishRequest{RoundID: "nope-nope", Answers: answers}, day0); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown round: %v", err)
	}
	if _, err := s.Start(ctx, kid, StartRequest{RoundID: "x", Kind: KindMath, Ops: []string{"addsub"}, Grade: 2, Count: 10}, day0); !errors.Is(err, ErrBadRequest) {
		t.Fatalf("short round id accepted: %v", err)
	}
}

func TestBelowGradeNeverEarnsFillsEvenWhenMissed(t *testing.T) {
	s, kid := newStore(t) // grade 2
	ctx := context.Background()
	first, _ := s.Start(ctx, kid, StartRequest{RoundID: "below-r1", Kind: KindSpelling, Focus: "g1-blends", Grade: 1, Count: 5}, day0)
	s.Finish(ctx, kid, FinishRequest{RoundID: "below-r1", Guesses: loseAll(first.Words)}, day0)
	second, _ := s.Start(ctx, kid, StartRequest{RoundID: "below-r2", Kind: KindSpelling, Focus: "g1-blends", Grade: 1, Count: 5}, day0.Add(time.Hour))
	fin, err := s.Finish(ctx, kid, FinishRequest{RoundID: "below-r2", Guesses: winAll(second.Words)}, day0.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if fin.Reward.Fills != 0 || fin.Reward.MosaicCells != 5 {
		t.Fatalf("below-grade words earn no fills even after a miss: %+v", fin.Reward)
	}
}

func TestBattleCreditNeedsFiveAttempts(t *testing.T) {
	s, kid := newStore(t)
	ctx := context.Background()
	if _, err := s.Start(ctx, kid, StartRequest{RoundID: "credit-r1", Kind: KindSpelling, Focus: FocusMixed, Grade: 2, Count: 5}, day0); err != nil {
		t.Fatal(err)
	}
	fin, err := s.Finish(ctx, kid, FinishRequest{RoundID: "credit-r1", Guesses: make([][]curriculum.Guess, 5)}, day0)
	if err != nil {
		t.Fatal(err)
	}
	if fin.Reward.BattleCredit {
		t.Fatal("five empty guess lists earned a battle credit")
	}
}

func TestFailedMathFinishLeavesTheRoundFinishable(t *testing.T) {
	s, kid := newStore(t)
	ctx := context.Background()
	start, _ := s.Start(ctx, kid, StartRequest{RoundID: "math-keep-1", Kind: KindMath, Ops: []string{"addsub"}, Grade: 2, Count: 10}, day0)
	// A malformed submission fails before any write and must not consume the sheet.
	if _, err := s.Finish(ctx, kid, FinishRequest{RoundID: "math-keep-1", Answers: []string{"1"}}, day0); !errors.Is(err, ErrBadRequest) {
		t.Fatalf("short answers: %v", err)
	}
	if _, ok := s.sheets.Peek(start.SheetID); !ok {
		t.Fatal("sheet consumed by a failed finish")
	}
	fin, err := s.Finish(ctx, kid, FinishRequest{RoundID: "math-keep-1", Answers: make([]string, 10)}, day0)
	if err != nil || fin.Total != 10 {
		t.Fatalf("second finish: %+v, %v", fin, err)
	}
	if _, ok := s.sheets.Peek(start.SheetID); ok {
		t.Fatal("sheet should be removed after a committed finish")
	}
}

func TestReviewResolvesSightWordsFromTheSightBank(t *testing.T) {
	s, kid := newStore(t)
	ctx := context.Background()
	// "said" is both a grade-1 heart word and a sight word; missing it as a
	// sight word must bring back the sight entry with its rank.
	for i, id := range []string{"sight-miss-1", "sight-miss-2"} {
		at := day0.Add(time.Duration(i) * time.Hour)
		start, err := s.Start(ctx, kid, StartRequest{RoundID: id, Kind: KindSpelling, Focus: curriculum.SightWordSkill, Grade: 1, Count: 10}, at)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := s.Finish(ctx, kid, FinishRequest{RoundID: id, Guesses: loseAll(start.Words)}, at); err != nil {
			t.Fatal(err)
		}
	}
	review, err := s.Start(ctx, kid, StartRequest{RoundID: "sight-review", Kind: KindSpelling, Focus: FocusReview, Grade: 1, Count: 10}, day0.Add(3*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range review.Words {
		if w.Skill != curriculum.SightWordSkill || w.Rank == 0 {
			t.Fatalf("review returned a non-sight entry: %+v", w)
		}
	}
}

func TestSelectWordsStrategies(t *testing.T) {
	s, _ := newStore(t)
	sight := s.selectWords(curriculum.SightWordSkill, 2, 10, nil)
	var current, earlier int
	for _, w := range sight {
		if s.sightBand(w.Word) == 2 {
			current++
		} else {
			earlier++
		}
	}
	if len(sight) != 10 || current != 6 || earlier != 4 {
		t.Fatalf("sight mix = %d current / %d earlier", current, earlier)
	}
	mixed := s.selectWords(FocusMixed, 3, 5, nil)
	skills := map[string]bool{}
	for _, w := range mixed {
		skills[w.Skill] = true
	}
	if len(mixed) != 5 || len(skills) != 5 {
		t.Fatalf("mixed session should rotate skills: %v", skills)
	}
	if got := s.selectWords(curriculum.SightWordSkill, 1, 5, nil); len(got) != 5 {
		t.Fatalf("grade 1 sight words = %d", len(got))
	}
}
