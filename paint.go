package main

import (
	"errors"
	"log"
	"net/http"

	"github.com/tgeorge06/skilldojo/internal/paint"
)

func (s *server) handleMosaicWeek(w http.ResponseWriter, r *http.Request) {
	child, ok := s.activeChild(w, r)
	if !ok {
		return
	}
	week, err := s.paint.CurrentWeek(r.Context(), child, s.now())
	if err != nil {
		writePaintError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, week)
}

func (s *server) handlePageStart(w http.ResponseWriter, r *http.Request) {
	child, ok := s.activeChild(w, r)
	if !ok {
		return
	}
	var req paint.StartPageRequest
	if err := decodeJSONLimit(w, r, &req, 1<<10); err != nil {
		return
	}
	page, err := s.paint.StartPage(r.Context(), child, req, s.now())
	if err != nil {
		writePaintError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, page)
}

func (s *server) handlePageFill(w http.ResponseWriter, r *http.Request) {
	child, ok := s.activeChild(w, r)
	if !ok {
		return
	}
	var req paint.FillRequest
	if err := decodeJSONLimit(w, r, &req, 1<<10); err != nil {
		return
	}
	resp, err := s.paint.Fill(r.Context(), child, req, s.now())
	if err != nil {
		writePaintError(w, err)
		return
	}
	writeJSON(w, resp)
}

func writePaintError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, paint.ErrNotFound):
		writeError(w, http.StatusNotFound, errors.New("page not found"))
	case errors.Is(err, paint.ErrBadRequest):
		writeError(w, http.StatusBadRequest, err)
	default:
		log.Printf("paint: %v", err)
		writeError(w, http.StatusInternalServerError, errors.New("something went wrong"))
	}
}
