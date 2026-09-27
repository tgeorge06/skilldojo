package main

import (
	"errors"
	"log"
	"net/http"

	"github.com/tgeorge06/skilldojo/internal/account"
	"github.com/tgeorge06/skilldojo/internal/progress"
)

// activeChild resolves the session's selected child for round endpoints.
// Without a session or a selected child the caller gets a JSON 401; the
// anonymous practice endpoints are untouched by this.
func (s *server) activeChild(w http.ResponseWriter, r *http.Request) (progress.Child, bool) {
	sess, ok := s.currentSession(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, errors.New("sign in and choose who is training first"))
		return progress.Child{}, false
	}
	if sess.ActiveChildID == 0 {
		writeError(w, http.StatusUnauthorized, errors.New("choose who is training first"))
		return progress.Child{}, false
	}
	child, err := s.accounts.Child(r.Context(), sess.AccountID, sess.ActiveChildID)
	if err != nil {
		if !errors.Is(err, account.ErrNotFound) {
			log.Printf("load child: %v", err)
		}
		writeError(w, http.StatusUnauthorized, errors.New("choose who is training first"))
		return progress.Child{}, false
	}
	acct, err := s.accounts.AccountByID(r.Context(), sess.AccountID)
	if err != nil {
		log.Printf("load account: %v", err)
		writeError(w, http.StatusInternalServerError, errors.New("something went wrong"))
		return progress.Child{}, false
	}
	return progress.Child{AccountID: acct.ID, ChildID: child.ID, Grade: child.Grade, Timezone: acct.Timezone}, true
}

func (s *server) handleRoundStart(w http.ResponseWriter, r *http.Request) {
	child, ok := s.activeChild(w, r)
	if !ok {
		return
	}
	var req progress.StartRequest
	if err := decodeJSONLimit(w, r, &req, 2<<10); err != nil {
		return
	}
	resp, err := s.progress.Start(r.Context(), child, req, s.now())
	if err != nil {
		writeProgressError(w, err)
		return
	}
	writeJSON(w, resp)
}

func (s *server) handleRoundFinish(w http.ResponseWriter, r *http.Request) {
	child, ok := s.activeChild(w, r)
	if !ok {
		return
	}
	var req progress.FinishRequest
	if err := decodeJSONLimit(w, r, &req, 16<<10); err != nil {
		return
	}
	resp, err := s.progress.Finish(r.Context(), child, req, s.now())
	if err != nil {
		writeProgressError(w, err)
		return
	}
	writeJSON(w, resp)
}

func writeProgressError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, progress.ErrNotFound):
		writeError(w, http.StatusNotFound, errors.New("round not found"))
	case errors.Is(err, progress.ErrBadRequest):
		writeError(w, http.StatusBadRequest, err)
	default:
		log.Printf("round: %v", err)
		writeError(w, http.StatusInternalServerError, errors.New("something went wrong"))
	}
}

// handleKataIndex returns the roster merged with the active child's state.
func (s *server) handleKataIndex(w http.ResponseWriter, r *http.Request) {
	child, ok := s.activeChild(w, r)
	if !ok {
		return
	}
	now := s.now()
	prog, err := s.progress.Progress(r.Context(), child, now)
	if err != nil {
		writeProgressError(w, err)
		return
	}
	missed, err := s.progress.MissedCount(r.Context(), child, now)
	if err != nil {
		writeProgressError(w, err)
		return
	}
	idx, err := s.kata.Index(r.Context(), child, prog, missed, s.cur)
	if err != nil {
		writeProgressError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, idx)
}
