package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/mail"
	"strings"
	"time"
)

type Subscriber interface {
	Add(ctx context.Context, email string) error
}

var ErrDuplicate = errors.New("duplicate subscriber")

type SubscribeHandler struct {
	store Subscriber
}

func (h *SubscribeHandler) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		writePlain(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	if err := r.ParseForm(); err != nil {
		writePlain(w, http.StatusBadRequest, "invalid form")
		return
	}

	email := strings.TrimSpace(r.FormValue("email"))
	if email == "" {
		writePlain(w, http.StatusBadRequest, "email is required")
		return
	}

	if _, err := net/mail.ParseAddress(email); err != nil {
		writePlain(w, http.StatusBadRequest, "invalid email address")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	if err := h.store.Add(ctx, email); err != nil {
		if errors.Is(err, ErrDuplicate) {
			writePlain(w, http.StatusOK, "already subscribed")
			return
		}
		slog.Error("failed to add subscriber", "email", email, "error", err)
		writePlain(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	writePlain(w, http.StatusCreated, "subscribed")
}

func writePlain(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	fmt.Fprint(w, body)
}
