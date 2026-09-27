package main

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/tgeorge06/skilldojo/internal/account"
)

type familyData struct {
	Email         string
	Children      []account.Child
	ActiveChildID int64
	Error         string
	Grades        []int
}

// requireSession loads the session or redirects to /login and returns false.
func (s *server) requireSession(w http.ResponseWriter, r *http.Request) (account.Session, bool) {
	sess, ok := s.currentSession(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
	return sess, ok
}

func (s *server) familyPage(w http.ResponseWriter, r *http.Request, sess account.Session, status int, errMsg string) {
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
	if status != http.StatusOK {
		w.WriteHeader(status)
	}
	s.render(w, "family.html", familyData{
		Email: acct.Email, Children: kids, ActiveChildID: sess.ActiveChildID, Error: errMsg,
		Grades: []int{1, 2, 3, 4, 5},
	})
}

func (s *server) handleFamily(w http.ResponseWriter, r *http.Request) {
	sess, ok := s.requireSession(w, r)
	if !ok {
		return
	}
	s.familyPage(w, r, sess, http.StatusOK, "")
}

func (s *server) handleCreateChild(w http.ResponseWriter, r *http.Request) {
	sess, ok := s.requireSession(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, formBodyLimit)
	if err := r.ParseForm(); err != nil {
		s.familyPage(w, r, sess, http.StatusBadRequest, "That form did not come through. Please try again.")
		return
	}
	grade, _ := strconv.Atoi(r.PostFormValue("grade"))
	child, err := s.accounts.CreateChild(r.Context(), sess.AccountID, r.PostFormValue("nickname"), grade, s.now())
	if err != nil {
		s.familyPage(w, r, sess, http.StatusBadRequest, userMessage(err))
		return
	}
	// First profile becomes the active one so the parent can hand over the tablet.
	if sess.ActiveChildID == 0 {
		if err := s.accounts.SetActiveChild(r.Context(), sess.ID, sess.AccountID, child.ID); err != nil {
			log.Printf("set active child: %v", err)
		}
	}
	http.Redirect(w, r, "/family", http.StatusSeeOther)
}

// handleChildAction handles select / rename / delete for one child. Every
// store call carries the session's account id, so a guessed id is a 404.
func (s *server) handleChildAction(w http.ResponseWriter, r *http.Request) {
	sess, ok := s.requireSession(w, r)
	if !ok {
		return
	}
	childID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || childID <= 0 {
		http.NotFound(w, r)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, formBodyLimit)
	if err := r.ParseForm(); err != nil {
		s.familyPage(w, r, sess, http.StatusBadRequest, "That form did not come through. Please try again.")
		return
	}
	switch r.PostFormValue("action") {
	case "select":
		err = s.accounts.SetActiveChild(r.Context(), sess.ID, sess.AccountID, childID)
		if err == nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
	case "rename":
		grade, _ := strconv.Atoi(r.PostFormValue("grade"))
		err = s.accounts.UpdateChild(r.Context(), sess.AccountID, childID, r.PostFormValue("nickname"), grade)
	case "delete":
		err = s.accounts.DeleteChild(r.Context(), sess.AccountID, childID, s.now())
	default:
		s.familyPage(w, r, sess, http.StatusBadRequest, "Unknown action.")
		return
	}
	if errors.Is(err, account.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		s.familyPage(w, r, sess, http.StatusBadRequest, userMessage(err))
		return
	}
	http.Redirect(w, r, "/family", http.StatusSeeOther)
}

// userMessage turns validation errors into parent-facing text and hides
// anything else behind a generic message.
func userMessage(err error) string {
	msg := err.Error()
	if len(msg) > len("account: ") && msg[:len("account: ")] == "account: " {
		return msg[len("account: "):]
	}
	log.Printf("family: %v", err)
	return "Something went wrong. Please try again."
}
