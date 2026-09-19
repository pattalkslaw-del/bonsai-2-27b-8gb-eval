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
