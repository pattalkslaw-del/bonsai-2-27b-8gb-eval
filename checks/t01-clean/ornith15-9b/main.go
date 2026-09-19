package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/mail"
	"time"
)

// Subscriber is the injected store interface.
type Subscriber interface {
	Add(ctx context.Context, email string) error
}

// ErrDuplicate is returned by Subscriber.Add when the email is already known.
var ErrDuplicate = errors.New("duplicate subscriber")

// subscriberStore holds the injected store and its logger.
type subscriberStore struct {
	store Subscriber
	logger *slog.Logger
}

// handleSubscribe subscribes an email address.
func (s *subscriberStore) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form data", http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")
	if email == "" {
		http.Error(w, "email is required", http.StatusBadRequest)
		return
	}
	if _, err := mail.ParseAddress(email); err != nil {
		http.Error(w, "invalid email address", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	if err := s.store.Add(ctx, email); err != nil {
		if errors.Is(err, ErrDuplicate) {
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, "already subscribed")
			return
		}
		s.logger.Error("failed to subscribe", "email", email, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_, _ = io.WriteString(w, "subscribed")
}
