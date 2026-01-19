package dateutil

import (
	"errors"
	"testing"
	"time"
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

func TestResolveDate(t *testing.T) {
	t.Parallel()

	// Fixed time for deterministic tests: 2024-03-15
	fixedTime := time.Date(2024, 3, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name    string
		value   string
		want    string
		wantErr error
	}{
		// Passthrough cases (non-auto values)
		{
			name:  "empty string passthrough",
			value: "",
			want:  "",
		},
		{
			name:  "literal date passthrough",
			value: "2024-01-01",
			want:  "2024-01-01",
		},
		{
			name:  "arbitrary text passthrough",
			value: "Q1 2024",
			want:  "Q1 2024",
		},
		// Auto with default format
		{
			name:  "auto uses default ISO format",
			value: "auto",
			want:  "2024-03-15",
		},
		{
			name:  "AUTO is case insensitive",
			value: "AUTO",
			want:  "2024-03-15",
		},
		{
			name:  "Auto mixed case works",
			value: "Auto",
			want:  "2024-03-15",
		},
		// Auto with custom format
		{
			name:  "auto:YYYY-MM-DD explicit ISO",
			value: "auto:YYYY-MM-DD",
			want:  "2024-03-15",
		},
		{
			name:  "auto:DD/MM/YYYY European format",
			value: "auto:DD/MM/YYYY",
			want:  "15/03/2024",
		},
		{
			name:  "auto:MM/DD/YYYY US format",
			value: "auto:MM/DD/YYYY",
			want:  "03/15/2024",
		},
		{
			name:  "auto:MMMM D, YYYY long format",
			value: "auto:MMMM D, YYYY",
			want:  "March 15, 2024",
		},
		{
			name:  "auto:MMM YYYY short month with year",
			value: "auto:MMM YYYY",
			want:  "Mar 2024",
		},
		// Preset formats
		{
			name:  "auto:iso preset",
			value: "auto:iso",
			want:  "2024-03-15",
		},
		{
			name:  "auto:european preset",
			value: "auto:european",
			want:  "15/03/2024",
		},
		{
			name:  "auto:us preset",
			value: "auto:us",
			want:  "03/15/2024",
		},
		{
			name:  "auto:long preset",
			value: "auto:long",
			want:  "March 15, 2024",
		},
		{
			name:  "preset is case insensitive",
			value: "auto:ISO",
			want:  "2024-03-15",
		},
		{
			name:  "preset mixed case works",
			value: "auto:European",
			want:  "15/03/2024",
		},
		// Bracket escape syntax
		{
			name:  "auto with bracket-escaped literal",
			value: "auto:[Date]: YYYY-MM-DD",
			want:  "Date: 2024-03-15",
		},
		// Error cases
		{
			name:    "auto: with empty format returns error",
			value:   "auto:",
			wantErr: ErrInvalidDateFormat,
		},
		{
			name:    "autoX invalid syntax returns error",
			value:   "autoX",
			wantErr: ErrInvalidDateFormat,
		},
		{
			name:    "auto123 invalid syntax returns error",
			value:   "auto123",
			wantErr: ErrInvalidDateFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ResolveDate(tt.value, fixedTime)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ResolveDate(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("ResolveDate(%q) unexpected error: %v", tt.value, err)
				return
			}

			if got != tt.want {
				t.Errorf("ResolveDate(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}
