# T06 refactor-tests (qwen38)

finish_reason: stop  elapsed: 551.37s  usage: {"prompt_tokens": 275, "total_tokens": 39593, "completion_tokens": 39318, "prompt_tokens_details": null}

## Answer



## Defects in the original code

### Rounding defects

1. **`hourly` truncates instead of rounding**

   ```go
   t = int(hours * float64(rate))
   ```

   `int()` truncates toward zero. For money, the fee should be rounded to the nearest cent.

   Example:

   ```text
   1.25 hours * 10 cents/hour = 12.5 cents
   ```

   The original code returns `12`; it should return `13`.

2. **`float64` hours are imprecise for decimal billing values**

   Values such as `1.005` are not exactly representable in binary floating point. For example:

   ```text
   1.005 hours * 100 cents/hour = 100.5 cents
   ```

   The original code returns `100`. Even a naive `math.Round` fix can still misround some decimal hour values because of `float64` representation error. The refactor below uses `time.Duration` and integer arithmetic so that values such as `1.005h` are handled exactly.

3. **`contingency` truncates and uses an inexact float**

   ```go
   t = int(float64(rate) * 0.33)
   ```

   This both truncates and relies on binary floating point.

   Examples:

   ```text
   rate = 3
   3 * 0.33 = 0.99 cents
   ```

   The original code returns `0`; rounded to the nearest cent it should return `1`.

   ```text
   rate = 102
   102 * 0.33 = 33.66 cents
   ```

   The original code returns `33`; rounded to the nearest cent it should return `34`.

---

### Validation defects

1. **Unknown `kind` silently returns `0`**

   ```go
   } else {
       t = 0
   }
   ```

   This hides caller mistakes. Unknown kinds should return an error.

2. **Negative `rate` is accepted**

   A negative rate is not valid money and can produce negative fees.

3. **Negative `cap` is ignored**

   ```go
   if cap > 0 && t > cap {
       t = cap
   }
   ```

   A negative cap is invalid, but the original code simply ignores it.

4. **`cap == 0` cannot mean “zero cap”**

   Because the original code only applies the cap when `cap > 0`, `cap == 0` means “no cap”. The refactor uses `*int` so that:

   - `nil` means no cap
   - `0` means zero cap
   - negative values are rejected

5. **Negative `hours` is accepted for `hourly`**

   Negative hours can produce negative fees. The refactor rejects negative hours for `hourly`. `hours` is ignored for `flat` and `contingency`.

6. **`NaN` and `Inf` hours are not rejected**

   Converting `NaN` or `Inf` to `int` is not meaningful. The refactor uses `time.Duration`, which cannot represent `NaN` or `Inf`, and rejects negative durations for `hourly`.

---

## Refactored code

```go
package fee

import (
	"errors"
	"time"
)

// Kind identifies the fee calculation rule.
type Kind string

const (
	KindFlat        Kind = "flat"
	KindHourly      Kind = "hourly"
	KindContingency Kind = "contingency"
)

var (
	ErrUnknownKind   = errors.New("unknown fee kind")
	ErrNegativeRate  = errors.New("rate must be non-negative")
	ErrNegativeCap   = errors.New("cap must be non-negative")
	ErrNegativeHours = errors.New("hours must be non-negative")
)

// Fee returns a fee in integer cents.
//
// hours is only used for KindHourly; it is ignored for the other kinds.
//
// cap is nil when no cap applies. A zero cap caps the fee to zero.
func Fee(hours time.Duration, rate int, kind Kind, cap *int) (int, error) {
	switch kind {
	case KindFlat, KindContingency:
		// hours is ignored for these kinds.
	case KindHourly:
		if hours < 0 {
			return 0, ErrNegativeHours
		}
	default:
		return 0, ErrUnknownKind
	}

	if rate < 0 {
		return 0, ErrNegativeRate
	}

	if cap != nil && *cap < 0 {
		return 0, ErrNegativeCap
	}

	var fee int

	switch kind {
	case KindFlat:
		fee = rate
	case KindHourly:
		fee = hourlyFee(hours, rate)
	case KindContingency:
		fee = contingencyFee(rate)
	}

	if cap != nil && fee > *cap {
		fee = *cap
	}

	return fee, nil
}

// hourlyFee rounds rate*hours to the nearest cent.
//
// It uses integer arithmetic so decimal hour values such as 1.005h are not
// subject to float64 truncation or representation error.
func hourlyFee(hours time.Duration, rate int) int {
	if rate == 0 || hours == 0 {
		return 0
	}

	const hour = int64(time.Hour)

	// Round half up. All inputs are validated to be non-negative.
	return int((int64(rate) * int64(hours) + hour/2) / hour)
}

// contingencyFee calculates 33% of rate in cents, rounded to the nearest cent.
//
// It avoids float64 arithmetic entirely.
func contingencyFee(rate int) int {
	base := rate / 100
	rem := rate % 100

	return base*33 + (rem*33+50)/100
}
```

---

## Test file

```go
package fee

import (
	"errors"
	"testing"
	"time"
)

func intPtr(v int) *int {
	return &v
}

func TestFee(t *testing.T) {
	cases := []struct {
		name  string
		hours time.Duration
		rate  int
		kind  Kind
		cap   *int
		want  int
		err   error
	}{
		// Flat fees.
		{
			name:  "flat no cap",
			hours: 0,
			rate:  100,
			kind:  KindFlat,
			cap:   nil,
			want:  100,
		},
		{
			name:  "flat cap below fee",
			hours: 0,
			rate:  100,
			kind:  KindFlat,
			cap:   intPtr(50),
			want:  50,
		},
		{
			name:  "flat cap above fee",
			hours: 0,
			rate:  100,
			kind:  KindFlat,
			cap:   intPtr(200),
			want:  100,
		},
		{
			name:  "flat zero cap",
			hours: 0,
			rate:  100,
			kind:  KindFlat,
			cap:   intPtr(0),
			want:  0,
		},
		{
			name:  "flat negative rate",
			hours: 0,
			rate:  -1,
			kind:  KindFlat,
			cap:   nil,
			err:   ErrNegativeRate,
		},
		{
			name:  "flat negative cap",
			hours: 0,
			rate:  100,
			kind:  KindFlat,
			cap:   intPtr(-1),
			err:   ErrNegativeCap,
		},
		{
			name:  "flat ignores negative hours",
			hours: -time.Second,
			rate:  100,
			kind:  KindFlat,
			cap:   nil,
			want:  100,
		},

		// Hourly fees.
		{
			name:  "hourly whole hour",
			hours: time.Hour,
			rate:  100,
			kind:  KindHourly,
			cap:   nil,
			want:  100,
		},
		{
			name:  "hourly half hour",
			hours: time.Hour / 2,
			rate:  100,
			kind:  KindHourly,
			cap:   nil,
			want:  50,
		},
		{
			name:  "hourly zero hours",
			hours: 0,
			rate:  100,
			kind:  KindHourly,
			cap:   nil,
			want:  0,
		},
		{
			name:  "hourly zero rate",
			hours: time.Hour,
			rate:  0,
			kind:  KindHourly,
			cap:   nil,
			want:  0,
		},
		{
			name:  "hourly rounds half up",
			hours: time.Hour + 15*time.Minute,
			rate:  10,
			kind:  KindHourly,
			cap:   nil,
			want:  13,
		},
		{
			name:  "hourly 0.005h at 2c/h rounds up",
			hours: 18 * time.Second,
			rate:  2,
			kind:  KindHourly,
			cap:   nil,
			want:  1,
		},
		{
			name:  "hourly 1.005h at 100c/h rounds up",
			hours: time.Hour + 18*time.Second,
			rate:  100,
			kind:  KindHourly,
			cap:   nil,
			want:  101,
		},
		{
			name:  "hourly 1.004h at 100c/h rounds down",
			hours: time.Hour + 14400*time.Millisecond,
			rate:  100,
			kind:  KindHourly,
			cap:   nil,
			want:  100,
		},
		{
			name:  "hourly cap below fee",
			hours: time.Hour + 15*time.Minute,
			rate:  100,
			kind:  KindHourly,
			cap:   intPtr(120),
			want:  120,
		},
		{
			name:  "hourly zero cap",
			hours: time.Hour,
			rate:  100,
			kind:  KindHourly,
			cap:   intPtr(0),
			want:  0,
		},
		{
			name:  "hourly negative hours",
			hours: -time.Second,
			rate:  100,
			kind:  KindHourly,
			cap:   nil,
			err:   ErrNegativeHours,
		},
		{
			name:  "hourly negative rate",
			hours: time.Hour,
			rate:  -1,
			kind:  KindHourly,
			cap:   nil,
			err:   ErrNegativeRate,
		},
		{
			name:  "hourly negative cap",
			hours: time.Hour,
			rate:  100,
			kind:  KindHourly,
			cap:   intPtr(-1),
			err:   ErrNegativeCap,
		},

		// Contingency fees.
		{
			name:  "contingency 1c stays zero",
			hours: 0,
			rate:  1,
			kind:  KindContingency,
			cap:   nil,
			want:  0,
		},
		{
			name:  "contingency 2c rounds up",
			hours: 0,
			rate:  2,
			kind:  KindContingency,
			cap:   nil,
			want:  1,
		},
		{
			name:  "contingency 3c rounds up",
			hours: 0,
			rate:  3,
			kind:  KindContingency,
			cap:   nil,
			want:  1,
		},
		{
			name:  "contingency 100c",
			hours: 0,
			rate:  100,
			kind:  KindContingency,
			cap:   nil,
			want:  33,
		},
		{
			name:  "contingency 101c rounds down",
			hours: 0,
			rate:  101,
			kind:  KindContingency,
			cap:   nil,
			want:  33,
		},
		{
			name:  "contingency 102c rounds up",
			hours: 0,
			rate:  102,
			kind:  KindContingency,
			cap:   nil,
			want:  34,
		},
		{
			name:  "contingency zero",
			hours: 0,
			rate:  0,
			kind:  KindContingency,
			cap:   nil,
			want:  0,
		},
		{
			name:  "contingency cap below fee",
			hours: 0,
			rate:  100,
			kind:  KindContingency,
			cap:   intPtr(30),
			want:  30,
		},
		{
			name:  "contingency zero cap",
			hours: 0,
			rate:  100,
			kind:  KindContingency,
			cap:   intPtr(0),
			want:  0,
		},
		{
			name:  "contingency negative rate",
			hours: 0,
			rate:  -1,
			kind:  KindContingency,
			cap:   nil,
			err:   ErrNegativeRate,
		},
		{
			name:  "contingency negative cap",
			hours: 0,
			rate:  100,
			kind:  KindContingency,
			cap:   intPtr(-1),
			err:   ErrNegativeCap,
		},

		// Unknown kinds.
		{
			name:  "unknown kind",
			hours: 0,
			rate:  100,
			kind:  Kind("other"),
			cap:   nil,
			err:   ErrUnknownKind,
		},
		{
			name:  "empty kind",
			hours: 0,
			rate:  100,
			kind:  Kind(""),
			cap:   nil,
			err:   ErrUnknownKind,
		},
		{
			name:  "unknown kind takes precedence over negative rate",
			hours: 0,
			rate:  -1,
			kind:  Kind("other"),
			cap:   nil,
			err:   ErrUnknownKind,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Fee(tc.hours, tc.rate, tc.kind, tc.cap)

			if !errors.Is(err, tc.err) {
				t.Fatalf("Fee() error = %v, want %v", err, tc.err)
			}

			if err == nil && got != tc.want {
				t.Fatalf("Fee() = %d, want %d", got, tc.want)
			}
		})
	}
}
```

