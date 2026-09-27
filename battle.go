package main

import (
	"errors"
	"log"
	"net/http"

	"github.com/tgeorge06/skilldojo/internal/battle"
)

func (s *server) battlesEnabled(w http.ResponseWriter) bool {
	if s.cfg.battlesDisabled {
		writeError(w, http.StatusNotFound, errors.New("battles are not available right now"))
		return false
	}
	return true
}

func (s *server) handleBattleCredits(w http.ResponseWriter, r *http.Request) {
	child, ok := s.activeChild(w, r)
	if !ok {
		return
	}
	n, err := s.battles.Credits(r.Context(), child)
	if err != nil {
		writeBattleError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, map[string]any{"credits": n, "enabled": !s.cfg.battlesDisabled})
}

func (s *server) handleBattleStart(w http.ResponseWriter, r *http.Request) {
	child, ok := s.activeChild(w, r)
	if !ok || !s.battlesEnabled(w) {
		return
	}
	var req battle.StartRequest
	if err := decodeJSONLimit(w, r, &req, 1<<10); err != nil {
		return
	}
	st, err := s.battles.Start(r.Context(), child, req, s.now())
	if err != nil {
		writeBattleError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, st)
}

func (s *server) handleBattleTurn(w http.ResponseWriter, r *http.Request) {
	child, ok := s.activeChild(w, r)
	if !ok || !s.battlesEnabled(w) {
		return
	}
	var req battle.TurnRequest
	if err := decodeJSONLimit(w, r, &req, 1<<10); err != nil {
		return
	}
	st, err := s.battles.Play(r.Context(), child, req, s.now())
	if err != nil {
		writeBattleError(w, err)
		return
	}
	writeJSON(w, st)
}

func writeBattleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, battle.ErrNotFound):
		writeError(w, http.StatusNotFound, errors.New("battle not found"))
	case errors.Is(err, battle.ErrNoCredit):
		writeError(w, http.StatusConflict, err)
	case errors.Is(err, battle.ErrBadRequest):
		writeError(w, http.StatusBadRequest, err)
	default:
		log.Printf("battle: %v", err)
		writeError(w, http.StatusInternalServerError, errors.New("something went wrong"))
	}
}
