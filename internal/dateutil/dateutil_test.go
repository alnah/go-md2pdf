package dateutil

import (
	"errors"
	"testing"
)

func TestParseDateFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		format  string
		want    string
		wantErr error
	}{
		// Valid token conversions
		{
			name:   "YYYY converts to Go year format",
			format: "YYYY",
			want:   "2006",
		},
		{
			name:   "YY converts to short year format",
			format: "YY",
			want:   "06",
		},
		{
			name:   "MMMM converts to full month name",
			format: "MMMM",
			want:   "January",
		},
		{
			name:   "MMM converts to short month name",
			format: "MMM",
			want:   "Jan",
		},
		{
			name:   "MM converts to zero-padded month",
			format: "MM",
			want:   "01",
		},
		{
			name:   "M converts to non-padded month",
			format: "M",
			want:   "1",
		},
		{
			name:   "DD converts to zero-padded day",
			format: "DD",
			want:   "02",
		},
		{
			name:   "D converts to non-padded day",
			format: "D",
			want:   "2",
		},
		// Combined formats
		{
			name:   "ISO date format YYYY-MM-DD",
			format: "YYYY-MM-DD",
			want:   "2006-01-02",
		},
		{
			name:   "European format DD/MM/YYYY",
			format: "DD/MM/YYYY",
			want:   "02/01/2006",
		},
		{
			name:   "US format MM/DD/YYYY",
			format: "MM/DD/YYYY",
			want:   "01/02/2006",
		},
		{
			name:   "long format with full month name",
			format: "MMMM D, YYYY",
			want:   "January 2, 2006",
		},
		{
			name:   "short month with year",
			format: "MMM YYYY",
			want:   "Jan 2006",
		},
		// Literal preservation
		{
			name:   "preserves literal separators",
			format: "YYYY/MM/DD",
			want:   "2006/01/02",
		},
		{
			name:   "preserves literal text without token chars",
			format: "(YYYY-MM-DD)",
			want:   "(2006-01-02)",
		},
		{
			name:   "preserves spaces",
			format: "DD MM YYYY",
			want:   "02 01 2006",
		},
		{
			name:   "D in text is matched as day token",
			format: "Date: YYYY",
			want:   "2ate: 2006", // D -> 2 (day), use [Date] to escape
		},
		// Bracket escape syntax
		{
			name:   "brackets preserve literal text",
			format: "[Date]: YYYY",
			want:   "Date: 2006",
		},
		{
			name:   "brackets preserve tokens as literals",
			format: "[YYYY]-MM-DD",
			want:   "YYYY-01-02",
		},
		{
			name:   "multiple bracket groups",
			format: "[Day]: D [Month]: M",
			want:   "Day: 2 Month: 1",
		},
		{
			name:   "empty brackets are valid",
			format: "YYYY[]MM",
			want:   "200601",
		},
		{
			name:   "brackets with special characters",
			format: "[Date/Time]: YYYY-MM-DD",
			want:   "Date/Time: 2006-01-02",
		},
		{
			name:    "unclosed bracket returns error",
			format:  "[Date YYYY",
			wantErr: ErrInvalidDateFormat,
		},
		{
			name:   "nested-looking brackets use first close",
			format: "[a[b]c",
			want:   "a[bc", // [a[b] is the escaped part, c is literal
		},
		// Edge cases
		{
			name:    "empty format returns error",
			format:  "",
			wantErr: ErrInvalidDateFormat,
		},
		{
			name:    "format exceeding max length returns error",
			format:  string(make([]byte, MaxDateFormatLength+1)),
			wantErr: ErrInvalidDateFormat,
		},
		{
			name:   "format at max length is valid",
			format: string(make([]byte, MaxDateFormatLength)),
			want:   string(make([]byte, MaxDateFormatLength)),
		},
		{
			name:   "only literal characters",
			format: "---",
			want:   "---",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseDateFormat(tt.format)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ParseDateFormat(%q) error = %v, wantErr %v", tt.format, err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseDateFormat(%q) unexpected error: %v", tt.format, err)
				return
			}

			if got != tt.want {
				t.Errorf("ParseDateFormat(%q) = %q, want %q", tt.format, got, tt.want)
			}
		})
	}
}
