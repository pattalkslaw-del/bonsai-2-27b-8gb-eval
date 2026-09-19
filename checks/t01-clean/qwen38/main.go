package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"time"
)

type Subscriber interface {
	Add(ctx context.Context, email string) error
}

var ErrDuplicate = errors.New("duplicate subscriber")

var emailPattern = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9-]+(\.[a-zA-Z0-9-]+)*\.[a-zA-Z]{2,}$`)

type Server struct {
	store Subscriber
}

func NewServer(store Subscriber) *Server {
	return &Server{store: store}
}

func (s *Server) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writePlain(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if err := r.ParseForm(); err != nil {
		writePlain(w, http.StatusBadRequest, "invalid form")
		return
	}

	email := r.FormValue("email")
	if email == "" {
		writePlain(w, http.StatusBadRequest, "email is required")
		return
	}
	if !emailPattern.MatchString(email) {
		writePlain(w, http.StatusBadRequest, "malformed email address")
		return
	}

	if s.store == nil {
		slog.Error("subscribe failed", "email", email, "error", "subscriber store is not configured")
		writePlain(w, http.StatusInternalServerError, "internal server error")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	err := s.store.Add(ctx, email)
	switch {
	case err == nil:
		writePlain(w, http.StatusCreated, "subscribed")
	case errors.Is(err, ErrDuplicate):
		writePlain(w, http.StatusOK, "already subscribed")
	default:
		slog.Error("subscribe failed", "email", email, "error", err)
		writePlain(w, http.StatusInternalServerError, "internal server error")
	}
}

func writePlain(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	io.WriteString(w, body)
}

var _ http.HandlerFunc = (*Server)(nil).handleSubscribe
