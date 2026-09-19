package fee

import (
	"errors"
	"math"
)

// Cents is an integer number of cents.
type Cents int64

// FeeKind identifies the fee calculation method.
type FeeKind string

const (
	Flat        FeeKind = "flat"
	Hourly      FeeKind = "hourly"
	Contingency FeeKind = "contingency"
)

var (
	ErrInvalidKind    = errors.New("invalid fee kind")
	ErrNonFiniteHours = errors.New("hours must be finite")
	ErrNegativeHours  = errors.New("hours must be non-negative")
	ErrNegativeRate   = errors.New("rate must be non-negative")
	ErrNegativeCap    = errors.New("cap must be non-negative")
	ErrFeeOverflow    = errors.New("fee overflows")
)

// Fee returns the fee in cents.
//
// Rounding policy: fractional cents are rounded to the nearest cent,
// with halves rounding up.
func Fee(hours float64, rate Cents, kind FeeKind, cap Cents) (Cents, error) {
	if kind != Flat && kind != Hourly && kind != Contingency {
		return 0, ErrInvalidKind
	}

	if math.IsNaN(hours) || math.IsInf(hours, 0) {
		return 0, ErrNonFiniteHours
	}
	if hours < 0 {
		return 0, ErrNegativeHours
	}
	if rate < 0 {
		return 0, ErrNegativeRate
	}
	if cap < 0 {
		return 0, ErrNegativeCap
	}

	var fee Cents

	switch kind {
	case Flat:
		fee = rate

	case Hourly:
		rounded := math.Round(hours * float64(rate))
		if math.IsInf(rounded, 0) || rounded > math.MaxInt64 || rounded == math.MaxInt64 {
			return 0, ErrFeeOverflow
		}
		fee = Cents(rounded)

	case Contingency:
		// 33% of rate, rounded to the nearest cent.
		if rate > (math.MaxInt64-50)/33 {
			return 0, ErrFeeOverflow
		}
		fee = Cents((int64(rate)*33 + 50) / 100)
	}

	if fee > cap {
		fee = cap
	}

	return fee, nil
}
