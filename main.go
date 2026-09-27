// SkillDojo — a tiny, offline-friendly learning app for kids.
package main

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"os"
	"strings"
	"time"
	_ "time/tzdata" // Fly's base image has no zoneinfo; embed it.

	"github.com/tgeorge06/skilldojo/internal/account"
	"github.com/tgeorge06/skilldojo/internal/curriculum"
	"github.com/tgeorge06/skilldojo/internal/db"
	"github.com/tgeorge06/skilldojo/internal/mail"
	"github.com/tgeorge06/skilldojo/internal/progress"
	"github.com/tgeorge06/skilldojo/internal/sheet"
)

//go:embed templates
var templateFS embed.FS

//go:embed static
var staticFS embed.FS

// config is everything main() reads from flags and the environment.
type config struct {
	addr        string
	dbPath      string
	dev         bool // console mail, insecure cookies, magic links echoed on-page
	baseURL     string
	behindProxy bool // trust Fly-Client-IP for rate limiting
	resendKey   string
	mailFrom    string
}

func (c config) validate() error {
	if c.dev {
		return nil
	}
	var problems []string
	if !strings.HasPrefix(c.baseURL, "https://") {
		problems = append(problems, "BASE_URL must start with https://")
	}
	if !strings.HasPrefix(c.resendKey, "re_") {
		problems = append(problems, "RESEND_API_KEY must be a Resend key (re_...)")
	}
	if c.mailFrom == "" || !strings.Contains(c.mailFrom, "@") {
		problems = append(problems, "MAIL_FROM must be an address")
	}
	if len(problems) > 0 {
		return errors.New("config: " + strings.Join(problems, "; ") + " (or run with -dev)")
	}
	return nil
}

type server struct {
	cfg      config
	store    *sheet.Store
	accounts *account.Store
	progress *progress.Store
	mailer   mail.Mailer
	tmpl     *template.Template
	limiter  *rateLimiter
	now      func() time.Time
}

func main() {
	cfg := config{
		baseURL:     strings.TrimRight(os.Getenv("BASE_URL"), "/"),
		behindProxy: os.Getenv("BEHIND_PROXY") == "1",
		resendKey:   os.Getenv("RESEND_API_KEY"),
		mailFrom:    os.Getenv("MAIL_FROM"),
	}
	flag.StringVar(&cfg.addr, "addr", "127.0.0.1:8080", "listen address")
	flag.StringVar(&cfg.dbPath, "db", envOr("DATABASE_PATH", "skilldojo.db"), "SQLite database path")
	flag.BoolVar(&cfg.dev, "dev", false, "development mode: log email instead of sending, allow http cookies")
	flag.Parse()
	if cfg.dev && cfg.baseURL == "" {
		cfg.baseURL = "http://" + cfg.addr
	}
	if err := cfg.validate(); err != nil {
		log.Fatal(err)
	}

	database, err := db.Open(cfg.dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()
	applied, err := db.Migrate(context.Background(), database)
	if err != nil {
		log.Fatal(err)
	}
	if applied > 0 {
		log.Printf("applied %d migration(s) to %s", applied, cfg.dbPath)
	}

	mailer, err := mail.New(cfg.dev, cfg.resendKey, cfg.mailFrom)
	if err != nil {
		log.Fatal(err)
	}

	s, err := newServer(cfg, database, mailer)
	if err != nil {
		log.Fatal(err)
	}
	go s.housekeeping(context.Background())

	srv := &http.Server{
		Addr:              cfg.addr,
		Handler:           s.handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
	}
	log.Printf("SkillDojo listening on http://%s", cfg.addr)
	log.Fatal(srv.ListenAndServe())
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// newServer wires the handlers. Tests call it with a temp database and a
// capturing mailer.
func newServer(cfg config, database *sql.DB, mailer mail.Mailer) (*server, error) {
	// Fail fast if the embedded curriculum is malformed; the server grades
	// spelling rounds against it.
	cur, err := curriculum.Load()
	if err != nil {
		return nil, err
	}
	tmpl, err := template.ParseFS(templateFS, "templates/*.html")
	if err != nil {
		return nil, err
	}
	sheets := sheet.NewStore()
	return &server{
		cfg:      cfg,
		store:    sheets,
		accounts: account.New(database),
		progress: progress.New(database, cur, sheets),
		mailer:   mailer,
		tmpl:     tmpl,
		limiter:  newRateLimiter(),
		now:      time.Now,
	}, nil
}

func (s *server) handler() http.Handler {
	staticFiles, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatal(err)
	}
	if err := mime.AddExtensionType(".opus", "audio/ogg"); err != nil {
		log.Printf("register Opus MIME type: %v", err)
	}
	staticHandler := http.StripPrefix("/static/", http.FileServerFS(staticFiles))
	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Audio-set directories are versioned, so clips can be cached forever.
		if strings.HasSuffix(r.URL.Path, ".opus") && strings.Contains(r.URL.Path, "/audio/spelling/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		staticHandler.ServeHTTP(w, r)
	}))
	mux.HandleFunc("GET /{$}", s.handleIndex)
	mux.HandleFunc("POST /api/sheet", s.handleNewSheet)
	mux.HandleFunc("POST /api/grade", s.handleGrade)
	mux.HandleFunc("POST /api/round/start", s.handleRoundStart)
	mux.HandleFunc("POST /api/round/finish", s.handleRoundFinish)

	mux.HandleFunc("GET /login", s.handleLoginPage)
	mux.HandleFunc("POST /auth/magic", s.handleMagic)
	mux.HandleFunc("GET /auth/verify", s.handleVerifyPage)
	mux.HandleFunc("POST /auth/verify", s.handleVerify)
	mux.HandleFunc("POST /auth/logout", s.handleLogout)
	mux.HandleFunc("GET /family", s.handleFamily)
	mux.HandleFunc("POST /family/children", s.handleCreateChild)
	mux.HandleFunc("POST /family/children/{id}", s.handleChildAction)

	// Reject cross-origin form posts (Sec-Fetch-Site / Origin based), which
	// with SameSite=Lax cookies is the CSRF defence for every POST above.
	csrf := http.NewCrossOriginProtection()
	return csrf.Handler(mux)
}

// housekeeping purges expired tokens and sessions hourly.
func (s *server) housekeeping(ctx context.Context) {
	t := time.NewTicker(time.Hour)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := s.accounts.PurgeExpired(ctx, s.now()); err != nil {
				log.Printf("purge expired: %v", err)
			}
			s.limiter.purge(s.now())
		}
	}
}

// indexData is what the practice page needs to know about the signed-in
// parent. Nicknames reach Alpine through data attributes, never x-data.
type indexData struct {
	SignedIn    bool
	ActiveChild *account.Child
	Children    []account.Child
}

func (s *server) handleIndex(w http.ResponseWriter, r *http.Request) {
	data := indexData{}
	if sess, ok := s.currentSession(r); ok {
		data.SignedIn = true
		if kids, err := s.accounts.Children(r.Context(), sess.AccountID); err == nil {
			data.Children = kids
			for i := range kids {
				if kids[i].ID == sess.ActiveChildID {
					data.ActiveChild = &kids[i]
				}
			}
		}
	}
	s.render(w, "index.html", data)
}

func (s *server) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("render %s: %v", name, err)
	}
}

type newSheetRequest struct {
	Ops   []string `json:"ops"`
	Grade int      `json:"grade"`
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
	sh, err := sheet.Generate(req.Ops, req.Grade, req.Count)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.store.Put(sh); err != nil {
		writeError(w, http.StatusTooManyRequests, err)
		return
	}
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
	return decodeJSONLimit(w, r, v, 64<<10)
}

// decodeJSONLimit is strict: a byte cap per endpoint, no unknown fields,
// no trailing data.
func decodeJSONLimit(w http.ResponseWriter, r *http.Request, v any, limit int64) error {
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("bad request body: %w", err))
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		err = fmt.Errorf("bad request body: trailing data")
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
