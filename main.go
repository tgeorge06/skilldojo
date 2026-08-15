// SkillDojo — a tiny practice-sheet app. Math is the first dojo.
package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"time"

	"github.com/tgeorge06/skilldojo/internal/sheet"
)

//go:embed templates/index.html
var templateFS embed.FS

//go:embed static
var staticFS embed.FS

type server struct {
	store *sheet.Store
	tmpl  *template.Template
}

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "listen address")
	flag.Parse()

	s := &server{
		store: sheet.NewStore(),
		tmpl:  template.Must(template.ParseFS(templateFS, "templates/index.html")),
	}

	staticFiles, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(staticFiles)))
	mux.HandleFunc("GET /{$}", s.handleIndex)
	mux.HandleFunc("POST /api/sheet", s.handleNewSheet)
	mux.HandleFunc("POST /api/grade", s.handleGrade)

	srv := &http.Server{
		Addr:              *addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
	}
	log.Printf("SkillDojo listening on http://%s", *addr)
	log.Fatal(srv.ListenAndServe())
}

func (s *server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.Execute(w, nil); err != nil {
		log.Printf("render: %v", err)
	}
}

type newSheetRequest struct {
	Ops   []string `json:"ops"`
	Belt  string   `json:"belt"`
	Count int      `json:"count"`
}

type newSheetResponse struct {
	ID        string           `json:"id"`
	Questions []sheet.Question `json:"questions"`
}

func (s *server) handleNewSheet(w http.ResponseWriter, r *http.Request) {
	var req newSheetRequest
	if err := decodeJSON(w, r, &req); err != nil {
		return
	}
	sh, err := sheet.Generate(req.Ops, req.Belt, req.Count)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s.store.Put(sh)
	writeJSON(w, newSheetResponse{ID: sh.ID, Questions: sh.Questions})
}

type gradeRequest struct {
	ID      string   `json:"id"`
	Answers []string `json:"answers"`
}

type gradeResponse struct {
	Results []sheet.Result `json:"results"`
	Score   int            `json:"score"`
	Total   int            `json:"total"`
	Percent int            `json:"percent"`
}

func (s *server) handleGrade(w http.ResponseWriter, r *http.Request) {
	var req gradeRequest
	if err := decodeJSON(w, r, &req); err != nil {
		return
	}
	results, err := s.store.Grade(req.ID, req.Answers)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	score := 0
	for _, res := range results {
		if res.Right {
			score++
		}
	}
	percent := 0
	if len(results) > 0 {
		percent = score * 100 / len(results)
	}
	writeJSON(w, gradeResponse{Results: results, Score: score, Total: len(results), Percent: percent})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("bad request body: %w", err))
		return err
	}
	if dec.More() {
		err := fmt.Errorf("bad request body: trailing data")
		writeError(w, http.StatusBadRequest, err)
		return err
	}
	return nil
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("encode: %v", err)
	}
}

func writeError(w http.ResponseWriter, code int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()}) //nolint:errcheck
}
