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
