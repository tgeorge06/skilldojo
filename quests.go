package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/tgeorge06/skilldojo/internal/ost"
	"github.com/tgeorge06/skilldojo/internal/progress"
)

// questSources lets the quest picker read missed words and practice-test
// history without depending on the whole server.
type questSources struct {
	prog  *progress.Store
	tests *ost.Store
}

func (q questSources) MissedCount(ctx context.Context, child progress.Child, now time.Time) (int, error) {
	return q.prog.MissedCount(ctx, child, now)
}

func (q questSources) History(ctx context.Context, child ost.Child, limit int) ([]ost.Summary, error) {
	return q.tests.History(ctx, child, limit)
}

// handleQuestsToday returns the active child's three quests for today,
// creating them on the first call.
func (s *server) handleQuestsToday(w http.ResponseWriter, r *http.Request) {
	child, ok := s.activeChild(w, r)
	if !ok {
		return
	}
	day, err := s.quests.Today(r.Context(), child, s.now())
	if err != nil {
		log.Printf("quests: %v", err)
		writeError(w, http.StatusInternalServerError, errors.New("something went wrong"))
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, day)
}
