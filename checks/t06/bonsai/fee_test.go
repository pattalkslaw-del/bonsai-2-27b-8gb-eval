package fee

import (
	"errors"
	"math"
	"testing"
)

func TestFee(t *testing.T) {
	tests := []struct {
		name    string
		hours   float64
		rate    Cents
		kind    FeeKind
		cap     Cents
		want    Cents
		wantErr error
	}{
		{
			name: "flat fee with no cap",
			hours: 0,
			rate: 100,
			kind: Flat,
			cap:  0,
			want: 100,
		},
		{
			name: "flat fee capped",
			hours: 0,
			rate: 1000,
			kind: Flat,
			cap:  500,
			want: 500,
		},
		{
			name: "flat fee with zero cap",
			hours: 0,
			rate: 1000,
			kind: Flat,
			cap:  0,
			want: 0,
		},
		{
			name: "flat fee with negative hours is invalid",
			hours: -1,
			rate: 100,
			kind: Flat,
			cap:  0,
			wantErr: ErrNegativeHours,
		},
		{
			name: "flat fee with negative rate is invalid",
			hours: 0,
			rate: -1,
			kind: Flat,
			cap:  0,
			wantErr: ErrNegativeRate,
		},
		{
			name: "flat fee with negative cap is invalid",
			hours: 0,
			rate: 100,
			kind: Flat,
			cap:  -1,
			wantErr: ErrNegativeCap,
		},
		{
			name: "hourly whole hours",
			hours: 2,
			rate: 100,
			kind: Hourly,
			cap:  0,
			want: 200,
		},
		{
			name: "hourly fractional hours round up",
			hours: 1.999,
			rate: 100,
			kind: Hourly,
			cap:  0,
			want: 200,
		},
		{
			name: "hourly half cent rounds up",
			hours: 0.005,
			rate: 100,
			kind: Hourly,
			cap:  0,
			want: 1,
		},
		{
			name: "hourly below half cent rounds down",
			hours: 0.0049,
			rate: 100,
			kind: Hourly,
			cap:  0,
			want: 0,
		},
		{
			name: "hourly fee capped",
			hours: 10,
			rate: 100,
			kind: Hourly,
			cap:  500,
			want: 500,
		},
		{
			name: "hourly fee with zero cap",
			hours: 10,
			rate: 100,
			kind: Hourly,
			cap:  0,
			want: 0,
		},
		{
			name: "hourly fee overflow",
			hours: 1e18,
			rate: Cents(math.MaxInt64),
			kind: Hourly,
			cap:  0,
			wantErr: ErrFeeOverflow,
		},
		{
			name: "contingency exact 33 percent",
			hours: 0,
			rate: 100,
			kind: Contingency,
			cap:  0,
			want: 33,
		},
		{
			name: "contingency rounds up",
			hours: 0,
			rate: 102,
			kind: Contingency,
			cap:  0,
			want: 34,
		},
		{
			name: "contingency small value rounds up",
			hours: 0,
			rate: 3,
			kind: Contingency,
			cap:  0,
			want: 1,
		},
		{
			name: "contingency small value rounds down",
			hours: 0,
			rate: 1,
			kind: Contingency,
			cap:  0,
			want: 0,
		},
		{
			name: "contingency capped",
			hours: 0,
			rate: 1000,
			kind: Contingency,
			cap:  30,
			want: 30,
		},
		{
			name: "contingency with zero cap",
			hours: 0,
			rate: 1000,
			kind: Contingency,
			cap:  0,
			want: 0,
		},
		{
			name: "contingency overflow",
			hours: 0,
			rate: Cents(math.MaxInt64),
			kind: Contingency,
			cap:  0,
			wantErr: ErrFeeOverflow,
		},
		{
			name: "zero rate",
			hours: 1,
			rate: 0,
			kind: Hourly,
			cap:  0,
			want: 0,
		},
		{
			name: "zero hours",
			hours: 0,
			rate: 100,
			kind: Hourly,
			cap:  0,
			want: 0,
		},
		{
			name: "unknown kind",
			hours: 0,
			rate: 100,
			kind: FeeKind("bad"),
			cap:  0,
			wantErr: ErrInvalidKind,
		},
		{
			name: "uppercase kind is invalid",
			hours: 0,
			rate: 100,
			kind: FeeKind("Flat"),
			cap:  0,
			wantErr: ErrInvalidKind,
		},
		{
			name: "negative hours",
			hours: -0.1,
			rate: 100,
			kind: Hourly,
			cap:  0,
			wantErr: ErrNegativeHours,
		},
		{
			name: "NaN hours",
			hours: math.NaN(),
			rate: 100,
			kind: Hourly,
			cap:  0,
			wantErr: ErrNonFiniteHours,
		},
		{
			name: "positive infinity hours",
			hours: math.Inf(1),
			rate: 100,
			kind: Hourly,
			cap:  0,
			wantErr: ErrNonFiniteHours,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Fee(tt.hours, tt.rate, tt.kind, tt.cap)

			if err != nil {
				if tt.wantErr == nil {
					t.Fatalf(
						"Fee(%v, %v, %v, %v) = %d, %v; want nil error",
						tt.hours, tt.rate, tt.kind, tt.cap, got, err,
					)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf(
						"Fee(%v, %v, %v, %v) error = %v; want %v",
						tt.hours, tt.rate, tt.kind, tt.cap, err, tt.wantErr,
					)
				}
				return
			}

			if tt.wantErr != nil {
				t.Fatalf(
					"Fee(%v, %v, %v, %v) = %d; want error %v",
					tt.hours, tt.rate, tt.kind, tt.cap, got, tt.wantErr,
				)
			}

			if got != tt.want {
				t.Fatalf(
					"Fee(%v, %v, %v, %v) = %d; want %d",
					tt.hours, tt.rate, tt.kind, tt.cap, got, tt.want,
				)
			}
		})
	}
}
