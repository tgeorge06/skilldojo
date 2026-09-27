package kata

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tgeorge06/skilldojo/internal/account"
	"github.com/tgeorge06/skilldojo/internal/curriculum"
	"github.com/tgeorge06/skilldojo/internal/db"
	"github.com/tgeorge06/skilldojo/internal/progress"
	"github.com/tgeorge06/skilldojo/internal/sheet"
)

var day0 = time.Date(2026, 9, 27, 15, 0, 0, 0, time.UTC)

func TestRosterCoversEverySkillOnce(t *testing.T) {
	cur := curriculum.MustLoad()
	r, err := Load(cur)
	if err != nil {
		t.Fatal(err)
	}
	if n := len(r.Creatures()); n != 51 {
		t.Fatalf("roster has %d creatures, want 51", n)
	}
	names := map[string]bool{}
	for _, c := range r.Creatures() {
		if strings.Contains(strings.ToLower(c.Name), "pok") || strings.Contains(strings.ToLower(c.Name), "mon") && strings.HasSuffix(strings.ToLower(c.Name), "mon") {
			t.Errorf("name %q is too close to the thing we are not copying", c.Name)
		}
		names[c.Name] = true
	}
	if len(names) != 51 {
		t.Fatal("names are not unique")
	}
	// Specific hints always name a grade and a subject so Train here is obvious.
	for _, c := range r.Creatures() {
		if !strings.Contains(c.HintSpecific, "Grade ") {
			t.Errorf("%s specific hint lacks a grade: %q", c.ID, c.HintSpecific)
		}
	}
}

func TestRosterValidationRejectsLeaksAndGaps(t *testing.T) {
	cur := curriculum.MustLoad()
	good := rosterJSON
	defer func() { rosterJSON = good }()

	var f rosterFile
	json.Unmarshal(good, &f)
	// Leak: a hint containing one of the skill's own words.
	leaky := f
	leaky.Creatures = append([]Creature(nil), f.Creatures...)
	var bankWord string
	for _, w := range cur.Words(1) {
		if w.Skill == "g1-blends" && len(w.Word) > 3 {
			bankWord = w.Word
			break
		}
	}
	for i, c := range leaky.Creatures {
		if c.ID == "g1-blends" {
			leaky.Creatures[i].HintVague = "Look for the word " + bankWord + "."
		}
	}
	rosterJSON, _ = json.Marshal(leaky)
	if _, err := Load(cur); err == nil || !strings.Contains(err.Error(), "leaks") {
		t.Fatalf("leaky hint accepted: %v", err)
	}
	// Gap: drop a creature.
	gap := f
	gap.Creatures = f.Creatures[1:]
	rosterJSON, _ = json.Marshal(gap)
	if _, err := Load(cur); err == nil || !strings.Contains(err.Error(), "no creature for skill") {
		t.Fatalf("missing creature accepted: %v", err)
	}
	// Stray: a creature for a skill that does not exist.
	stray := f
	stray.Creatures = append(append([]Creature(nil), f.Creatures...), Creature{ID: "g9-nope", SkillID: "g9-nope", Kind: "spelling", Grade: 1, Name: "Nope", HintVague: "x", HintSpecific: "x", Seed: 99, Regions: 20})
	rosterJSON, _ = json.Marshal(stray)
	if _, err := Load(cur); err == nil {
		t.Fatal("stray creature accepted")
	}
	// A creature whose kind or grade disagrees with its skill id is rejected.
	wrong := f
	wrong.Creatures = append([]Creature(nil), f.Creatures...)
	for i, c := range wrong.Creatures {
		if c.ID == "g2-endings" {
			wrong.Creatures[i].Grade = 3
		}
	}
	rosterJSON, _ = json.Marshal(wrong)
	if _, err := Load(cur); err == nil || !strings.Contains(err.Error(), "do not match") {
		t.Fatalf("wrong grade accepted: %v", err)
	}
	// Sight hints may use function words but not content words from the band.
	sightLeak := f
	sightLeak.Creatures = append([]Creature(nil), f.Creatures...)
	var content string
	for _, w := range cur.SightWords(1) {
		if !hintStopWords[w.Word] {
			content = w.Word
			break
		}
	}
	for i, c := range sightLeak.Creatures {
		if c.ID == "sight-g1" {
			sightLeak.Creatures[i].HintVague = "The one " + content + " on every page."
		}
	}
	rosterJSON, _ = json.Marshal(sightLeak)
	if _, err := Load(cur); err == nil || !strings.Contains(err.Error(), "leaks") {
		t.Fatalf("sight content word accepted: %v", err)
	}
}

type env struct {
	prog  *progress.Store
	kata  *Store
	cur   *curriculum.Curriculum
	child progress.Child
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
	roster, err := Load(cur)
	if err != nil {
		t.Fatal(err)
	}
	prog := progress.New(d, cur, sheet.NewStore())
	k := New(d, roster)
	prog.AddSink(k)
	return env{prog: prog, kata: k, cur: cur, child: progress.Child{AccountID: acct.ID, ChildID: kid.ID, Grade: 2, Timezone: "UTC"}}
}

func winAll(words []curriculum.Word) [][]curriculum.Guess {
	var out [][]curriculum.Guess
	for _, w := range words {
		out = append(out, []curriculum.Guess{{Kind: curriculum.GuessWord, Value: w.Word}})
	}
	return out
}

func (e env) play(t *testing.T, id, focus string, grade int, at time.Time) progress.FinishResponse {
	t.Helper()
	start, err := e.prog.Start(context.Background(), e.child, progress.StartRequest{RoundID: id, Kind: progress.KindSpelling, Focus: focus, Grade: grade, Count: 5}, at)
	if err != nil {
		t.Fatal(err)
	}
	fin, err := e.prog.Finish(context.Background(), e.child, progress.FinishRequest{RoundID: id, Guesses: winAll(start.Words)}, at)
	if err != nil {
		t.Fatal(err)
	}
	return fin
}

func (e env) index(t *testing.T, at time.Time) Index {
	t.Helper()
	prog, _ := e.prog.Progress(context.Background(), e.child, at)
	idx, err := e.kata.Index(context.Background(), e.child, prog, 0, e.cur)
	if err != nil {
		t.Fatal(err)
	}
	return idx
}

func entry(idx Index, id string) Entry {
	for _, e := range idx.Entries {
		if e.ID == id {
			return e
		}
	}
	return Entry{}
}

func TestSeenCaughtEvolvedThroughRounds(t *testing.T) {
	e := newEnv(t)
	skill := "g2-endings"

	// Before any round: unknown, vague hint, no name.
	idx := e.index(t, day0)
	if len(idx.Entries) != 51 || entry(idx, skill).State != StateUnknown || entry(idx, skill).Hint == "" {
		t.Fatalf("fresh index: %+v", entry(idx, skill))
	}
	if entry(idx, skill).Hint == entryHintSpecific(e, skill) {
		t.Fatal("unknown creature should show the vague hint")
	}

	// A focused round: five focus words, plus review words from other skills.
	fin := e.play(t, "kata-round-1", skill, 2, day0)
	var touched []Touched
	if err := json.Unmarshal(fin.Reward.Creatures, &touched); err != nil {
		t.Fatal(err)
	}
	var focusTouch *Touched
	for i := range touched {
		if touched[i].ID == skill {
			focusTouch = &touched[i]
		}
	}
	if focusTouch == nil || !focusTouch.NewlySeen || focusTouch.FillsAdded != 5 || focusTouch.Fills != 5 || focusTouch.State != StateSeen {
		t.Fatalf("focus creature after one round: %+v", touched)
	}
	idx = e.index(t, day0)
	got := entry(idx, skill)
	if got.State != StateSeen || got.Fills != 5 || got.Hint != entryHintSpecific(e, skill) || got.Name != "Tailfin" {
		t.Fatalf("seen entry: %+v", got)
	}
	if idx.ByGrade[2][1] != 10 || idx.ByGrade[2][0] != 0 {
		t.Fatalf("grade 2 totals: %v", idx.ByGrade[2])
	}

	// Enough rounds to color all 20 regions: caught, without evolving.
	for i := 2; i <= 4; i++ {
		fin = e.play(t, "kata-round-"+string(rune('0'+i)), skill, 2, day0.Add(time.Duration(i)*time.Hour))
	}
	json.Unmarshal(fin.Reward.Creatures, &touched)
	for _, tc := range touched {
		if tc.ID == skill && (!tc.Caught || tc.State != StateCaught || tc.Fills != tc.Regions) {
			t.Fatalf("should be caught on the fourth round: %+v", tc)
		}
	}
	idx = e.index(t, day0)
	if entry(idx, skill).State != StateCaught || idx.ByGrade[2][0] != 1 {
		t.Fatalf("caught entry: %+v totals %v", entry(idx, skill), idx.ByGrade[2])
	}
	// Fills never exceed regions.
	e.play(t, "kata-round-x", skill, 2, day0.Add(5*time.Hour))
	if got := entry(e.index(t, day0), skill); got.Fills != got.Regions {
		t.Fatal("fills exceeded regions")
	}

	// Evolution follows mastery: due-date reviews across two weeks.
	e.play(t, "kata-evo-1", skill, 2, day0.AddDate(0, 0, 3))
	e.play(t, "kata-evo-2", skill, 2, day0.AddDate(0, 0, 10))
	fin = e.play(t, "kata-evo-3", skill, 2, day0.AddDate(0, 0, 14))
	json.Unmarshal(fin.Reward.Creatures, &touched)
	var evolved bool
	for _, tc := range touched {
		if tc.ID == skill && tc.Evolved && tc.State == StateEvolved {
			evolved = true
		}
	}
	if !evolved {
		t.Fatalf("expected evolution on day 14: %+v", touched)
	}
	if got := entry(e.index(t, day0.AddDate(0, 0, 14)), skill); got.State != StateEvolved || !got.Mastered {
		t.Fatalf("evolved entry: %+v", got)
	}
}

func entryHintSpecific(e env, skill string) string {
	c, _ := e.kata.roster.BySkill(skill)
	return c.HintSpecific
}

func TestBelowGradeRoundsRevealButDoNotColor(t *testing.T) {
	e := newEnv(t) // grade 2 child
	fin := e.play(t, "below-kata-1", "g1-blends", 1, day0)
	var touched []Touched
	json.Unmarshal(fin.Reward.Creatures, &touched)
	for _, tc := range touched {
		if tc.ID == "g1-blends" && (!tc.NewlySeen || tc.FillsAdded != 0 || tc.State != StateSeen) {
			t.Fatalf("below-grade creature: %+v", tc)
		}
	}
	got := entry(e.index(t, day0), "g1-blends")
	if got.State != StateSeen || got.Fills != 0 {
		t.Fatalf("below-grade entry: %+v", got)
	}
}

func TestBlankRoundsDoNotRevealCreatures(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	if _, err := e.prog.Start(ctx, e.child, progress.StartRequest{RoundID: "blank-round-1", Kind: progress.KindSpelling, Focus: "g2-endings", Grade: 2, Count: 5}, day0); err != nil {
		t.Fatal(err)
	}
	fin, err := e.prog.Finish(ctx, e.child, progress.FinishRequest{RoundID: "blank-round-1", Guesses: make([][]curriculum.Guess, 5)}, day0)
	if err != nil {
		t.Fatal(err)
	}
	if len(fin.Reward.Creatures) != 0 {
		t.Fatalf("blank round touched creatures: %s", fin.Reward.Creatures)
	}
	if got := entry(e.index(t, day0), "g2-endings"); got.State != StateUnknown {
		t.Fatalf("blank round revealed a creature: %+v", got)
	}
}

func TestFillsAddedIsTheCappedDelta(t *testing.T) {
	e := newEnv(t)
	skill := "g2-endings"
	var fin progress.FinishResponse
	for i := 0; i < 5; i++ {
		fin = e.play(t, "cap-round-"+string(rune('a'+i)), skill, 2, day0.Add(time.Duration(i)*time.Hour))
	}
	var touched []Touched
	json.Unmarshal(fin.Reward.Creatures, &touched)
	for _, tc := range touched {
		if tc.ID == skill && (tc.Fills != tc.Regions || tc.FillsAdded != 0) {
			t.Fatalf("full creature should report +0: %+v", tc)
		}
	}
}

func TestIndexIsFencedPerChild(t *testing.T) {
	e := newEnv(t)
	e.play(t, "fence-round-1", "g2-endings", 2, day0)
	other := e.child
	other.ChildID = 999
	idx, err := e.kata.Index(context.Background(), other, nil, 0, e.cur)
	if err != nil {
		t.Fatal(err)
	}
	for _, en := range idx.Entries {
		if en.State != StateUnknown {
			t.Fatalf("another child sees state: %+v", en)
		}
	}
}
