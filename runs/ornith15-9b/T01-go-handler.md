# T01 go-handler (ornith15-9b)

finish_reason: stop  elapsed: 73.53s  usage: {"completion_tokens": 5073, "prompt_tokens": 253, "total_tokens": 5326, "prompt_tokens_details": {"cached_tokens": 0}}

## Answer

```go
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
```

## Thinking

Let me write the Go code carefully.

Requirements:
- `handleSubscribe` is an `http.HandlerFunc`
- It's a method on a struct that holds an injected `Subscriber` interface
- POST only, else 405 with `Allow: POST`
- Parse form value `email`, reject empty/malformed with 400 and plain-text reason
- Call `store.Add(ctx, email)` on injected interface `Subscriber`
- `ErrDuplicate` → 200 with body `already subscribed`
- other error → 500, logged with slog, never echoed
- success → 201 with body `subscribed`
- 3-second timeout derived from request context

Let me think about the interface and error declarations.

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
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
func (s *subscriberStore) handleSubscribe(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	...
}
```

Wait, the requirement says "The handler must be a method on a struct that holds that interface." So the handler is a method. But `handleSubscribe` is named as the handler. Let me think about how to structure this.

The requirement: "Write a Go http.HandlerFunc named `handleSubscribe`". And "The handler must be a method on a struct that holds that interface."

So `handleSubscribe` is a method. But a method isn't itself an `http.HandlerFunc` unless it has the signature `func(w http.ResponseWriter, r *http.Request)`. 

Hmm. Let me reconsider. Perhaps the struct has a method `handleSubscribe` with signature `func(w http.ResponseWriter, r *http.Request)` — that IS an http.HandlerFunc (a method value). Actually a method with that signature can be used as an http.HandlerFunc.

But the requirement says "named `handleSubscribe`". Let me make `handleSubscribe` a method on the struct with the handler signature. That satisfies both: it's named handleSubscribe, it's a method on the struct, and it can be used as http.HandlerFunc.

Actually, let me reconsider. The cleanest interpretation:

```go
func (s *subscriberStore) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	...
}
```

This is a method. It can be used as an http.HandlerFunc via `s.handleSubscribe`. But the requirement says "named handleSubscribe" as an http.HandlerFunc. A method value `s.handleSubscribe` is an http.HandlerFunc.

Alternatively, maybe they want:

```go
func (s *subscriberStore) handleSubscribe() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) { ... }
}
```

Hmm, but that's awkward. Let me think about what's most natural and satisfies the requirements.

Actually, re-reading: "Write a Go http.HandlerFunc named `handleSubscribe` for a stdlib-only net/http server". "The handler must be a method on a struct that holds that interface."

I think the intent is: there's a struct that holds the `Subscriber` interface, and `handleSubscribe` is a method on that struct. The method itself is the handler (has the http.HandlerFunc signature). This is the most idiomatic.

Let me write it that way. The method `handleSubscribe` has signature `func(w http.ResponseWriter, r *http.Request)`.

Now let me think about the details:

1. POST only:
```go
if r.Method != http.MethodPost {
	w.Header().Set("Allow", http.MethodPost)
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	return
}
```

2. Parse form value `email`:
```go
if err := r.ParseForm(); err != nil {
	http.Error(w, "invalid form", http.StatusBadRequest)
	return
}
email := r.FormValue("email")
if email == "" {
	http.Error(w, "email is required", http.StatusBadRequest)
	return
}
if _, err := net.ParseEmail(email); err != nil {
	http.Error(w, "invalid email address", http.StatusBadRequest)
	return
}
```

Wait, `net.ParseEmail` was added in Go 1.22. Since we're on Go 1.24, that's fine.

3. Timeout derived from request context:
```go
ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
defer cancel()
```

4. Call store.Add:
```go
if err := s.store.Add(ctx, email); err != nil {
	if errors.Is(err, ErrDuplicate) {
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "already subscribed")
		return
	}
	s.logger.Error("failed to add subscriber", "email", email, "error", err)
	http.Error(w, "internal server error", http.StatusInternalServerError)
	return
}
w.WriteHeader(http.StatusCreated)
io.WriteString(w, "subscribed")
```

Wait, the requirement says "any other error is 500 and must be logged with slog, never echoed to the client." So I log the error. Good.

Let me reconsider the logging. `slog` — I need a logger. I'll inject it or create one. The requirement says "logged with slog". Let me inject a logger into the struct, or create one. Since it's a method on a struct that holds the interface, I could add a logger field. But the requirement only mentions the interface. Let me add a logger field to be safe, or use `slog.Default()`.

Actually, to be clean, I'll add a logger field. But the requirement says "a struct that holds that interface" — it doesn't say ONLY that interface. Adding a logger is fine.

Hmm, but let me reconsider. Maybe I should keep it minimal and use `slog.Default()`. But injecting is cleaner. Let me inject a logger.

Actually, let me reconsider whether to make the struct hold both the store and a logger. The requirement says "a struct that holds that interface" (the Subscriber interface). Adding a logger field is reasonable and doesn't violate anything.

Let me write the final code.

Let me reconsider the `handleSubscribe` naming. The requirement says "Write a Go http.HandlerFunc named `handleSubscribe`". If I make it a method `func (s *subscriberStore) handleSubscribe(...)`, then `s.handleSubscribe` is an http.HandlerFunc. That works.

But wait — should I return an http.HandlerFunc from a method? Let me re-read once more: "The handler must be a method on a struct that holds that interface." This strongly implies the handler IS a method. So the method has the http.HandlerFunc signature. Good.

Let me finalize.

```go
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
```

Wait, `mail.ParseAddress` returns a `*MailAddress` and an error. If the email has multiple addresses (comma-separated), `ParseAddress` returns an error. That's fine for rejecting malformed. Actually, `mail.ParseAddress` parses a single address. If there are multiple, it errors. Good for our purposes.

Hmm, but `mail.ParseAddress` might accept things like "John Doe <john@example.com>". That's a valid email address with a name. The requirement says "malformed addresses". Using `mail.ParseAddress` is reasonable. Alternatively, `net.ParseEmail` just validates the email syntax. Let me think.

The requirement: "Reject empty or malformed addresses with 400 and a plain-text reason." So we reject malformed. `mail.ParseAddress` is stricter (accepts names too). `net.ParseEmail` is more lenient (just checks the email format). 

I think either is acceptable. Let me use `mail.ParseAddress` since it's more thorough about "addresses". Actually, let me reconsider — `net.ParseEmail` is simpler and just validates the email. But `mail.ParseAddress` handles the full address parsing. 

Hmm, I'll go with `mail.ParseAddress` because it's the more complete validation for an "address". Actually, let me reconsider. The field is called `email`. A plain email like "john@example.com" — both parse it fine. A malformed one like "not-an-email" — both reject. A name-only "John Doe" — `mail.ParseAddress` accepts it (returns an address with empty email), `net.ParseEmail` rejects it. 

I think for robustness, `mail.ParseAddress` is good. But there's a subtlety: `mail.ParseAddress` accepts "John Doe" which has no email. That might be considered malformed for our purposes. But it's an edge case. Let me just use `mail.ParseAddress`.

Actually, I realize I should double check: does `mail.ParseAddress` require the email part? Let me recall. `mail.ParseAddress` parses a single address. It can parse "Name <email>" or just "email". If given "John Doe" without angle brackets, it parses the whole thing as the name and email is empty. So `addr.Address` would be "John Doe" and `addr.Email` would be "". 

For our purposes, this is an edge case. I'll keep `mail.ParseAddress`. It's a reasonable choice.

Let me reconsider whether to use `net.ParseEmail`. Given the field is "email", maybe `net.ParseEmail` is more appropriate as it strictly validates email format. But `mail.ParseAddress` is also fine. I'll go with `mail.ParseAddress` for thoroughness. Actually, you know what, let me use `mail.ParseAddress` — it's the standard way to parse email addresses in Go.

Now, about the logger. I inject `*slog.Logger`. That's fine.

Let me reconsider the structure. The requirement says "The handler must be a method on a struct that holds that interface." My struct `subscriberStore` holds `store Subscriber` and `logger *slog.Logger`. The method `handleSubscribe` is on this struct. Good.

Now, one concern: the requirement says "Return only the Go code, including the interface and error declarations. No prose." So I should output only the code, no explanation.

Let me also double-check the 405 handling. "Accept POST only; anything else returns 405 with an `Allow: POST` header." My code sets `Allow: POST` and returns 405. Good.

Let me reconsider: should I write the 405 body? `http.Error` writes a plain-text body. That's fine. The requirement doesn't specify a body for 405.

Let me reconsider the 400 case: "Reject empty or malformed addresses with 400 and a plain-text reason." `http.Error` writes a plain-text body with the reason. Good.

Let me reconsider the 500 case: "any other error is 500 and must be logged with slog, never echoed to the client." My code logs with `s.logger.Error(...)` and returns 500 without echoing the error. Good.

Now, one more consideration: the order of operations. Should I set up the timeout context before or after parsing the form? The requirement says "Apply a 3-second timeout to the store call, derived from the request context." So the timeout is specifically for the store call. I create the context right before the store call. That's fine. But actually, it might be cleaner to create it before the store call. Let me keep it right before.

Actually, wait — should the timeout context be created before parsing the form? The requirement says the timeout is "to the store call". So creating it right before the store call is correct. But there's a subtle issue: if I create the context before parsing the form and parsing takes too long, the timeout would apply to parsing too. But the requirement specifically says the timeout is for the store call. So creating it right before the store call is correct.

Hmm, but actually, it might be more robust to create the context earlier so that the whole handler is bounded. But the requirement is specific: "Apply a 3-second timeout to the store call, derived from the request context." So I'll create it right before the store call.

Let me finalize the code.

Actually, let me reconsider the logger injection. The requirement says the struct "holds that interface" (the Subscriber interface). It doesn't mention a logger. But I need a logger for slog. I could use `slog.Default()` instead of injecting a logger. But injecting is cleaner and more testable. Let me inject a logger.

Actually, let me reconsider. If I inject a logger, the struct holds both the interface and the logger. The requirement says "a struct that holds that interface" — this is satisfied since it holds the interface. Adding a logger doesn't violate anything.

Let me write the final version.

Actually, I want to reconsider whether the method should return an http.HandlerFunc or just be the handler. Let me re-read: "Write a Go http.HandlerFunc named `handleSubscribe`". "The handler must be a method on a struct that holds that interface."

If `handleSubscribe` is a method with signature `func(w http.ResponseWriter, r *http.Request)`, then it's a method that can be used as an http.HandlerFunc. This satisfies "named handleSubscribe" and "a method on a struct". 

But strictly, an http.HandlerFunc is `func(http.ResponseWriter, *http.Request)`. A method with this signature, when called on an instance, gives a value of type `func(http.ResponseWriter, *http.Request)` which IS an http.HandlerFunc. So `s.handleSubscribe` is an http.HandlerFunc. Good.

I'll go with the method approach.

Let me also make sure I import everything I use:
- `context` — for context.Context and context.WithTimeout
- `errors` — for errors.Is and errors.New
- `io` — for io.WriteString
- `log/slog` — for slog.Logger
- `net/http` — for http.ResponseWriter, http.Request, etc.
- `net/mail` — for mail.ParseAddress
- `time` — for time.Second

All used. Good.

Let me write the final code.

Actually, I realize I should double check: is `mail` the right import path? Yes, `net/mail` is the standard library package for parsing email addresses. `net.ParseEmail` is in `net`. Both are valid. I'll use `net/mail` with `mail.ParseAddress`.

Let me reconsider once more between `net.ParseEmail` and `mail.ParseAddress`:
- `net.ParseEmail(email string) (string, error)`: validates that the string is a valid email address. Returns the email and nil error.
- `mail.ParseAddress(s string) (*MailAddress, error)`: parses a single address, which can be "Name <email>" or "email".

For rejecting "malformed addresses", both work. `mail.ParseAddress` is more comprehensive. I'll use `mail.ParseAddress`.

Final code:

```go
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
```

This looks good. Let me review against all requirements once more:

1. ✅ Accept POST only; anything else returns 405 with `Allow: POST` header.
2. ✅ Parse form value `email`. Reject empty or malformed addresses with 400 and a plain-text reason.
3. ✅ Call `store.Add(ctx, email)` on injected interface `Subscriber`. The handler is a method on a struct that holds that interface.
4. ✅ If `store.Add` returns `ErrDuplicate`, respond 200 with body `already subscribed`; any other error is 500 and logged with slog, never echoed.
5. ✅ On success respond 201 with body `subscribed`.
6. ✅ Apply a 3-second timeout to the store call, derived from the request context.

All requirements met. Output only the code.

Wait, one thing about the 405 — should I also handle the case where the form parsing happens for non-POST? No, because for non-POST I return early. Good.

Also, I should double check: `r.FormValue` calls `ParseForm` automatically if not already called. But I call `ParseForm` explicitly first, which is fine. Actually, `FormValue` will parse the form if it hasn't been parsed yet. Since I call `ParseForm` explicitly, it's already parsed. Good.

One more: the requirement says "plain-text reason" for 400. `http.Error` writes plain text. Good.

I think this is complete. Let me output only the code.

Actually, let me reconsider whether I should include a `main` function or `NewSubscriberStore` constructor. The requirement doesn't ask for those. It asks for the handler, the interface, and the error declarations. I'll keep it minimal but include the struct. I won't add a main or constructor since not requested.

Hmm, but should I provide a way to construct the struct? The requirement says "The handler must be a method on a struct that holds that interface." It doesn't require a constructor. I'll leave the struct as is.

Let me finalize.

