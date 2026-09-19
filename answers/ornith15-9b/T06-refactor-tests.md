# T06 refactor-tests (ornith15-9b)

finish_reason: stop  elapsed: 290.21s  usage: {"completion_tokens": 18943, "prompt_tokens": 237, "total_tokens": 19180, "prompt_tokens_details": {"cached_tokens": 29}}


## Answer

## Defects

### Rounding defects

1. **Hourly truncates instead of rounding.** `int(hours * float64(rate))` truncates toward zero (floors for positive values). `hours=0.999, rate=100` → `99.9` → `99`, but should be `100`. `hours=1.999, rate=100` → `199.9` → `199`, but should be `200`.

2. **Contingency uses the wrong constant *and* truncates.** `int(float64(rate) * 0.33)` uses `0.33` instead of `1/3`, and truncates.
   - `rate=300` → `300*0.33 = 99`, but `300/3 = 100`.
   - `rate=101` → `101*0.33 = 33.33` → `33`, but `101/3 = 33.67` → `34`.
   - `rate=600` → `198`, but should be `200`.

### Validation defects

3. **Negative hours** produces a negative fee (`hours=-1, rate=100` → `-100`).
4. **Negative rate** produces a negative fee (`rate=-100, kind="flat"` → `-100`).
5. **Negative cap is silently ignored.** The `cap > 0` guard means a negative cap disables capping instead of being rejected.
6. **Unknown kind silently returns `0`.** This hides bugs; it should be an error.

> Signature change: the function now returns `(int, error)` so invalid input can be reported. `cap == 0` is preserved as "no cap". Minor note: the `int` return can overflow for extremely large fees — inherent to the cents type.

## Refactored function

```go
package fee

import (
	"fmt"
	"math"
)

// FeeError is returned by Fee when an argument is invalid.
type FeeError struct {
	Kind  string // which argument was invalid
	Reason string
}

func (e *FeeError) Error() string {
	return fmt.Sprintf("Fee: invalid %s: %s", e.Kind, e.Reason)
}

// Fee computes the fee in integer cents.
//
// kind must be "flat", "hourly", or "contingency".
// rate is the per-unit rate in cents (non-negative).
// hours is the number of hours (non-negative).
// cap is the maximum fee in cents (non-negative; 0 means no cap).
func Fee(hours float64, rate int, kind string, cap int) (int, error) {
	var base float64
	switch kind {
	case "flat":
		base = float64(rate)
	case "hourly":
		base = float64(rate) * hours
	case "contingency":
		base = float64(rate) / 3
	default:
		return 0, &FeeError{Kind: "kind", Reason: fmt.Sprintf("unknown kind %q", kind)}
	}

	if hours < 0 || rate < 0 {
		return 0, &FeeError{Kind: "hours/rate", Reason: "must be non-negative"}
	}
	if cap < 0 {
		return 0, &FeeError{Kind: "cap", Reason: "must be non-negative"}
	}

	fee := int(math.Round(base))
	if cap > 0 && fee > cap {
		fee = cap
	}
	return fee, nil
}
```

## Test file

```go
package fee_test

import (
	"testing"

	"fee"
)

func TestFee(t *testing.T) {
	tests := []struct {
		name    string
		hours   float64
		rate    int
		kind    string
		cap     int
		want    int
		wantErr bool
	}{
		// Flat
		{"flat no cap", 0, 100, "flat", 0, 100, false},
		{"flat capped", 0, 100, "flat", 50, 50, false},
		{"flat at cap", 0, 50, "flat", 50, 50, false},
		{"flat below cap", 0, 100, "flat", 200, 100, false},
		{"flat zero rate", 0, 0, "flat", 0, 0, false},

		// Hourly
		{"hourly exact", 2, 100, "hourly", 0, 200, false},
		{"hourly half", 1.5, 100, "hourly", 0, 150, false},
		{"hourly rounds up", 0.999, 100, "hourly", 0, 100, false},
		{"hourly rounds up 2", 1.999, 100, "hourly", 0, 200, false},
		{"hourly rounds half up", 0.5, 101, "hourly", 0, 51, false},
		{"hourly zero", 0, 100, "hourly", 0, 0, false},
		{"hourly capped", 10, 100, "hourly", 140, 140, false},
		{"hourly rounds then caps", 1.5, 100, "hourly", 149, 149, false},
		{"hourly at cap", 1.5, 100, "hourly", 150, 150, false},

		// Contingency
		{"contingency 100", 0, 100, "contingency", 0, 33, false},
		{"contingency 300", 0, 300, "contingency", 0, 100, false},
		{"contingency 101", 0, 101, "contingency", 0, 34, false},
		{"contingency 600", 0, 600, "contingency", 0, 200, false},
		{"contingency zero", 0, 0, "contingency", 0, 0, false},
		{"contingency capped", 0, 300, "contingency", 50, 50, false},

		// Unknown kind
		{"unknown kind", 0, 100, "weekly", 0, 0, true},

		// Validation
		{"negative hours", -1, 100, "hourly", 0, 0, true},
		{"negative rate", 0, -100, "flat", 0, 0, true},
		{"negative cap", 0, 100, "flat", -5, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fee.Fee(tt.hours, tt.rate, tt.kind, tt.cap)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Fee(%v, %d, %q, %d) = nil, nil; want error", tt.hours, tt.rate, tt.kind, tt.cap)
				}
				return
			}
			if err != nil {
				t.Fatalf("Fee(%v, %d, %q, %d) returned unexpected error: %v", tt.hours, tt.rate, tt.kind, tt.cap, err)
			}
			if got != tt.want {
				t.Errorf("Fee(%v, %d, %q, %d) = %d, want %d", tt.hours, tt.rate, tt.kind, tt.cap, got, tt.want)
			}
		})
	}
}
```
