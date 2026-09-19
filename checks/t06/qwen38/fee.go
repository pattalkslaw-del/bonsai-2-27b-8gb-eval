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
