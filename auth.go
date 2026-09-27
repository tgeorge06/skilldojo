package main

import (
	"errors"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/tgeorge06/skilldojo/internal/account"
	"github.com/tgeorge06/skilldojo/internal/mail"
)

const sessionCookie = "sd_session"

// Form bodies are tiny; cap them well below the JSON limit.
const formBodyLimit = 4 << 10

// rateLimiter is a fixed-window counter keyed by string (email or IP).
type rateLimiter struct {
	mu   sync.Mutex
	hits map[string][]time.Time
}

func newRateLimiter() *rateLimiter { return &rateLimiter{hits: map[string][]time.Time{}} }

func (l *rateLimiter) allow(key string, limit int, window time.Duration, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	cutoff := now.Add(-window)
	kept := l.hits[key][:0]
	for _, t := range l.hits[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= limit {
		l.hits[key] = kept
		return false
	}
	l.hits[key] = append(kept, now)
	// Drop empty keys occasionally so the map cannot grow without bound.
	if len(l.hits) > 10000 {
		for k, v := range l.hits {
			if len(v) == 0 || !v[len(v)-1].After(cutoff) {
				delete(l.hits, k)
			}
		}
	}
	return true
}

func (s *server) clientIP(r *http.Request) string {
	if s.cfg.behindProxy {
		if ip := strings.TrimSpace(r.Header.Get("Fly-Client-IP")); ip != "" {
			return ip
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// currentSession resolves the cookie to a live session, if any.
func (s *server) currentSession(r *http.Request) (account.Session, bool) {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return account.Session{}, false
	}
	sess, err := s.accounts.SessionByToken(r.Context(), c.Value, s.now())
	if err != nil {
		return account.Session{}, false
	}
	return sess, true
}

func (s *server) setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		MaxAge:   int(account.SessionTTL / time.Second),
		HttpOnly: true,
		Secure:   !s.cfg.dev,
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *server) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: !s.cfg.dev, SameSite: http.SameSiteLaxMode,
	})
}

type loginData struct {
	Error string
}

type sentData struct {
	Email   string
	DevLink string // only populated with -dev so a developer can click through
}

func (s *server) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.currentSession(r); ok {
		http.Redirect(w, r, "/family", http.StatusSeeOther)
		return
	}
	s.render(w, "login.html", loginData{})
}

// handleMagic creates the account on first sight and emails a single-use
// link. The response is identical whether or not the address was known, and
// identical when rate-limited, so it cannot be used to probe for accounts.
func (s *server) handleMagic(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, formBodyLimit)
	if err := r.ParseForm(); err != nil {
		s.render(w, "login.html", loginData{Error: "That form did not come through. Please try again."})
		return
	}
	email, err := account.NormalizeEmail(r.PostFormValue("email"))
	if err != nil {
		s.render(w, "login.html", loginData{Error: "Please enter a valid email address."})
		return
	}
	now := s.now()
	sent := sentData{Email: email}
	if !s.limiter.allow("email:"+email, 3, 15*time.Minute, now) ||
		!s.limiter.allow("ip:"+s.clientIP(r), 10, 15*time.Minute, now) {
		s.render(w, "sent.html", sent)
		return
	}

	acct, err := s.accounts.EnsureAccount(r.Context(), email, r.PostFormValue("tz"), now)
	if err != nil {
		log.Printf("ensure account: %v", err)
		s.render(w, "login.html", loginData{Error: "Something went wrong on our side. Please try again."})
		return
	}
	token, err := s.accounts.CreateLoginToken(r.Context(), acct.ID, now)
	if err != nil {
		log.Printf("create login token: %v", err)
		s.render(w, "login.html", loginData{Error: "Something went wrong on our side. Please try again."})
		return
	}
	link := s.cfg.baseURL + "/auth/verify?t=" + token
	msg := mail.Message{
		To:      email,
		Subject: "Your SkillDojo sign-in link",
		Text: "Tap this link to sign in to SkillDojo. It works once and expires in 15 minutes.\n\n" +
			link + "\n\nIf you did not ask for this, you can ignore it.",
		HTML: `<p>Tap this link to sign in to SkillDojo. It works once and expires in 15 minutes.</p>` +
			`<p><a href="` + link + `">Sign in to SkillDojo</a></p>` +
			`<p>If you did not ask for this, you can ignore it.</p>`,
	}
	if err := s.mailer.Send(r.Context(), msg); err != nil {
		log.Printf("send magic link: %v", err)
		s.render(w, "login.html", loginData{Error: "We could not send the email. Please try again in a minute."})
		return
	}
	if s.cfg.dev {
		sent.DevLink = link
	}
	s.render(w, "sent.html", sent)
}

func (s *server) handleVerify(w http.ResponseWriter, r *http.Request) {
	now := s.now()
	accountID, err := s.accounts.ConsumeLoginToken(r.Context(), r.URL.Query().Get("t"), now)
	if err != nil {
		if !errors.Is(err, account.ErrInvalidToken) {
			log.Printf("consume login token: %v", err)
		}
		w.WriteHeader(http.StatusBadRequest)
		s.render(w, "login.html", loginData{Error: "That sign-in link has expired or was already used. Request a new one."})
		return
	}
	token, err := s.accounts.CreateSession(r.Context(), accountID, now)
	if err != nil {
		log.Printf("create session: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		s.render(w, "login.html", loginData{Error: "Something went wrong on our side. Please try again."})
		return
	}
	s.setSessionCookie(w, token)
	http.Redirect(w, r, "/family", http.StatusSeeOther)
}

func (s *server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		if err := s.accounts.DeleteSession(r.Context(), c.Value); err != nil {
			log.Printf("delete session: %v", err)
		}
	}
	s.clearSessionCookie(w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
