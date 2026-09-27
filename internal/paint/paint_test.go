package paint

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/tgeorge06/skilldojo/internal/account"
	"github.com/tgeorge06/skilldojo/internal/curriculum"
	"github.com/tgeorge06/skilldojo/internal/db"
	"github.com/tgeorge06/skilldojo/internal/progress"
	"github.com/tgeorge06/skilldojo/internal/sheet"
)

var day0 = time.Date(2026, 9, 30, 23, 30, 0, 0, time.UTC) // a Wednesday in UTC

type env struct {
	prog   *progress.Store
	paint  *Store
	sheets *sheet.Store
	child  progress.Child
}

func newEnv(t *testing.T, tz string) env {
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
	tok, _ := accounts.CreateLoginToken(ctx, "p@example.com", tz, day0)
	_, acct, _ := accounts.RedeemLoginToken(ctx, tok, day0)
	kid, _ := accounts.CreateChild(ctx, acct.ID, "Nova", 2, day0)
	sheets := sheet.NewStore()
	prog := progress.New(d, curriculum.MustLoad(), sheets)
	p, err := New(d, sheets)
	if err != nil {
		t.Fatal(err)
	}
	prog.AddSink(p)
	return env{prog: prog, paint: p, sheets: sheets, child: progress.Child{AccountID: acct.ID, ChildID: kid.ID, Grade: 2, Timezone: tz}}
}

func TestMosaicsAreWellFormed(t *testing.T) {
	s, _ := New(nil, sheet.NewStore())
	if len(s.mosaics) < 3 {
		t.Fatalf("only %d mosaics", len(s.mosaics))
	}
	for _, m := range s.mosaics {
		if len(m.Cells) != m.Size*m.Size || m.Name == "" {
			t.Errorf("mosaic %s malformed", m.ID)
		}
		if !strings.ContainsAny(m.Cells, "123456789") {
			t.Errorf("mosaic %s is blank", m.ID)
		}
	}
	order := RevealOrder(20, "2026-09-28", 7)
	seen := map[int]bool{}
	for _, i := range order {
		seen[i] = true
	}
	if len(order) != 400 || len(seen) != 400 {
		t.Fatal("reveal order is not a permutation")
	}
	if strings.Join(toStrings(order[:5]), ",") != strings.Join(toStrings(RevealOrder(20, "2026-09-28", 7)[:5]), ",") {
		t.Fatal("reveal order is not deterministic")
	}
	if strings.Join(toStrings(order[:5]), ",") == strings.Join(toStrings(RevealOrder(20, "2026-10-05", 7)[:5]), ",") {
		t.Fatal("reveal order should differ by week")
	}
}

func toStrings(xs []int) []string {
	out := make([]string, len(xs))
	for i, x := range xs {
		out[i] = strconv.Itoa(x)
	}
	return out
}

func TestWeekKeyFollowsTheLocalCalendar(t *testing.T) {
	// 23:30 UTC on Wednesday Sept 30 is still Wednesday in New York but
	// already Thursday Oct 1 in Tokyo; both are the week of Monday Sept 28.
	if k := WeekKey(day0, "America/New_York"); k != "2026-09-28" {
		t.Fatalf("NY week = %s", k)
	}
	if k := WeekKey(day0, "Asia/Tokyo"); k != "2026-09-28" {
		t.Fatalf("Tokyo week = %s", k)
	}
	// Sunday 23:30 UTC is Monday in Tokyo: a new week there, not in New York.
	sunday := time.Date(2026, 10, 4, 23, 30, 0, 0, time.UTC)
	if k := WeekKey(sunday, "Asia/Tokyo"); k != "2026-10-05" {
		t.Fatalf("Tokyo Monday = %s", k)
	}
	if k := WeekKey(sunday, "America/New_York"); k != "2026-09-28" {
		t.Fatalf("NY Sunday = %s", k)
	}
	if k := WeekKey(sunday, "Mars/Olympus"); k != "2026-09-28" {
		t.Fatalf("bad tz should fall back to UTC: %s", k)
	}
}

func winAll(words []curriculum.Word) [][]curriculum.Guess {
	var out [][]curriculum.Guess
	for _, w := range words {
		out = append(out, []curriculum.Guess{{Kind: curriculum.GuessWord, Value: w.Word}})
	}
	return out
}

func TestMosaicFillsOneCellPerCorrectAnswerAndCapsAtTheImage(t *testing.T) {
	e := newEnv(t, "America/New_York")
	ctx := context.Background()
	start, _ := e.prog.Start(ctx, e.child, progress.StartRequest{RoundID: "mosaic-r1", Kind: progress.KindSpelling, Focus: "g2-endings", Grade: 2, Count: 5}, day0)
	fin, err := e.prog.Finish(ctx, e.child, progress.FinishRequest{RoundID: "mosaic-r1", Guesses: winAll(start.Words)}, day0)
	if err != nil {
		t.Fatal(err)
	}
	var w Week
	if err := json.Unmarshal(fin.Reward.Mosaic, &w); err != nil || w.Added != 5 || w.Revealed != 5 || w.Total != 400 || w.WeekKey != "2026-09-28" {
		t.Fatalf("mosaic reward: %+v %v", w, err)
	}
	cur, err := e.paint.CurrentWeek(ctx, e.child, day0)
	if err != nil || cur.Revealed != 5 || len(cur.Order) != 400 || len(cur.Cells) != 400 || cur.ImageID != w.ImageID {
		t.Fatalf("current week: %+v %v", cur, err)
	}
	// A below-grade round still reveals cells: every correct answer counts.
	below, _ := e.prog.Start(ctx, e.child, progress.StartRequest{RoundID: "mosaic-r2", Kind: progress.KindSpelling, Focus: "g1-blends", Grade: 1, Count: 5}, day0.Add(time.Hour))
	fin, _ = e.prog.Finish(ctx, e.child, progress.FinishRequest{RoundID: "mosaic-r2", Guesses: winAll(below.Words)}, day0.Add(time.Hour))
	json.Unmarshal(fin.Reward.Mosaic, &w)
	if w.Revealed != 10 || fin.Reward.Fills != 0 {
		t.Fatalf("below-grade round: mosaic %+v fills %d", w, fin.Reward.Fills)
	}
	// Next local week starts a fresh picture.
	next := day0.AddDate(0, 0, 7)
	cur, _ = e.paint.CurrentWeek(ctx, e.child, next)
	if cur.Revealed != 0 || cur.WeekKey != "2026-10-05" {
		t.Fatalf("next week: %+v", cur)
	}
	// Cap: a lot of correct answers never exceed the image.
	for i := 0; i < 90; i++ {
		id := "cap-round-" + strconv.Itoa(i)
		st, err := e.prog.Start(ctx, e.child, progress.StartRequest{RoundID: id, Kind: progress.KindSpelling, Focus: progress.FocusMixed, Grade: 2, Count: 5}, next.Add(time.Duration(i)*time.Minute))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := e.prog.Finish(ctx, e.child, progress.FinishRequest{RoundID: id, Guesses: winAll(st.Words)}, next.Add(time.Duration(i)*time.Minute)); err != nil {
			t.Fatal(err)
		}
	}
	cur, _ = e.paint.CurrentWeek(ctx, e.child, next)
	if cur.Revealed != 400 {
		t.Fatalf("revealed %d, want capped at 400", cur.Revealed)
	}
}

func TestKeyPageFillsEveryRegionSharingTheAnswer(t *testing.T) {
	e := newEnv(t, "UTC")
	ctx := context.Background()
	page, err := e.paint.StartPage(ctx, e.child, StartPageRequest{PageID: "page-0001", Ops: []string{"addsub"}, Grade: 2}, day0)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 14 || len(page.Regions) != 14 || page.Filled != 0 || page.Done {
		t.Fatalf("page: %+v", page)
	}
	for _, r := range page.Regions {
		if r.Answer != "" || r.Filled {
			t.Fatalf("unfilled region leaks its answer: %+v", r)
		}
	}
	again, _ := e.paint.StartPage(ctx, e.child, StartPageRequest{PageID: "page-0001", Ops: []string{"mul"}, Grade: 5}, day0)
	if again.Total != 14 || again.Regions[0].Prompt != page.Regions[0].Prompt {
		t.Fatal("replaying a page id must return the same page")
	}

	// Wrong answer: nothing fills, attempts count.
	res, err := e.paint.Fill(ctx, e.child, FillRequest{PageID: "page-0001", Idx: 0, Answer: "x"}, day0)
	if err != nil || res.Right || len(res.Filled) != 0 || res.Page.Attempts != 1 || res.Page.Filled != 0 {
		t.Fatalf("wrong fill: %+v %v", res, err)
	}
	// Right answer: solve region 0 from the answer key and expect every
	// region with that answer to fill.
	sh, _ := e.sheets.Peek(sheetIDFor(t, e, "page-0001"))
	answers := sh.Answers()
	want := 0
	for _, a := range answers {
		if a == answers[0] {
			want++
		}
	}
	res, err = e.paint.Fill(ctx, e.child, FillRequest{PageID: "page-0001", Idx: 0, Answer: answers[0]}, day0)
	if err != nil || !res.Right || len(res.Filled) != want || res.Page.Filled != want {
		t.Fatalf("right fill: %+v %v (want %d)", res, err, want)
	}
	if !res.Page.Regions[0].Filled || res.Page.Regions[0].Answer != answers[0] || res.Page.Regions[0].Color < 0 {
		t.Fatalf("filled region should show its answer and color: %+v", res.Page.Regions[0])
	}
	// Distinct answers never share a color index; unfilled regions carry none.
	seen := map[int]string{}
	for _, r := range res.Page.Regions {
		if !r.Filled {
			if r.Color != -1 {
				t.Fatalf("unfilled region leaks a color: %+v", r)
			}
			continue
		}
		if prev, ok := seen[r.Color]; ok && prev != r.Answer {
			t.Fatalf("color %d shared by %q and %q", r.Color, prev, r.Answer)
		}
		seen[r.Color] = r.Answer
	}
	// Bad region index and cross-child access.
	if _, err := e.paint.Fill(ctx, e.child, FillRequest{PageID: "page-0001", Idx: 99, Answer: "1"}, day0); !errors.Is(err, ErrBadRequest) {
		t.Fatalf("bad idx: %v", err)
	}
	other := e.child
	other.ChildID = 999
	if _, err := e.paint.Fill(ctx, other, FillRequest{PageID: "page-0001", Idx: 0, Answer: answers[0]}, day0); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-child fill: %v", err)
	}
	// Finish the page: the sheet is released and every answer is shown.
	// Regions already colored by a shared answer are skipped.
	for i, a := range answers {
		if res.Page.Regions[i].Filled {
			continue
		}
		res, err = e.paint.Fill(ctx, e.child, FillRequest{PageID: "page-0001", Idx: i, Answer: a}, day0)
		if err != nil {
			t.Fatal(err)
		}
	}
	if !res.Page.Done || res.Page.Filled != 14 {
		t.Fatalf("page should be done: %+v", res.Page)
	}
	if _, ok := e.sheets.Peek(sheetIDFor(t, e, "page-0001")); ok {
		t.Fatal("finished page should release its sheet")
	}
	// A replay of the finished page (the sheet is gone) still returns it whole.
	replay, err := e.paint.StartPage(ctx, e.child, StartPageRequest{PageID: "page-0001", Ops: []string{"addsub"}, Grade: 2}, day0)
	if err != nil || !replay.Done || replay.Filled != 14 || replay.Regions[3].Answer == "" {
		t.Fatalf("replay after completion: %+v %v", replay, err)
	}
	if _, err := e.paint.Fill(ctx, e.child, FillRequest{PageID: "page-0001", Idx: 0, Answer: "1"}, day0); err != nil {
		t.Fatalf("fill on a finished page should be a no-op, got %v", err)
	}
}

func TestFillGuards(t *testing.T) {
	e := newEnv(t, "UTC")
	ctx := context.Background()
	if _, err := e.paint.StartPage(ctx, e.child, StartPageRequest{PageID: "page-guard", Ops: []string{"addsub"}, Grade: 1}, day0); err != nil {
		t.Fatal(err)
	}
	// Twelve regions allow 96 attempts; the 97th is refused.
	for i := 0; i < 12*MaxAttemptsPerRegion; i++ {
		if _, err := e.paint.Fill(ctx, e.child, FillRequest{PageID: "page-guard", Idx: 0, Answer: "x"}, day0); err != nil {
			t.Fatalf("attempt %d: %v", i, err)
		}
	}
	if _, err := e.paint.Fill(ctx, e.child, FillRequest{PageID: "page-guard", Idx: 0, Answer: "x"}, day0); !errors.Is(err, ErrBadRequest) {
		t.Fatalf("attempt cap: %v", err)
	}
}

func TestMosaicKeepsItsPictureAcrossCatalogChanges(t *testing.T) {
	e := newEnv(t, "UTC")
	ctx := context.Background()
	start, _ := e.prog.Start(ctx, e.child, progress.StartRequest{RoundID: "keep-round-1", Kind: progress.KindSpelling, Focus: "g2-endings", Grade: 2, Count: 5}, day0)
	e.prog.Finish(ctx, e.child, progress.FinishRequest{RoundID: "keep-round-1", Guesses: winAll(start.Words)}, day0)
	before, _ := e.paint.CurrentWeek(ctx, e.child, day0)
	// Simulate a catalog reorder: reverse the slice the picker indexes.
	for i, j := 0, len(e.paint.mosaics)-1; i < j; i, j = i+1, j-1 {
		e.paint.mosaics[i], e.paint.mosaics[j] = e.paint.mosaics[j], e.paint.mosaics[i]
	}
	after, _ := e.paint.CurrentWeek(ctx, e.child, day0)
	if after.ImageID != before.ImageID || after.Revealed != 5 {
		t.Fatalf("picture changed under the child: %s -> %s", before.ImageID, after.ImageID)
	}
}

func sheetIDFor(t *testing.T, e env, pageID string) string {
	t.Helper()
	var id string
	if err := e.paint.db.QueryRow(`SELECT sheet_id FROM page_state WHERE child_id = ? AND page_id = ?`, e.child.ChildID, pageID).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}
