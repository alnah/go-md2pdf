package md2pdf

import (
	"errors"
	"testing"
	"time"

	"github.com/alnah/go-md2pdf/internal/dateutil"
)

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
			wantErr: dateutil.ErrInvalidDateFormat,
		},
		{
			name:    "autoX invalid syntax returns error",
			value:   "autoX",
			wantErr: dateutil.ErrInvalidDateFormat,
		},
		{
			name:    "auto123 invalid syntax returns error",
			value:   "auto123",
			wantErr: dateutil.ErrInvalidDateFormat,
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
