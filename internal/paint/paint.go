// Package paint is color by number: a weekly pixel mosaic that fills one
// cell per correct answer, and math key pages where solving a region's
// problem colors every region with that answer.
package paint

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/tgeorge06/skilldojo/internal/progress"
	"github.com/tgeorge06/skilldojo/internal/sheet"
)

//go:embed mosaics.json
var mosaicsJSON []byte

var (
	ErrNotFound   = errors.New("paint: not found")
	ErrBadRequest = errors.New("paint: bad request")
)

// Mosaic is one week's picture: a size x size grid of palette indexes.
type Mosaic struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Size    int      `json:"size"`
	Palette []string `json:"palette"`
	Cells   string   `json:"cells"` // size*size digits, 0 = background
}

type mosaicFile struct {
	Mosaics []Mosaic `json:"mosaics"`
}

// Store applies mosaic rewards and serves pages.
type Store struct {
	db      *sql.DB
	sheets  *sheet.Store
	mosaics []Mosaic
	byID    map[string]Mosaic
}

// New loads the embedded mosaics and wraps the database.
func New(db *sql.DB, sheets *sheet.Store) (*Store, error) {
	var f mosaicFile
	if err := json.Unmarshal(mosaicsJSON, &f); err != nil {
		return nil, fmt.Errorf("paint: parse mosaics: %w", err)
	}
	if len(f.Mosaics) == 0 {
		return nil, errors.New("paint: no mosaics")
	}
	s := &Store{db: db, sheets: sheets, mosaics: f.Mosaics, byID: map[string]Mosaic{}}
	for _, m := range f.Mosaics {
		if m.Size < 8 || m.Size > 40 || len(m.Cells) != m.Size*m.Size || len(m.Palette) < 2 {
			return nil, fmt.Errorf("paint: mosaic %q is malformed", m.ID)
		}
		for _, ch := range m.Cells {
			if ch < '0' || int(ch-'0') >= len(m.Palette) {
				return nil, fmt.Errorf("paint: mosaic %q has a cell outside its palette", m.ID)
			}
		}
		if _, dup := s.byID[m.ID]; dup {
			return nil, fmt.Errorf("paint: duplicate mosaic %q", m.ID)
		}
		s.byID[m.ID] = m
	}
	return s, nil
}

const tsLayout = "2006-01-02T15:04:05.000000000Z"

func ts(t time.Time) string { return t.UTC().Format(tsLayout) }

// WeekKey is the Monday of the child's local week, as YYYY-MM-DD.
func WeekKey(now time.Time, tz string) string {
	day, _ := time.Parse("2006-01-02", progress.LocalDate(now, tz))
	offset := (int(day.Weekday()) + 6) % 7 // Monday = 0
	return day.AddDate(0, 0, -offset).Format("2006-01-02")
}

// imageFor picks the week's picture deterministically per child and week
// so siblings and weeks vary without storing a choice up front.
func (s *Store) imageFor(childID int64, weekKey string) Mosaic {
	h := fnv.New32a()
	fmt.Fprintf(h, "%d:%s", childID, weekKey)
	return s.mosaics[int(h.Sum32()%uint32(len(s.mosaics)))]
}

// RevealOrder is the seeded order cells appear in for a week, so the
// picture emerges scattered rather than top-down. Same input, same order.
func RevealOrder(size int, weekKey string, childID int64) []int {
	h := fnv.New64a()
	fmt.Fprintf(h, "%d:%s", childID, weekKey)
	r := rand.New(rand.NewPCG(h.Sum64(), 0x5eed))
	order := make([]int, size*size)
	for i := range order {
		order[i] = i
	}
	r.Shuffle(len(order), func(i, j int) { order[i], order[j] = order[j], order[i] })
	return order
}

// Week is the mosaic as the client renders it.
type Week struct {
	WeekKey  string   `json:"week_key"`
	ImageID  string   `json:"image_id"`
	Name     string   `json:"name"`
	Size     int      `json:"size"`
	Palette  []string `json:"palette"`
	Cells    string   `json:"cells"`
	Revealed int      `json:"revealed"`
	Total    int      `json:"total"`
	Order    []int    `json:"order"`
	Added    int      `json:"added,omitempty"`
}

// Apply is the progress.Sink for mosaics: every mosaic cell earned reveals
// one more cell of this week's picture, capped at the picture size.
func (s *Store) Apply(ctx context.Context, tx *sql.Tx, child progress.Child, _ []string, reward *progress.Reward, now time.Time) error {
	if reward.MosaicCells <= 0 {
		return nil
	}
	week := WeekKey(now, child.Timezone)
	img := s.imageFor(child.ChildID, week)
	total := img.Size * img.Size
	var cells int
	err := tx.QueryRowContext(ctx,
		`SELECT cells FROM mosaic_weeks WHERE child_id = ? AND account_id = ? AND week_key = ?`,
		child.ChildID, child.AccountID, week).Scan(&cells)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	added := min(reward.MosaicCells, total-cells)
	cells += added
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO mosaic_weeks (account_id, child_id, week_key, image_id, cells, updated_at) VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(child_id, week_key) DO UPDATE SET cells = excluded.cells, updated_at = excluded.updated_at`,
		child.AccountID, child.ChildID, week, img.ID, cells, ts(now)); err != nil {
		return err
	}
	w := Week{WeekKey: week, ImageID: img.ID, Name: img.Name, Size: img.Size, Revealed: cells, Total: total, Added: added}
	encoded, err := json.Marshal(w)
	if err != nil {
		return err
	}
	reward.Mosaic = encoded
	return nil
}

// CurrentWeek returns this week's mosaic with everything the client needs
// to draw it.
func (s *Store) CurrentWeek(ctx context.Context, child progress.Child, now time.Time) (Week, error) {
	week := WeekKey(now, child.Timezone)
	img := s.imageFor(child.ChildID, week)
	var cells int
	err := s.db.QueryRowContext(ctx,
		`SELECT cells FROM mosaic_weeks WHERE child_id = ? AND account_id = ? AND week_key = ?`,
		child.ChildID, child.AccountID, week).Scan(&cells)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Week{}, err
	}
	return Week{
		WeekKey: week, ImageID: img.ID, Name: img.Name, Size: img.Size, Palette: img.Palette, Cells: img.Cells,
		Revealed: cells, Total: img.Size * img.Size, Order: RevealOrder(img.Size, week, child.ChildID),
	}, nil
}

// ---- Math key pages ----

// pageRegions sizes a page to one session by grade.
func pageRegions(grade int) int {
	return []int{0, 12, 14, 18, 20, 24}[grade]
}

// Region is one area of the page with its problem. The answer stays on
// the server until the region is filled.
type Region struct {
	Idx    int    `json:"idx"`
	Prompt string `json:"prompt"`
	Op     string `json:"op"`
	Filled bool   `json:"filled"`
	Answer string `json:"answer,omitempty"` // only once filled
}

// Page is a key page as the client renders it.
type Page struct {
	PageID   string   `json:"page_id"`
	Grade    int      `json:"grade"`
	Seed     int      `json:"seed"`
	Regions  []Region `json:"regions"`
	Filled   int      `json:"filled"`
	Total    int      `json:"total"`
	Attempts int      `json:"attempts"`
	Done     bool     `json:"done"`
}

// StartPageRequest opens (or replays) a page.
type StartPageRequest struct {
	PageID string   `json:"page_id"`
	Ops    []string `json:"ops"`
	Grade  int      `json:"grade"`
}

func validID(id string) bool {
	if len(id) < 8 || len(id) > 64 {
		return false
	}
	for _, r := range id {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_') {
			return false
		}
	}
	return true
}

// StartPage generates a page of problems and stores its sheet; replaying
// the same page id returns the page as it stands.
func (s *Store) StartPage(ctx context.Context, child progress.Child, req StartPageRequest, now time.Time) (Page, error) {
	if !validID(req.PageID) {
		return Page{}, fmt.Errorf("%w: page_id must be 8-64 url-safe characters", ErrBadRequest)
	}
	if req.Grade < 1 || req.Grade > 5 {
		return Page{}, fmt.Errorf("%w: grade must be 1-5", ErrBadRequest)
	}
	if existing, err := s.page(ctx, child, req.PageID); err == nil {
		return existing, nil
	} else if !errors.Is(err, ErrNotFound) {
		return Page{}, err
	}
	n := pageRegions(req.Grade)
	sh, err := sheet.GenerateCount(req.Ops, req.Grade, n)
	if err != nil {
		return Page{}, fmt.Errorf("%w: %v", ErrBadRequest, err)
	}
	seed := int(fnv32(child.ChildID, req.PageID))
	if err := s.sheets.Put(sh); err != nil {
		return Page{}, err
	}
	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO page_state (account_id, child_id, page_id, sheet_id, grade, ops, seed, regions, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		child.AccountID, child.ChildID, req.PageID, sh.ID, req.Grade, strings.Join(req.Ops, ","), seed, n, ts(now), ts(now)); err != nil {
		s.sheets.Remove(sh.ID)
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return s.page(ctx, child, req.PageID)
		}
		return Page{}, err
	}
	return s.page(ctx, child, req.PageID)
}

func fnv32(childID int64, id string) uint32 {
	h := fnv.New32a()
	fmt.Fprintf(h, "%d:%s", childID, id)
	return h.Sum32() % 1_000_000
}

// page loads the page and merges the sheet's prompts with the fill mask.
func (s *Store) page(ctx context.Context, child progress.Child, pageID string) (Page, error) {
	var p Page
	var sheetID string
	var mask int64
	err := s.db.QueryRowContext(ctx,
		`SELECT sheet_id, grade, seed, regions, filled_mask, attempts FROM page_state
		 WHERE child_id = ? AND account_id = ? AND page_id = ?`, child.ChildID, child.AccountID, pageID).
		Scan(&sheetID, &p.Grade, &p.Seed, &p.Total, &mask, &p.Attempts)
	if errors.Is(err, sql.ErrNoRows) {
		return Page{}, ErrNotFound
	}
	if err != nil {
		return Page{}, err
	}
	sh, ok := s.sheets.Peek(sheetID)
	if !ok {
		return Page{}, fmt.Errorf("%w: this page has expired, start a new one", ErrBadRequest)
	}
	p.PageID = pageID
	answers := sh.Answers()
	for i, q := range sh.Questions {
		r := Region{Idx: i, Prompt: q.Prompt, Op: q.Op, Filled: mask&(1<<i) != 0}
		if r.Filled {
			r.Answer = answers[i]
			p.Filled++
		}
		p.Regions = append(p.Regions, r)
	}
	p.Done = p.Filled == p.Total
	return p, nil
}

// FillRequest is one attempt at one region.
type FillRequest struct {
	PageID string `json:"page_id"`
	Idx    int    `json:"idx"`
	Answer string `json:"answer"`
}

// FillResponse reports whether the attempt was right and the page after it.
type FillResponse struct {
	Right  bool  `json:"right"`
	Filled []int `json:"filled"` // regions colored by this attempt
	Page   Page  `json:"page"`
}

// Fill grades one region. A right answer colors that region and every
// region sharing the answer; a wrong one changes nothing but the attempt
// count. Wrong answers never touch mastery: this mode is for calm.
func (s *Store) Fill(ctx context.Context, child progress.Child, req FillRequest, now time.Time) (FillResponse, error) {
	if !validID(req.PageID) {
		return FillResponse{}, fmt.Errorf("%w: bad page_id", ErrBadRequest)
	}
	if len(req.Answer) > 32 {
		return FillResponse{}, fmt.Errorf("%w: answer too long", ErrBadRequest)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FillResponse{}, err
	}
	defer tx.Rollback()
	var sheetID string
	var regions int
	var mask int64
	err = tx.QueryRowContext(ctx,
		`SELECT sheet_id, regions, filled_mask FROM page_state WHERE child_id = ? AND account_id = ? AND page_id = ?`,
		child.ChildID, child.AccountID, req.PageID).Scan(&sheetID, &regions, &mask)
	if errors.Is(err, sql.ErrNoRows) {
		return FillResponse{}, ErrNotFound
	}
	if err != nil {
		return FillResponse{}, err
	}
	if req.Idx < 0 || req.Idx >= regions {
		return FillResponse{}, fmt.Errorf("%w: no such region", ErrBadRequest)
	}
	sh, ok := s.sheets.Peek(sheetID)
	if !ok {
		return FillResponse{}, fmt.Errorf("%w: this page has expired, start a new one", ErrBadRequest)
	}
	answers := make([]string, len(sh.Questions))
	answers[req.Idx] = req.Answer
	results, err := sh.Check(answers)
	if err != nil {
		return FillResponse{}, fmt.Errorf("%w: %v", ErrBadRequest, err)
	}
	resp := FillResponse{Right: results[req.Idx].Right, Filled: []int{}}
	if resp.Right {
		target := sh.Answers()[req.Idx]
		for i, a := range sh.Answers() {
			if a == target && mask&(1<<i) == 0 {
				mask |= 1 << i
				resp.Filled = append(resp.Filled, i)
			}
		}
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE page_state SET filled_mask = ?, attempts = attempts + 1, updated_at = ?
		 WHERE child_id = ? AND account_id = ? AND page_id = ?`,
		mask, ts(now), child.ChildID, child.AccountID, req.PageID); err != nil {
		return FillResponse{}, err
	}
	if err := tx.Commit(); err != nil {
		return FillResponse{}, err
	}
	resp.Page, err = s.page(ctx, child, req.PageID)
	if err != nil {
		return FillResponse{}, err
	}
	if resp.Page.Done {
		s.sheets.Remove(sheetID)
		resp.Page.Regions = pageWithAnswers(resp.Page, sh)
	}
	return resp, nil
}

func pageWithAnswers(p Page, sh *sheet.Sheet) []Region {
	answers := sh.Answers()
	for i := range p.Regions {
		p.Regions[i].Filled = true
		p.Regions[i].Answer = answers[i]
	}
	return p.Regions
}
