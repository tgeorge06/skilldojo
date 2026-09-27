package main

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/tgeorge06/skilldojo/internal/account"
	"github.com/tgeorge06/skilldojo/internal/ost"
	"github.com/tgeorge06/skilldojo/internal/progress"
)

// Practice tests (Ohio's State Tests style) live behind the parent portal:
// a parent starts one for the active child from /family/tests, the child
// takes it at /test, and results and focus areas come back to /family/tests.

func ostChild(c progress.Child) ost.Child {
	return ost.Child{AccountID: c.AccountID, ChildID: c.ChildID, Grade: c.Grade}
}

type testPageData struct {
	Child account.Child
	Grade int
}

// handleTestPage renders the test-taking page for the active child.
func (s *server) handleTestPage(w http.ResponseWriter, r *http.Request) {
	sess, ok := s.requireSession(w, r)
	if !ok {
		return
	}
	if sess.ActiveChildID == 0 {
		http.Redirect(w, r, "/family", http.StatusSeeOther)
		return
	}
	child, err := s.accounts.Child(r.Context(), sess.AccountID, sess.ActiveChildID)
	if err != nil {
		http.Redirect(w, r, "/family", http.StatusSeeOther)
		return
	}
	grade := ost.NearestGrade(child.Grade)
	if g, err := strconv.Atoi(r.URL.Query().Get("grade")); err == nil && ost.ValidGrade(g) {
		grade = g
	}
	w.Header().Set("Cache-Control", "no-store")
	s.render(w, "test.html", testPageData{Child: child, Grade: grade})
}

type ostStartRequest struct {
	Grade int `json:"grade"`
}

func (s *server) handleOSTStart(w http.ResponseWriter, r *http.Request) {
	child, ok := s.activeChild(w, r)
	if !ok {
		return
	}
	var req ostStartRequest
	if err := decodeJSONLimit(w, r, &req, 1<<10); err != nil {
		return
	}
	if req.Grade == 0 {
		req.Grade = ost.NearestGrade(child.Grade)
	}
	a, err := s.tests.Start(r.Context(), ostChild(child), req.Grade, s.now())
	if err != nil {
		writeOSTError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, a)
}

type ostAnswerRequest struct {
	AttemptID string `json:"attempt_id"`
	ItemID    string `json:"item_id"`
	Choices   []int  `json:"choices"`
	Text      string `json:"text"`
}

func (s *server) handleOSTAnswer(w http.ResponseWriter, r *http.Request) {
	child, ok := s.activeChild(w, r)
	if !ok {
		return
	}
	var req ostAnswerRequest
	if err := decodeJSONLimit(w, r, &req, 2<<10); err != nil {
		return
	}
	err := s.tests.SaveAnswer(r.Context(), ostChild(child), req.AttemptID, req.ItemID, ost.Answer{Choices: req.Choices, Text: req.Text}, s.now())
	if err != nil {
		writeOSTError(w, err)
		return
	}
	writeJSON(w, map[string]bool{"saved": true})
}

type ostSubmitRequest struct {
	AttemptID string `json:"attempt_id"`
}

func (s *server) handleOSTSubmit(w http.ResponseWriter, r *http.Request) {
	child, ok := s.activeChild(w, r)
	if !ok {
		return
	}
	var req ostSubmitRequest
	if err := decodeJSONLimit(w, r, &req, 1<<10); err != nil {
		return
	}
	a, err := s.tests.Submit(r.Context(), ostChild(child), req.AttemptID, s.now())
	if err != nil {
		writeOSTError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, a)
}

func writeOSTError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ost.ErrNotFound):
		writeError(w, http.StatusNotFound, errors.New("practice test not found"))
	case errors.Is(err, ost.ErrFinished):
		writeError(w, http.StatusConflict, errors.New("this test has already been turned in"))
	case errors.Is(err, ost.ErrBadRequest):
		writeError(w, http.StatusBadRequest, err)
	default:
		log.Printf("ost: %v", err)
		writeError(w, http.StatusInternalServerError, errors.New("something went wrong"))
	}
}

// Parent report.

type childReport struct {
	Child         account.Child
	TestGrade     int // default grade for a new test
	AnalysisGrade int // grade whose attempts the analysis covers
	InProgress    *ost.Summary
	History       []attemptRow
	Analysis      ost.Analysis
	Categories    []string
}

type attemptRow struct {
	ost.Summary
	When string
}

type testsPageData struct {
	Email         string
	Children      []account.Child
	Selected      *childReport
	ActiveChildID int64
	TestGrades    []int
	Levels        []ost.Level
}

func (s *server) handleFamilyTests(w http.ResponseWriter, r *http.Request) {
	sess, ok := s.requireSession(w, r)
	if !ok {
		return
	}
	acct, err := s.accounts.AccountByID(r.Context(), sess.AccountID)
	if err != nil {
		log.Printf("account by id: %v", err)
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		return
	}
	kids, err := s.accounts.Children(r.Context(), sess.AccountID)
	if err != nil {
		log.Printf("children: %v", err)
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		return
	}
	data := testsPageData{Email: acct.Email, Children: kids, ActiveChildID: sess.ActiveChildID, TestGrades: []int{3, 4, 5}, Levels: ost.Levels}
	selectedID, _ := strconv.ParseInt(r.URL.Query().Get("child"), 10, 64)
	if selectedID == 0 {
		selectedID = sess.ActiveChildID
	}
	for i := range kids {
		if kids[i].ID == selectedID || (data.Selected == nil && i == len(kids)-1 && selectedID == 0) {
			rep, err := s.childReport(r, acct, kids[i])
			if err != nil {
				log.Printf("ost report: %v", err)
				http.Error(w, "something went wrong", http.StatusInternalServerError)
				return
			}
			data.Selected = rep
			break
		}
	}
	if data.Selected == nil && len(kids) > 0 {
		rep, err := s.childReport(r, acct, kids[0])
		if err != nil {
			log.Printf("ost report: %v", err)
			http.Error(w, "something went wrong", http.StatusInternalServerError)
			return
		}
		data.Selected = rep
	}
	w.Header().Set("Cache-Control", "no-store")
	s.render(w, "family-tests.html", data)
}

func (s *server) childReport(r *http.Request, acct account.Account, child account.Child) (*childReport, error) {
	oc := ost.Child{AccountID: acct.ID, ChildID: child.ID, Grade: child.Grade}
	history, err := s.tests.History(r.Context(), oc, 50)
	if err != nil {
		return nil, err
	}
	loc, err := time.LoadLocation(acct.Timezone)
	if err != nil {
		loc = time.UTC
	}
	rep := &childReport{Child: child, TestGrade: ost.NearestGrade(child.Grade)}
	rep.AnalysisGrade = ost.AnalysisGrade(history, rep.TestGrade)
	rep.Categories = ost.Categories(rep.AnalysisGrade)
	for _, h := range history {
		if !h.Finished && rep.InProgress == nil {
			hh := h
			rep.InProgress = &hh
		}
		rep.History = append(rep.History, attemptRow{Summary: h, When: h.StartedAt.In(loc).Format("Jan 2, 2006 3:04 PM")})
	}
	rep.Analysis = ost.Analyze(ost.OfGrade(history, rep.AnalysisGrade), rep.AnalysisGrade)
	return rep, nil
}

type attemptPageData struct {
	Child   account.Child
	Attempt ost.Attempt
	When    string
}

// handleFamilyAttempt shows one graded attempt item by item.
func (s *server) handleFamilyAttempt(w http.ResponseWriter, r *http.Request) {
	sess, ok := s.requireSession(w, r)
	if !ok {
		return
	}
	childID, err := strconv.ParseInt(r.PathValue("child"), 10, 64)
	if err != nil || childID <= 0 {
		http.NotFound(w, r)
		return
	}
	child, err := s.accounts.Child(r.Context(), sess.AccountID, childID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	a, err := s.tests.Attempt(r.Context(), ost.Child{AccountID: sess.AccountID, ChildID: child.ID, Grade: child.Grade}, r.PathValue("id"))
	if err != nil || a.Report == nil {
		if err != nil && !errors.Is(err, ost.ErrNotFound) {
			log.Printf("ost attempt: %v", err)
		}
		http.NotFound(w, r)
		return
	}
	acct, _ := s.accounts.AccountByID(r.Context(), sess.AccountID)
	loc, err := time.LoadLocation(acct.Timezone)
	if err != nil {
		loc = time.UTC
	}
	when := ""
	if t, err := time.Parse("2006-01-02T15:04:05.000000000Z", a.FinishedAt); err == nil {
		when = t.In(loc).Format("Jan 2, 2006 3:04 PM")
	}
	w.Header().Set("Cache-Control", "no-store")
	s.render(w, "family-attempt.html", attemptPageData{Child: child, Attempt: a, When: when})
}
