package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tgeorge06/skilldojo/internal/db"
	"github.com/tgeorge06/skilldojo/internal/mail"
)

// captureMailer records outbound mail so tests can pull the magic link.
type captureMailer struct {
	mu   sync.Mutex
	sent []mail.Message
}

func (c *captureMailer) Send(_ context.Context, m mail.Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sent = append(c.sent, m)
	return nil
}

func (c *captureMailer) last() mail.Message {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.sent) == 0 {
		return mail.Message{}
	}
	return c.sent[len(c.sent)-1]
}

type testEnv struct {
	srv    *httptest.Server
	mailer *captureMailer
	s      *server
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	d, err := db.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	if _, err := db.Migrate(context.Background(), d); err != nil {
		t.Fatal(err)
	}
	mailer := &captureMailer{}
	s, err := newServer(config{dev: true, baseURL: "http://placeholder"}, d, mailer)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(s.handler())
	t.Cleanup(srv.Close)
	s.cfg.baseURL = srv.URL
	return &testEnv{srv: srv, mailer: mailer, s: s}
}

// client returns an HTTP client with its own cookie jar that does not follow
// redirects, so tests can assert on 303s.
func (e *testEnv) client(t *testing.T) *http.Client {
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Client{
		Jar:           jar,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

func (e *testEnv) postForm(t *testing.T, c *http.Client, path string, form url.Values, headers ...string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, e.srv.URL+path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	res, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

// snippet truncates a body for failure messages without panicking on
// short bodies.
func snippet(s string) string {
	if len(s) > 200 {
		return s[:200]
	}
	return s
}

func body(t *testing.T, res *http.Response) string {
	t.Helper()
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

var linkRE = regexp.MustCompile(`/auth/verify\?t=[A-Za-z0-9_-]+`)

// signIn runs the magic-link flow for email and returns a signed-in client.
func (e *testEnv) signIn(t *testing.T, email string) *http.Client {
	t.Helper()
	c := e.client(t)
	res := e.postForm(t, c, "/auth/magic", url.Values{"email": {email}, "tz": {"America/New_York"}})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("magic: %d", res.StatusCode)
	}
	page := body(t, res)
	if !strings.Contains(page, strings.ToLower(email)) {
		t.Fatalf("sent page should echo the address: %s", page)
	}
	link := linkRE.FindString(e.mailer.last().Text)
	if link == "" {
		t.Fatalf("no link in mail: %q", e.mailer.last().Text)
	}
	// GET only confirms; it must not consume the token.
	res, err := c.Get(e.srv.URL + link)
	if err != nil {
		t.Fatal(err)
	}
	if page := body(t, res); res.StatusCode != http.StatusOK || !strings.Contains(page, "Sign in as "+strings.ToLower(email)) {
		t.Fatalf("verify page: %d %s", res.StatusCode, snippet(page))
	}
	res, _ = c.Get(e.srv.URL + link)
	if body(t, res); res.StatusCode != http.StatusOK {
		t.Fatalf("second GET of the link should still confirm, got %d", res.StatusCode)
	}
	token := strings.TrimPrefix(link, "/auth/verify?t=")
	res = e.postForm(t, c, "/auth/verify", url.Values{"t": {token}})
	body(t, res)
	if res.StatusCode != http.StatusSeeOther || res.Header.Get("Location") != "/family" {
		t.Fatalf("verify: %d -> %q", res.StatusCode, res.Header.Get("Location"))
	}
	return c
}

func TestMagicLinkSignInFlow(t *testing.T) {
	e := newTestEnv(t)
	c := e.signIn(t, "Parent@Example.com")

	// The link is single use: both the page and the redeem refuse it now.
	link := linkRE.FindString(e.mailer.last().Text)
	res, _ := c.Get(e.srv.URL + link)
	if page := body(t, res); res.StatusCode != http.StatusBadRequest || !strings.Contains(page, "expired or was already used") {
		t.Fatalf("reused link page: %d %s", res.StatusCode, page)
	}
	res = e.postForm(t, c, "/auth/verify", url.Values{"t": {strings.TrimPrefix(link, "/auth/verify?t=")}})
	if page := body(t, res); res.StatusCode != http.StatusBadRequest || !strings.Contains(page, "expired or was already used") {
		t.Fatalf("reused link redeem: %d", res.StatusCode)
	}
	// A cross-site redeem is refused before the token is touched.
	c3 := e.client(t)
	e.postForm(t, c3, "/auth/magic", url.Values{"email": {"other@example.com"}}).Body.Close()
	tok := strings.TrimPrefix(linkRE.FindString(e.mailer.last().Text), "/auth/verify?t=")
	res = e.postForm(t, c3, "/auth/verify", url.Values{"t": {tok}}, "Sec-Fetch-Site", "cross-site")
	body(t, res)
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("cross-site redeem: %d", res.StatusCode)
	}
	res = e.postForm(t, c3, "/auth/verify", url.Values{"t": {tok}})
	body(t, res)
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("token should survive a refused cross-site attempt: %d", res.StatusCode)
	}

	// Signed-in parent sees the family page with their (normalized) email.
	res, _ = c.Get(e.srv.URL + "/family")
	if page := body(t, res); res.StatusCode != http.StatusOK || !strings.Contains(page, "parent@example.com") {
		t.Fatalf("family: %d %s", res.StatusCode, snippet(page))
	}

	// Cookie flags.
	u, _ := url.Parse(e.srv.URL)
	var found bool
	for _, ck := range c.Jar.Cookies(u) {
		if ck.Name == sessionCookie {
			found = true
		}
	}
	if !found {
		t.Fatal("session cookie not set")
	}

	// Sign out clears it.
	res = e.postForm(t, c, "/auth/logout", url.Values{})
	body(t, res)
	res, _ = c.Get(e.srv.URL + "/family")
	body(t, res)
	if res.StatusCode != http.StatusSeeOther || res.Header.Get("Location") != "/login" {
		t.Fatalf("after logout: %d -> %q", res.StatusCode, res.Header.Get("Location"))
	}
}

func TestMagicDoesNotRevealOrFlood(t *testing.T) {
	e := newTestEnv(t)
	c := e.client(t)
	// Production responses must be byte-identical whether or not mail was
	// sent; dev mode deliberately echoes the link, so turn it off here.
	e.s.cfg.dev = false
	var pages []string
	for i := 0; i < 5; i++ {
		res := e.postForm(t, c, "/auth/magic", url.Values{"email": {"same@example.com"}})
		if res.StatusCode != http.StatusOK {
			t.Fatalf("attempt %d: %d", i, res.StatusCode)
		}
		pages = append(pages, body(t, res))
	}
	// Rate-limited responses are identical to successful ones.
	if pages[0] != pages[4] {
		t.Fatal("rate-limited response differs from normal response")
	}
	if n := len(e.mailer.sent); n != 3 {
		t.Fatalf("sent %d mails, want 3 (limit)", n)
	}
	if strings.Contains(pages[0], "/auth/verify?t=") {
		t.Fatal("production response must never echo the sign-in link")
	}
	res := e.postForm(t, c, "/auth/magic", url.Values{"email": {"not-an-email"}})
	if page := body(t, res); !strings.Contains(page, "valid email") {
		t.Fatalf("bad email should re-render login: %s", snippet(page))
	}
}

func TestSessionCookieAttributes(t *testing.T) {
	e := newTestEnv(t)
	c := e.client(t)
	e.postForm(t, c, "/auth/magic", url.Values{"email": {"a@example.com"}}).Body.Close()
	link := linkRE.FindString(e.mailer.last().Text)
	res := e.postForm(t, c, "/auth/verify", url.Values{"t": {strings.TrimPrefix(link, "/auth/verify?t=")}})
	body(t, res)
	raw := res.Header.Get("Set-Cookie")
	for _, want := range []string{"HttpOnly", "SameSite=Lax", "Path=/", "Max-Age=7776000"} {
		if !strings.Contains(raw, want) {
			t.Errorf("Set-Cookie %q lacks %s", raw, want)
		}
	}
	if strings.Contains(raw, "Secure") {
		t.Error("dev mode should not set Secure over http")
	}
	e.s.cfg.dev = false
	rec := httptest.NewRecorder()
	e.s.setSessionCookie(rec, "x")
	if !strings.Contains(rec.Header().Get("Set-Cookie"), "Secure") {
		t.Error("production cookie must be Secure")
	}
}

func TestChildrenAreFencedAcrossAccounts(t *testing.T) {
	e := newTestEnv(t)
	a := e.signIn(t, "a@example.com")
	b := e.signIn(t, "b@example.com")

	res := e.postForm(t, a, "/family/children", url.Values{"nickname": {" Nova "}, "grade": {"2"}})
	body(t, res)
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("create: %d", res.StatusCode)
	}
	res, _ = a.Get(e.srv.URL + "/family")
	page := body(t, res)
	if !strings.Contains(page, "Nova") || !strings.Contains(page, "training now") {
		t.Fatalf("first child should be active: %s", page)
	}
	idRE := regexp.MustCompile(`/family/children/(\d+)`)
	m := idRE.FindStringSubmatch(page)
	if m == nil {
		t.Fatal("no child id in page")
	}
	childPath := "/family/children/" + m[1]

	// The practice page shows the active child via data attributes, escaped.
	res, _ = a.Get(e.srv.URL + "/")
	page = body(t, res)
	if !strings.Contains(page, `data-child-nickname="Nova"`) || !strings.Contains(page, "Training: Nova") {
		t.Fatalf("index should carry the active child: %s", snippet(page))
	}

	// B cannot touch A's child by id: every action is a 404.
	for _, action := range []string{"select", "rename", "delete"} {
		res = e.postForm(t, b, childPath, url.Values{"action": {action}, "nickname": {"Hacked"}, "grade": {"1"}})
		body(t, res)
		if res.StatusCode != http.StatusNotFound {
			t.Errorf("cross-account %s: %d, want 404", action, res.StatusCode)
		}
	}
	res, _ = b.Get(e.srv.URL + "/family")
	if page := body(t, res); strings.Contains(page, "Train as Nova") || strings.Contains(page, childPath) {
		t.Fatal("B can see A's child")
	}

	// A can rename, and validation errors come back as 400 with a message.
	res = e.postForm(t, a, childPath, url.Values{"action": {"rename"}, "nickname": {"<b>x</b>"}, "grade": {"3"}})
	if page := body(t, res); res.StatusCode != http.StatusBadRequest || !strings.Contains(page, "letters, numbers") {
		t.Fatalf("bad nickname: %d %s", res.StatusCode, snippet(page))
	}
	res = e.postForm(t, a, childPath, url.Values{"action": {"rename"}, "nickname": {"Nova B"}, "grade": {"3"}})
	body(t, res)
	res, _ = a.Get(e.srv.URL + "/")
	if page := body(t, res); !strings.Contains(page, `data-child-nickname="Nova B"`) || !strings.Contains(page, `data-child-grade="3"`) {
		t.Fatalf("rename not reflected: %s", snippet(page))
	}

	// Nicknames are HTML-escaped wherever they render.
	res = e.postForm(t, a, "/family/children", url.Values{"nickname": {"O'Neil"}, "grade": {"1"}})
	body(t, res)
	res, _ = a.Get(e.srv.URL + "/family")
	if page := body(t, res); !strings.Contains(page, "O&#39;Neil") {
		t.Fatalf("apostrophe not escaped: %s", page)
	}

	// Delete hides it and clears the selection.
	res = e.postForm(t, a, childPath, url.Values{"action": {"delete"}})
	body(t, res)
	res, _ = a.Get(e.srv.URL + "/")
	if page := body(t, res); strings.Contains(page, "Nova B") {
		t.Fatal("deleted child still active on index")
	}
}

func TestCrossOriginPostsAreRejected(t *testing.T) {
	e := newTestEnv(t)
	c := e.client(t)
	res := e.postForm(t, c, "/auth/magic", url.Values{"email": {"a@example.com"}}, "Sec-Fetch-Site", "cross-site")
	body(t, res)
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("cross-site POST: %d, want 403", res.StatusCode)
	}
	if len(e.mailer.sent) != 0 {
		t.Fatal("cross-site POST sent mail")
	}
}

func TestFreeTierIsUntouchedWithoutSession(t *testing.T) {
	e := newTestEnv(t)
	c := e.client(t)
	res, _ := c.Get(e.srv.URL + "/")
	page := body(t, res)
	if res.StatusCode != http.StatusOK || strings.Contains(page, "data-child-id") || !strings.Contains(page, `href="/login"`) {
		t.Fatalf("anonymous index: %d", res.StatusCode)
	}
	req, _ := http.NewRequest(http.MethodPost, e.srv.URL+"/api/sheet", strings.NewReader(`{"ops":["addsub"],"grade":1,"count":10}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	res, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if page := body(t, res); res.StatusCode != http.StatusOK || !strings.Contains(page, `"questions"`) {
		t.Fatalf("api/sheet without session: %d %s", res.StatusCode, page)
	}
}

func TestRateLimiterIsBoundedAndChecksIPFirst(t *testing.T) {
	l := newRateLimiter()
	l.maxKeys = 3
	base := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	for i, key := range []string{"a", "b", "c"} {
		if !l.allow(key, 1, base) {
			t.Fatalf("key %d denied", i)
		}
	}
	if l.allow("d", 1, base) {
		t.Fatal("table full: a new key must be denied, not grown")
	}
	if l.allow("a", 1, base) {
		t.Fatal("limit 1 exceeded")
	}
	// After the window the sweep frees space.
	if !l.allow("d", 1, base.Add(16*time.Minute)) {
		t.Fatal("expired keys should be swept when full")
	}

	e := newTestEnv(t)
	c := e.client(t)
	for i := 0; i < 12; i++ {
		e.postForm(t, c, "/auth/magic", url.Values{"email": {fmt.Sprintf("u%d@example.com", i)}}).Body.Close()
	}
	if n := len(e.mailer.sent); n != 10 {
		t.Fatalf("one IP sent %d mails, want 10", n)
	}
	if len(e.s.limiter.hits) != 11 { // ip + 10 emails; the 2 denied never added keys
		t.Fatalf("denied requests added limiter keys: %d", len(e.s.limiter.hits))
	}
}

func (e *testEnv) postJSON(t *testing.T, c *http.Client, path, payload string) (*http.Response, string) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, e.srv.URL+path, strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	res, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return res, body(t, res)
}

func TestRoundEndpointsRequireASelectedChild(t *testing.T) {
	e := newTestEnv(t)
	anon := e.client(t)
	res, page := e.postJSON(t, anon, "/api/round/start", `{"round_id":"abcdefgh","kind":"spelling","grade":1,"count":5}`)
	if res.StatusCode != http.StatusUnauthorized || !strings.Contains(page, "sign in") {
		t.Fatalf("anonymous start: %d %s", res.StatusCode, page)
	}
	parent := e.signIn(t, "p@example.com")
	res, page = e.postJSON(t, parent, "/api/round/start", `{"round_id":"abcdefgh","kind":"spelling","grade":1,"count":5}`)
	if res.StatusCode != http.StatusUnauthorized || !strings.Contains(page, "choose who") {
		t.Fatalf("no child selected: %d %s", res.StatusCode, page)
	}
}

func TestSignedInSpellingRoundIsGradedByTheServer(t *testing.T) {
	e := newTestEnv(t)
	c := e.signIn(t, "p@example.com")
	e.postForm(t, c, "/family/children", url.Values{"nickname": {"Nova"}, "grade": {"2"}}).Body.Close()

	res, page := e.postJSON(t, c, "/api/round/start", `{"round_id":"round-abc-123","kind":"spelling","focus":"g2-endings","grade":2,"count":5}`)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("start: %d %s", res.StatusCode, page)
	}
	var start struct {
		Words []struct{ Word string } `json:"words"`
	}
	if err := json.Unmarshal([]byte(page), &start); err != nil || len(start.Words) != 5 {
		t.Fatalf("start payload: %v %s", err, page)
	}

	// The client claims every word but actually only knows the first two;
	// the other three are six wrong whole-word guesses.
	var guesses []string
	for i, w := range start.Words {
		if i < 2 {
			guesses = append(guesses, fmt.Sprintf(`[{"kind":"word","value":%q}]`, w.Word))
		} else {
			guesses = append(guesses, `[{"kind":"word","value":"zz"},{"kind":"word","value":"zz"},{"kind":"word","value":"zz"},{"kind":"word","value":"zz"},{"kind":"word","value":"zz"},{"kind":"word","value":"zz"}]`)
		}
	}
	finishBody := `{"round_id":"round-abc-123","guesses":[` + strings.Join(guesses, ",") + `]}`
	res, page = e.postJSON(t, c, "/api/round/finish", finishBody)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("finish: %d %s", res.StatusCode, page)
	}
	var fin struct {
		Score  int `json:"score"`
		Total  int `json:"total"`
		Reward struct {
			Fills        int  `json:"fills"`
			MosaicCells  int  `json:"mosaic_cells"`
			BattleCredit bool `json:"battle_credit"`
		} `json:"reward"`
	}
	if err := json.Unmarshal([]byte(page), &fin); err != nil {
		t.Fatal(err)
	}
	if fin.Score != 2 || fin.Total != 5 || fin.Reward.Fills != 2 || fin.Reward.MosaicCells != 2 || !fin.Reward.BattleCredit {
		t.Fatalf("server verdict: %s", page)
	}
	// Retrying with an all-wins body returns the stored result.
	res, page2 := e.postJSON(t, c, "/api/round/finish", finishBody)
	if res.StatusCode != http.StatusOK || page2 != page {
		t.Fatalf("retry differs: %s", page2)
	}
	// Unknown fields and oversized bodies are rejected.
	res, _ = e.postJSON(t, c, "/api/round/finish", `{"round_id":"round-abc-123","correct":true}`)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("unknown field accepted: %d", res.StatusCode)
	}
	res, _ = e.postJSON(t, c, "/api/round/finish", `{"round_id":"round-abc-123","guesses":[`+strings.Repeat(`[{"kind":"word","value":"zz"}],`, 2000)+`[]]}`)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("oversized body accepted: %d", res.StatusCode)
	}

	// Another parent cannot finish this round even with the id.
	other := e.signIn(t, "q@example.com")
	e.postForm(t, other, "/family/children", url.Values{"nickname": {"Max"}, "grade": {"2"}}).Body.Close()
	res, _ = e.postJSON(t, other, "/api/round/finish", finishBody)
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("cross-account finish: %d", res.StatusCode)
	}

	// Math rounds wrap the sheet store.
	res, page = e.postJSON(t, c, "/api/round/start", `{"round_id":"round-math-1","kind":"math","ops":["addsub"],"grade":2,"count":10}`)
	if res.StatusCode != http.StatusOK || !strings.Contains(page, `"questions"`) {
		t.Fatalf("math start: %d %s", res.StatusCode, page)
	}
	res, page = e.postJSON(t, c, "/api/round/finish", `{"round_id":"round-math-1","answers":["","","","","","","","","",""]}`)
	if res.StatusCode != http.StatusOK || !strings.Contains(page, `"score":0`) || !strings.Contains(page, `"results"`) {
		t.Fatalf("math finish: %d %s", res.StatusCode, page)
	}
}

func TestKataIndexEndpoint(t *testing.T) {
	e := newTestEnv(t)
	anon := e.client(t)
	res, err := anon.Get(e.srv.URL + "/api/kata/index")
	if err != nil {
		t.Fatal(err)
	}
	if body(t, res); res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous index: %d", res.StatusCode)
	}
	c := e.signIn(t, "p@example.com")
	e.postForm(t, c, "/family/children", url.Values{"nickname": {"Nova"}, "grade": {"2"}}).Body.Close()
	res, _ = c.Get(e.srv.URL + "/api/kata/index")
	page := body(t, res)
	if res.StatusCode != http.StatusOK || res.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("index: %d %s", res.StatusCode, res.Header.Get("Cache-Control"))
	}
	var idx struct {
		Entries    []struct{ ID, State, Hint, Name string } `json:"entries"`
		ChildGrade int                                      `json:"child_grade"`
	}
	if err := json.Unmarshal([]byte(page), &idx); err != nil || len(idx.Entries) != 51 || idx.ChildGrade != 2 {
		t.Fatalf("index payload: %v, %d entries", err, len(idx.Entries))
	}
	// Names of unknown creatures are still in the payload (the client masks
	// them); what must never be there is a curriculum word in a hint.
	for _, en := range idx.Entries {
		if en.State != "unknown" {
			t.Fatalf("fresh child should have no state: %+v", en)
		}
	}

	// Playing a round through the API touches the creature and the reveal
	// payload carries it.
	res, page = e.postJSON(t, c, "/api/round/start", `{"round_id":"kata-api-1","kind":"spelling","focus":"g2-endings","grade":2,"count":5}`)
	var start struct {
		Words []struct{ Word string } `json:"words"`
	}
	json.Unmarshal([]byte(page), &start)
	var guesses []string
	for _, w := range start.Words {
		guesses = append(guesses, fmt.Sprintf(`[{"kind":"word","value":%q}]`, w.Word))
	}
	res, page = e.postJSON(t, c, "/api/round/finish", `{"round_id":"kata-api-1","guesses":[`+strings.Join(guesses, ",")+`]}`)
	if res.StatusCode != http.StatusOK || !strings.Contains(page, `"creatures"`) || !strings.Contains(page, `"newly_seen":true`) {
		t.Fatalf("finish reveal: %d %s", res.StatusCode, page)
	}
	res, _ = c.Get(e.srv.URL + "/api/kata/index")
	page = body(t, res)
	if !strings.Contains(page, `"id":"g2-endings","name":"Tailfin"`) || !strings.Contains(page, `"state":"seen"`) {
		t.Fatalf("index after round: %s", snippet(page))
	}
}

func TestPaintEndpoints(t *testing.T) {
	e := newTestEnv(t)
	anon := e.client(t)
	res, _ := anon.Get(e.srv.URL + "/api/mosaic/week")
	if body(t, res); res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous mosaic: %d", res.StatusCode)
	}
	c := e.signIn(t, "p@example.com")
	e.postForm(t, c, "/family/children", url.Values{"nickname": {"Nova"}, "grade": {"2"}}).Body.Close()

	res, _ = c.Get(e.srv.URL + "/api/mosaic/week")
	page := body(t, res)
	if res.StatusCode != http.StatusOK || !strings.Contains(page, `"revealed":0`) || !strings.Contains(page, `"total":400`) {
		t.Fatalf("mosaic week: %d %s", res.StatusCode, snippet(page))
	}

	res, page = e.postJSON(t, c, "/api/paint/page", `{"page_id":"paint-page-1","ops":["addsub"],"grade":2}`)
	if res.StatusCode != http.StatusOK || !strings.Contains(page, `"total":14`) || strings.Contains(page, `"answer"`) {
		t.Fatalf("page start: %d %s", res.StatusCode, page)
	}
	res, page = e.postJSON(t, c, "/api/paint/fill", `{"page_id":"paint-page-1","idx":0,"answer":"nope"}`)
	if res.StatusCode != http.StatusOK || !strings.Contains(page, `"right":false`) || !strings.Contains(page, `"attempts":1`) {
		t.Fatalf("wrong fill: %d %s", res.StatusCode, snippet(page))
	}
	res, page = e.postJSON(t, c, "/api/paint/fill", `{"page_id":"paint-page-1","idx":99,"answer":"1"}`)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad region: %d", res.StatusCode)
	}
	res, page = e.postJSON(t, c, "/api/paint/fill", `{"page_id":"paint-page-1","idx":0,"answer":"1","extra":true}`)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("unknown field: %d", res.StatusCode)
	}
	other := e.signIn(t, "q@example.com")
	e.postForm(t, other, "/family/children", url.Values{"nickname": {"Max"}, "grade": {"2"}}).Body.Close()
	res, _ = e.postJSON(t, other, "/api/paint/fill", `{"page_id":"paint-page-1","idx":0,"answer":"1"}`)
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("cross-account fill: %d", res.StatusCode)
	}

	// A spelling round through the API reveals mosaic tiles.
	res, page = e.postJSON(t, c, "/api/round/start", `{"round_id":"mosaic-api-1","kind":"spelling","focus":"g2-endings","grade":2,"count":5}`)
	var start struct {
		Words []struct{ Word string } `json:"words"`
	}
	json.Unmarshal([]byte(page), &start)
	var guesses []string
	for _, w := range start.Words {
		guesses = append(guesses, fmt.Sprintf(`[{"kind":"word","value":%q}]`, w.Word))
	}
	res, page = e.postJSON(t, c, "/api/round/finish", `{"round_id":"mosaic-api-1","guesses":[`+strings.Join(guesses, ",")+`]}`)
	if res.StatusCode != http.StatusOK || !strings.Contains(page, `"mosaic":{`) || !strings.Contains(page, `"added":5`) {
		t.Fatalf("finish mosaic: %d %s", res.StatusCode, page)
	}
	res, _ = c.Get(e.srv.URL + "/api/mosaic/week")
	if page = body(t, res); !strings.Contains(page, `"revealed":5`) {
		t.Fatalf("mosaic after round: %s", snippet(page))
	}
}

func TestBattleEndpoints(t *testing.T) {
	e := newTestEnv(t)
	c := e.signIn(t, "p@example.com")
	e.postForm(t, c, "/family/children", url.Values{"nickname": {"Nova"}, "grade": {"2"}}).Body.Close()

	res, _ := c.Get(e.srv.URL + "/api/battle/credits")
	page := body(t, res)
	if res.StatusCode != http.StatusOK || !strings.Contains(page, `"credits":0`) || !strings.Contains(page, `"enabled":true`) {
		t.Fatalf("credits: %d %s", res.StatusCode, page)
	}
	res, page = e.postJSON(t, c, "/api/battle/start", `{"battle_id":"battle-api-1","creature_id":"g2-endings"}`)
	if res.StatusCode != http.StatusBadRequest { // creature not found yet
		t.Fatalf("start before finding: %d %s", res.StatusCode, page)
	}

	// Earn a credit with a full round.
	res, page = e.postJSON(t, c, "/api/round/start", `{"round_id":"battle-round-1","kind":"spelling","focus":"g2-endings","grade":2,"count":5}`)
	var start struct {
		Words []struct{ Word string } `json:"words"`
	}
	json.Unmarshal([]byte(page), &start)
	var guesses []string
	for _, w := range start.Words {
		guesses = append(guesses, fmt.Sprintf(`[{"kind":"word","value":%q}]`, w.Word))
	}
	res, page = e.postJSON(t, c, "/api/round/finish", `{"round_id":"battle-round-1","guesses":[`+strings.Join(guesses, ",")+`]}`)
	if res.StatusCode != http.StatusOK || !strings.Contains(page, `"battle_credit":true`) {
		t.Fatalf("finish: %d %s", res.StatusCode, page)
	}
	res, _ = c.Get(e.srv.URL + "/api/battle/credits")
	if page = body(t, res); !strings.Contains(page, `"credits":1`) {
		t.Fatalf("credits after round: %s", page)
	}

	res, page = e.postJSON(t, c, "/api/battle/start", `{"battle_id":"battle-api-1","creature_id":"g2-endings"}`)
	if res.StatusCode != http.StatusOK || strings.Contains(page, `"item_answer"`) || !strings.Contains(page, `"hp":5`) {
		t.Fatalf("start: %d %s", res.StatusCode, page)
	}
	res, page = e.postJSON(t, c, "/api/battle/turn", `{"battle_id":"battle-api-1","turn":0,"answer":"zzzz"}`)
	if res.StatusCode != http.StatusOK || !strings.Contains(page, `"turn":1`) || strings.Contains(page, `"item_answer"`) {
		t.Fatalf("turn: %d %s", res.StatusCode, page)
	}
	res, page = e.postJSON(t, c, "/api/battle/start", `{"battle_id":"battle-api-2","creature_id":"g2-endings"}`)
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("second battle without credit: %d %s", res.StatusCode, page)
	}
	other := e.signIn(t, "q@example.com")
	e.postForm(t, other, "/family/children", url.Values{"nickname": {"Max"}, "grade": {"2"}}).Body.Close()
	res, _ = e.postJSON(t, other, "/api/battle/turn", `{"battle_id":"battle-api-1","turn":1,"answer":"x"}`)
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("cross-account turn: %d", res.StatusCode)
	}

	// The pilot switch hides battles without touching anything else.
	e.s.cfg.battlesDisabled = true
	res, page = e.postJSON(t, c, "/api/battle/turn", `{"battle_id":"battle-api-1","turn":1,"answer":"x"}`)
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("disabled battles: %d", res.StatusCode)
	}
	res, _ = c.Get(e.srv.URL + "/api/battle/credits")
	if page = body(t, res); !strings.Contains(page, `"enabled":false`) {
		t.Fatalf("credits should report disabled: %s", page)
	}
	res, _ = c.Get(e.srv.URL + "/")
	if page = body(t, res); res.StatusCode != http.StatusOK {
		t.Fatal("index broke with battles disabled")
	}
}

func TestTimesTablesSheetsAndRounds(t *testing.T) {
	e := newTestEnv(t)
	anon := e.client(t)
	res, page := e.postJSON(t, anon, "/api/sheet", `{"ops":["tables"],"grade":3,"count":12,"table":8,"ordered":true}`)
	if res.StatusCode != http.StatusOK || !strings.Contains(page, `"prompt":"1 × 8"`) || !strings.Contains(page, `"prompt":"12 × 8"`) {
		t.Fatalf("tables sheet: %d %s", res.StatusCode, snippet(page))
	}
	res, _ = e.postJSON(t, anon, "/api/sheet", `{"ops":["tables","addsub"],"grade":3,"count":12,"table":8}`)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("tables mixed with another op accepted: %d", res.StatusCode)
	}
	res, _ = e.postJSON(t, anon, "/api/sheet", `{"ops":["tables"],"grade":3,"count":12,"table":1}`)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("table 1 accepted: %d", res.StatusCode)
	}
	res, _ = e.postJSON(t, anon, "/api/sheet", `{"ops":["addsub"],"grade":3,"count":10,"table":7}`)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("table field outside table mode accepted: %d", res.StatusCode)
	}

	c := e.signIn(t, "p@example.com")
	e.postForm(t, c, "/family/children", url.Values{"nickname": {"Nova"}, "grade": {"3"}}).Body.Close()
	res, page = e.postJSON(t, c, "/api/round/start", `{"round_id":"tables-round-1","kind":"math","ops":["tables"],"grade":3,"count":12,"table":6,"ordered":false}`)
	if res.StatusCode != http.StatusOK || !strings.Contains(page, `"questions"`) {
		t.Fatalf("tables round: %d %s", res.StatusCode, snippet(page))
	}
	answers := make([]string, 12)
	for i := range answers {
		answers[i] = "1"
	}
	res, page = e.postJSON(t, c, "/api/round/finish", `{"round_id":"tables-round-1","answers":["1","1","1","1","1","1","1","1","1","1","1","1"]}`)
	if res.StatusCode != http.StatusOK || !strings.Contains(page, `"total":12`) {
		t.Fatalf("tables finish: %d %s", res.StatusCode, snippet(page))
	}
	// The round trains the multiplication skill and its creature.
	res, _ = c.Get(e.srv.URL + "/api/kata/index")
	if page = body(t, res); !strings.Contains(page, `"id":"math-mul-g3","name":"Gridcat"`) || !strings.Contains(page, `"state":"seen"`) {
		t.Fatalf("tables round should touch the multiplication creature: %s", snippet(page))
	}
}

func TestConfigValidation(t *testing.T) {
	if err := (config{dev: true}).validate(); err != nil {
		t.Fatalf("dev config should validate: %v", err)
	}
	bad := []config{
		{},
		{baseURL: "http://skilldojo.io", resendKey: "re_x", mailFrom: "a@b.c"},
		{baseURL: "https://skilldojo.io", resendKey: "changeme", mailFrom: "a@b.c"},
		{baseURL: "https://skilldojo.io", resendKey: "re_x", mailFrom: ""},
	}
	for i, c := range bad {
		if err := c.validate(); err == nil {
			t.Errorf("config %d accepted: %+v", i, c)
		}
	}
	if err := (config{baseURL: "https://skilldojo.io", resendKey: "re_x", mailFrom: "SkillDojo <hi@skilldojo.io>"}).validate(); err != nil {
		t.Fatal(err)
	}
	_ = time.Second
}

func TestPracticeTestEndpoints(t *testing.T) {
	e := newTestEnv(t)
	anon := e.client(t)
	res, _ := e.postJSON(t, anon, "/api/ost/start", `{"grade":3}`)
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous start: %d", res.StatusCode)
	}
	res, _ = anon.Get(e.srv.URL + "/test")
	if res.StatusCode != http.StatusSeeOther {
		t.Fatalf("anonymous test page: %d", res.StatusCode)
	}

	c := e.signIn(t, "p@example.com")
	e.postForm(t, c, "/family/children", url.Values{"nickname": {"Nova"}, "grade": {"2"}}).Body.Close()

	// A grade-2 child gets the grade-3 test by default; the page and the
	// report both render before any attempt exists.
	res, _ = c.Get(e.srv.URL + "/test")
	if page := body(t, res); res.StatusCode != http.StatusOK || !strings.Contains(page, `data-test-grade="3"`) {
		t.Fatalf("test page: %d %s", res.StatusCode, snippet(page))
	}
	res, _ = c.Get(e.srv.URL + "/family/tests")
	if page := body(t, res); res.StatusCode != http.StatusOK || !strings.Contains(page, "No practice tests yet") {
		t.Fatalf("tests page: %d %s", res.StatusCode, snippet(page))
	}

	res, page := e.postJSON(t, c, "/api/ost/start", `{"grade":9}`)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("grade 9: %d %s", res.StatusCode, page)
	}
	res, page = e.postJSON(t, c, "/api/ost/start", `{"grade":0}`)
	var a struct {
		ID    string `json:"id"`
		Grade int    `json:"grade"`
		Items []struct {
			ID      string   `json:"id"`
			Type    string   `json:"type"`
			Choices []string `json:"choices"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(page), &a); err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("start: %d %s", res.StatusCode, page)
	}
	if a.Grade != 3 || len(a.Items) != 40 || strings.Contains(page, `"answer"`) || strings.Contains(page, `"numeric"`) || strings.Contains(page, `"explanation"`) {
		t.Fatalf("start payload leaks or is short: grade %d, %d items, %s", a.Grade, len(a.Items), snippet(page))
	}

	// Starting again resumes the same attempt.
	res, page = e.postJSON(t, c, "/api/ost/start", `{"grade":3}`)
	if !strings.Contains(page, `"id":"`+a.ID+`"`) {
		t.Fatalf("start should resume: %s", snippet(page))
	}

	// Answer every item with something valid; a wrong index is a 400.
	first := a.Items[0]
	res, page = e.postJSON(t, c, "/api/ost/answer", fmt.Sprintf(`{"attempt_id":%q,"item_id":%q,"choices":[9],"text":""}`, a.ID, first.ID))
	if first.Type != "number" && res.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad choice: %d %s", res.StatusCode, page)
	}
	res, page = e.postJSON(t, c, "/api/ost/answer", fmt.Sprintf(`{"attempt_id":%q,"item_id":"nope","choices":[],"text":"1"}`, a.ID))
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad item: %d %s", res.StatusCode, page)
	}
	for _, it := range a.Items {
		payload := fmt.Sprintf(`{"attempt_id":%q,"item_id":%q,"choices":[0],"text":""}`, a.ID, it.ID)
		if it.Type == "number" {
			payload = fmt.Sprintf(`{"attempt_id":%q,"item_id":%q,"choices":[],"text":"7"}`, a.ID, it.ID)
		}
		if res, page = e.postJSON(t, c, "/api/ost/answer", payload); res.StatusCode != http.StatusOK {
			t.Fatalf("answer %s: %d %s", it.ID, res.StatusCode, page)
		}
	}

	// The saved answers come back on resume; the parent sees progress.
	res, page = e.postJSON(t, c, "/api/ost/start", `{"grade":3}`)
	if !strings.Contains(page, `"answers":{"q01"`) && !strings.Contains(page, `"q01":{`) {
		t.Fatalf("resume should carry answers: %s", snippet(page))
	}
	res, _ = c.Get(e.srv.URL + "/family/tests")
	if page := body(t, res); !strings.Contains(page, "40 of 40 answered") || !strings.Contains(page, "Resume test") {
		t.Fatalf("in-progress row: %s", snippet(page))
	}

	res, page = e.postJSON(t, c, "/api/ost/submit", fmt.Sprintf(`{"attempt_id":%q}`, a.ID))
	if res.StatusCode != http.StatusOK || !strings.Contains(page, `"report":{`) || !strings.Contains(page, `"level":"`) || !strings.Contains(page, `"categories":[`) {
		t.Fatalf("submit: %d %s", res.StatusCode, snippet(page))
	}
	// After grading, answers are locked and a second submit returns the same report.
	res, page = e.postJSON(t, c, "/api/ost/answer", fmt.Sprintf(`{"attempt_id":%q,"item_id":%q,"choices":[],"text":"8"}`, a.ID, a.Items[0].ID))
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("answer after submit: %d %s", res.StatusCode, page)
	}
	res, page2 := e.postJSON(t, c, "/api/ost/submit", fmt.Sprintf(`{"attempt_id":%q}`, a.ID))
	if res.StatusCode != http.StatusOK || page2 != page[:0]+page2 && !strings.Contains(page2, `"finished_at"`) {
		t.Fatalf("second submit: %d", res.StatusCode)
	}

	// Parent report: history row, focus areas, and the item review page.
	res, _ = c.Get(e.srv.URL + "/family/tests")
	page = body(t, res)
	if !strings.Contains(page, "Review</a>") || !strings.Contains(page, "Areas needing improvement") || !strings.Contains(page, "At a glance") {
		t.Fatalf("report page: %s", snippet(page))
	}
	res, _ = c.Get(e.srv.URL + "/family/tests/1/" + a.ID)
	if page := body(t, res); res.StatusCode != http.StatusOK || !strings.Contains(page, "Every question") || !strings.Contains(page, "DOK") {
		t.Fatalf("attempt page: %d %s", res.StatusCode, snippet(page))
	}

	// A new attempt after submitting is a different test; the old one is fenced from other accounts.
	res, page = e.postJSON(t, c, "/api/ost/start", `{"grade":3}`)
	if res.StatusCode != http.StatusOK || strings.Contains(page, `"id":"`+a.ID+`"`) {
		t.Fatalf("second attempt should be new: %d %s", res.StatusCode, snippet(page))
	}
	other := e.signIn(t, "q@example.com")
	e.postForm(t, other, "/family/children", url.Values{"nickname": {"Zed"}, "grade": {"4"}}).Body.Close()
	res, page = e.postJSON(t, other, "/api/ost/submit", fmt.Sprintf(`{"attempt_id":%q}`, a.ID))
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("cross-account submit: %d %s", res.StatusCode, page)
	}
	res, _ = other.Get(e.srv.URL + "/family/tests/1/" + a.ID)
	if body(t, res); res.StatusCode != http.StatusNotFound {
		t.Fatalf("cross-account review: %d", res.StatusCode)
	}
}
