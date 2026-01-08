package md2pdf

import (
	"errors"
	"testing"
	"time"
)

func TestPageSettings_Validate(t *testing.T) {
	tests := []struct {
		name    string
		ps      *PageSettings
		wantErr error
	}{
		{
			name:    "nil is valid (use defaults)",
			ps:      nil,
			wantErr: nil,
		},
		{
			name: "valid letter portrait",
			ps: &PageSettings{
				Size:        PageSizeLetter,
				Orientation: OrientationPortrait,
				Margin:      DefaultMargin,
			},
			wantErr: nil,
		},
		{
			name: "valid a4 landscape",
			ps: &PageSettings{
				Size:        PageSizeA4,
				Orientation: OrientationLandscape,
				Margin:      1.0,
			},
			wantErr: nil,
		},
		{
			name: "valid legal portrait",
			ps: &PageSettings{
				Size:        PageSizeLegal,
				Orientation: OrientationPortrait,
				Margin:      MinMargin,
			},
			wantErr: nil,
		},
		{
			name: "case insensitive size",
			ps: &PageSettings{
				Size:        "A4",
				Orientation: OrientationPortrait,
				Margin:      DefaultMargin,
			},
			wantErr: nil,
		},
		{
			name: "case insensitive orientation",
			ps: &PageSettings{
				Size:        PageSizeLetter,
				Orientation: "LANDSCAPE",
				Margin:      DefaultMargin,
			},
			wantErr: nil,
		},
		{
			name: "margin at minimum",
			ps: &PageSettings{
				Size:        PageSizeLetter,
				Orientation: OrientationPortrait,
				Margin:      MinMargin,
			},
			wantErr: nil,
		},
		{
			name: "margin at maximum",
			ps: &PageSettings{
				Size:        PageSizeLetter,
				Orientation: OrientationPortrait,
				Margin:      MaxMargin,
			},
			wantErr: nil,
		},
		{
			name: "invalid page size",
			ps: &PageSettings{
				Size:        "tabloid",
				Orientation: OrientationPortrait,
				Margin:      DefaultMargin,
			},
			wantErr: ErrInvalidPageSize,
		},
		{
			name: "empty page size",
			ps: &PageSettings{
				Size:        "",
				Orientation: OrientationPortrait,
				Margin:      DefaultMargin,
			},
			wantErr: ErrInvalidPageSize,
		},
		{
			name: "invalid orientation",
			ps: &PageSettings{
				Size:        PageSizeLetter,
				Orientation: "diagonal",
				Margin:      DefaultMargin,
			},
			wantErr: ErrInvalidOrientation,
		},
		{
			name: "empty orientation",
			ps: &PageSettings{
				Size:        PageSizeLetter,
				Orientation: "",
				Margin:      DefaultMargin,
			},
			wantErr: ErrInvalidOrientation,
		},
		{
			name: "margin below minimum",
			ps: &PageSettings{
				Size:        PageSizeLetter,
				Orientation: OrientationPortrait,
				Margin:      0.1,
			},
			wantErr: ErrInvalidMargin,
		},
		{
			name: "margin above maximum",
			ps: &PageSettings{
				Size:        PageSizeLetter,
				Orientation: OrientationPortrait,
				Margin:      5.0,
			},
			wantErr: ErrInvalidMargin,
		},
		{
			name: "margin zero",
			ps: &PageSettings{
				Size:        PageSizeLetter,
				Orientation: OrientationPortrait,
				Margin:      0,
			},
			wantErr: ErrInvalidMargin,
		},
		{
			name: "margin negative",
			ps: &PageSettings{
				Size:        PageSizeLetter,
				Orientation: OrientationPortrait,
				Margin:      -1.0,
			},
			wantErr: ErrInvalidMargin,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.ps.Validate()

			if tt.wantErr != nil {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestDefaultPageSettings(t *testing.T) {
	ps := DefaultPageSettings()

	if ps.Size != PageSizeLetter {
		t.Errorf("Size = %q, want %q", ps.Size, PageSizeLetter)
	}
	if ps.Orientation != OrientationPortrait {
		t.Errorf("Orientation = %q, want %q", ps.Orientation, OrientationPortrait)
	}
	if ps.Margin != DefaultMargin {
		t.Errorf("Margin = %v, want %v", ps.Margin, DefaultMargin)
	}

	// Ensure defaults are valid
	if err := ps.Validate(); err != nil {
		t.Errorf("DefaultPageSettings() not valid: %v", err)
	}
}

func TestIsValidPageSize(t *testing.T) {
	tests := []struct {
		size string
		want bool
	}{
		{"letter", true},
		{"a4", true},
		{"legal", true},
		{"LETTER", true},
		{"A4", true},
		{"Letter", true},
		{"tabloid", false},
		{"", false},
		{"a5", false},
	}

	for _, tt := range tests {
		t.Run(tt.size, func(t *testing.T) {
			got := isValidPageSize(tt.size)
			if got != tt.want {
				t.Errorf("isValidPageSize(%q) = %v, want %v", tt.size, got, tt.want)
			}
		})
	}
}

func TestIsValidOrientation(t *testing.T) {
	tests := []struct {
		orientation string
		want        bool
	}{
		{"portrait", true},
		{"landscape", true},
		{"PORTRAIT", true},
		{"LANDSCAPE", true},
		{"Portrait", true},
		{"diagonal", false},
		{"", false},
		{"auto", false},
	}

	for _, tt := range tests {
		t.Run(tt.orientation, func(t *testing.T) {
			got := isValidOrientation(tt.orientation)
			if got != tt.want {
				t.Errorf("isValidOrientation(%q) = %v, want %v", tt.orientation, got, tt.want)
			}
		})
	}
}

func TestFooter_Validate(t *testing.T) {
	tests := []struct {
		name    string
		footer  *Footer
		wantErr error
	}{
		{
			name:    "nil is valid",
			footer:  nil,
			wantErr: nil,
		},
		{
			name:    "empty position is valid",
			footer:  &Footer{Position: ""},
			wantErr: nil,
		},
		{
			name:    "left position is valid",
			footer:  &Footer{Position: "left"},
			wantErr: nil,
		},
		{
			name:    "center position is valid",
			footer:  &Footer{Position: "center"},
			wantErr: nil,
		},
		{
			name:    "right position is valid",
			footer:  &Footer{Position: "right"},
			wantErr: nil,
		},
		{
			name:    "case insensitive LEFT",
			footer:  &Footer{Position: "LEFT"},
			wantErr: nil,
		},
		{
			name:    "case insensitive Center",
			footer:  &Footer{Position: "Center"},
			wantErr: nil,
		},
		{
			name:    "invalid position returns error",
			footer:  &Footer{Position: "top"},
			wantErr: ErrInvalidFooterPosition,
		},
		{
			name:    "invalid position middle",
			footer:  &Footer{Position: "middle"},
			wantErr: ErrInvalidFooterPosition,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.footer.Validate()

			if tt.wantErr != nil {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestWithTimeoutPanic(t *testing.T) {
	t.Run("zero duration panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for zero duration")
			}
		}()
		WithTimeout(0)
	})

	t.Run("negative duration panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for negative duration")
			}
		}()
		WithTimeout(-1 * time.Second)
	})
}
