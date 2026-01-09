package md2pdf

import (
	"errors"
	"os"
	"testing"
	"time"
)

func TestPageSettings_Validate(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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
			t.Parallel()

			got := isValidPageSize(tt.size)
			if got != tt.want {
				t.Errorf("isValidPageSize(%q) = %v, want %v", tt.size, got, tt.want)
			}
		})
	}
}

func TestIsValidOrientation(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

			got := isValidOrientation(tt.orientation)
			if got != tt.want {
				t.Errorf("isValidOrientation(%q) = %v, want %v", tt.orientation, got, tt.want)
			}
		})
	}
}

func TestFooter_Validate(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

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
	t.Parallel()

	t.Run("zero duration panics", func(t *testing.T) {
		t.Parallel()
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for zero duration")
			}
		}()
		WithTimeout(0)
	})

	t.Run("negative duration panics", func(t *testing.T) {
		t.Parallel()

		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for negative duration")
			}
		}()
		WithTimeout(-1 * time.Second)
	})
}

func TestIsValidHexColor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		color string
		want  bool
	}{
		// Valid colors
		{"#fff", true},
		{"#FFF", true},
		{"#000", true},
		{"#abc", true},
		{"#ABC", true},
		{"#123", true},
		{"#ffffff", true},
		{"#FFFFFF", true},
		{"#000000", true},
		{"#abcdef", true},
		{"#ABCDEF", true},
		{"#123456", true},
		{"#aAbBcC", true},
		{"#888888", true},
		{"#ff0000", true},

		// Invalid colors
		{"", false},
		{"fff", false},          // missing #
		{"#ff", false},          // too short
		{"#ffff", false},        // wrong length (4)
		{"#fffff", false},       // wrong length (5)
		{"#fffffff", false},     // too long (7)
		{"#ggg", false},         // invalid hex char
		{"#xyz", false},         // invalid hex char
		{"#12345g", false},      // invalid hex char
		{"red", false},          // color name not supported
		{"rgb(255,0,0)", false}, // rgb not supported
		{"#", false},            // just hash
		{" #fff", false},        // leading space
		{"#fff ", false},        // trailing space
	}

	for _, tt := range tests {
		t.Run(tt.color, func(t *testing.T) {
			t.Parallel()

			got := isValidHexColor(tt.color)
			if got != tt.want {
				t.Errorf("isValidHexColor(%q) = %v, want %v", tt.color, got, tt.want)
			}
		})
	}
}

func TestCover_Validate(t *testing.T) {
	t.Parallel()

	// Create a temp file for logo path tests
	tempDir := t.TempDir()
	existingLogo := tempDir + "/logo.png"
	if err := os.WriteFile(existingLogo, []byte("fake png"), 0644); err != nil {
		t.Fatalf("failed to create test logo: %v", err)
	}

	tests := []struct {
		name    string
		cover   *Cover
		wantErr error
	}{
		{
			name:    "nil is valid",
			cover:   nil,
			wantErr: nil,
		},
		{
			name:    "empty cover is valid",
			cover:   &Cover{},
			wantErr: nil,
		},
		{
			name: "all fields populated is valid",
			cover: &Cover{
				Title:        "My Document",
				Subtitle:     "A Comprehensive Guide",
				Logo:         existingLogo,
				Author:       "John Doe",
				AuthorTitle:  "Senior Developer",
				Organization: "Acme Corp",
				Date:         "2025-01-01",
				Version:      "v1.0.0",
			},
			wantErr: nil,
		},
		{
			name:    "logo URL accepted without file validation",
			cover:   &Cover{Logo: "https://example.com/logo.png"},
			wantErr: nil,
		},
		{
			name:    "logo http URL accepted",
			cover:   &Cover{Logo: "http://example.com/logo.png"},
			wantErr: nil,
		},
		{
			name:    "logo empty is valid",
			cover:   &Cover{Logo: ""},
			wantErr: nil,
		},
		{
			name:    "existing logo path is valid",
			cover:   &Cover{Logo: existingLogo},
			wantErr: nil,
		},
		{
			name:    "nonexistent logo path returns error",
			cover:   &Cover{Logo: "/nonexistent/path/to/logo.png"},
			wantErr: ErrCoverLogoNotFound,
		},
		{
			name:    "relative nonexistent logo returns error",
			cover:   &Cover{Logo: "nonexistent.png"},
			wantErr: ErrCoverLogoNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.cover.Validate()

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

func TestWatermark_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		watermark *Watermark
		wantErr   error
	}{
		{
			name:      "nil is valid",
			watermark: nil,
			wantErr:   nil,
		},
		{
			name:      "empty color is valid (uses default)",
			watermark: &Watermark{Text: "DRAFT", Color: ""},
			wantErr:   nil,
		},
		{
			name:      "valid 3-char hex color",
			watermark: &Watermark{Text: "DRAFT", Color: "#fff"},
			wantErr:   nil,
		},
		{
			name:      "valid 6-char hex color",
			watermark: &Watermark{Text: "DRAFT", Color: "#888888"},
			wantErr:   nil,
		},
		{
			name:      "valid uppercase hex color",
			watermark: &Watermark{Text: "DRAFT", Color: "#AABBCC"},
			wantErr:   nil,
		},
		{
			name:      "valid mixed case hex color",
			watermark: &Watermark{Text: "DRAFT", Color: "#aAbBcC"},
			wantErr:   nil,
		},
		{
			name:      "invalid color - missing hash",
			watermark: &Watermark{Text: "DRAFT", Color: "888888"},
			wantErr:   ErrInvalidWatermarkColor,
		},
		{
			name:      "invalid color - wrong length",
			watermark: &Watermark{Text: "DRAFT", Color: "#8888"},
			wantErr:   ErrInvalidWatermarkColor,
		},
		{
			name:      "invalid color - invalid hex char",
			watermark: &Watermark{Text: "DRAFT", Color: "#gggggg"},
			wantErr:   ErrInvalidWatermarkColor,
		},
		{
			name:      "invalid color - color name",
			watermark: &Watermark{Text: "DRAFT", Color: "red"},
			wantErr:   ErrInvalidWatermarkColor,
		},
		{
			name:      "invalid color - rgb format",
			watermark: &Watermark{Text: "DRAFT", Color: "rgb(255,0,0)"},
			wantErr:   ErrInvalidWatermarkColor,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.watermark.Validate()

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

func TestPageBreaks_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		pageBreaks *PageBreaks
		wantErr    error
	}{
		{
			name:       "nil is valid",
			pageBreaks: nil,
			wantErr:    nil,
		},
		{
			name:       "empty struct is valid (uses defaults)",
			pageBreaks: &PageBreaks{},
			wantErr:    nil,
		},
		{
			name:       "orphans 0 is valid (means use default)",
			pageBreaks: &PageBreaks{Orphans: 0},
			wantErr:    nil,
		},
		{
			name:       "widows 0 is valid (means use default)",
			pageBreaks: &PageBreaks{Widows: 0},
			wantErr:    nil,
		},
		{
			name:       "valid orphans at minimum",
			pageBreaks: &PageBreaks{Orphans: MinOrphans},
			wantErr:    nil,
		},
		{
			name:       "valid orphans at maximum",
			pageBreaks: &PageBreaks{Orphans: MaxOrphans},
			wantErr:    nil,
		},
		{
			name:       "valid widows at minimum",
			pageBreaks: &PageBreaks{Widows: MinWidows},
			wantErr:    nil,
		},
		{
			name:       "valid widows at maximum",
			pageBreaks: &PageBreaks{Widows: MaxWidows},
			wantErr:    nil,
		},
		{
			name:       "valid orphans and widows mid range",
			pageBreaks: &PageBreaks{Orphans: 3, Widows: 3},
			wantErr:    nil,
		},
		{
			name:       "valid with all heading breaks enabled",
			pageBreaks: &PageBreaks{BeforeH1: true, BeforeH2: true, BeforeH3: true, Orphans: 2, Widows: 2},
			wantErr:    nil,
		},
		{
			name:       "invalid orphans below minimum",
			pageBreaks: &PageBreaks{Orphans: -1},
			wantErr:    ErrInvalidOrphans,
		},
		{
			name:       "invalid orphans above maximum",
			pageBreaks: &PageBreaks{Orphans: MaxOrphans + 1},
			wantErr:    ErrInvalidOrphans,
		},
		{
			name:       "invalid orphans large value",
			pageBreaks: &PageBreaks{Orphans: 100},
			wantErr:    ErrInvalidOrphans,
		},
		{
			name:       "invalid widows below minimum",
			pageBreaks: &PageBreaks{Widows: -1},
			wantErr:    ErrInvalidWidows,
		},
		{
			name:       "invalid widows above maximum",
			pageBreaks: &PageBreaks{Widows: MaxWidows + 1},
			wantErr:    ErrInvalidWidows,
		},
		{
			name:       "invalid widows large value",
			pageBreaks: &PageBreaks{Widows: 100},
			wantErr:    ErrInvalidWidows,
		},
		{
			name:       "orphans validated before widows",
			pageBreaks: &PageBreaks{Orphans: -1, Widows: -1},
			wantErr:    ErrInvalidOrphans,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.pageBreaks.Validate()

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

func TestTOC_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		toc     *TOC
		wantErr error
	}{
		{
			name:    "nil is valid",
			toc:     nil,
			wantErr: nil,
		},
		{
			name:    "valid depth 1",
			toc:     &TOC{MaxDepth: 1},
			wantErr: nil,
		},
		{
			name:    "valid depth 3",
			toc:     &TOC{MaxDepth: 3},
			wantErr: nil,
		},
		{
			name:    "valid depth 6",
			toc:     &TOC{MaxDepth: 6},
			wantErr: nil,
		},
		{
			name:    "with title",
			toc:     &TOC{Title: "Table of Contents", MaxDepth: 3},
			wantErr: nil,
		},
		{
			name:    "min depth boundary",
			toc:     &TOC{MaxDepth: MinTOCDepth},
			wantErr: nil,
		},
		{
			name:    "max depth boundary",
			toc:     &TOC{MaxDepth: MaxTOCDepth},
			wantErr: nil,
		},
		{
			name:    "depth 0 invalid",
			toc:     &TOC{MaxDepth: 0},
			wantErr: ErrInvalidTOCDepth,
		},
		{
			name:    "depth 7 invalid",
			toc:     &TOC{MaxDepth: 7},
			wantErr: ErrInvalidTOCDepth,
		},
		{
			name:    "negative depth invalid",
			toc:     &TOC{MaxDepth: -1},
			wantErr: ErrInvalidTOCDepth,
		},
		{
			name:    "large negative depth invalid",
			toc:     &TOC{MaxDepth: -100},
			wantErr: ErrInvalidTOCDepth,
		},
		{
			name:    "large positive depth invalid",
			toc:     &TOC{MaxDepth: 100},
			wantErr: ErrInvalidTOCDepth,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.toc.Validate()

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
