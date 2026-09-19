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
