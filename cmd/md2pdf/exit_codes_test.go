package main

// Notes:
// - exitCodeFor: we test all sentinel errors from md2pdf and config packages,
//   plus wrapped errors to verify errors.Is() chain works correctly.
// - Exit code constants: we verify Unix conventions (0=success, 1=general, 2=usage)
//   and custom codes are below 126.
// These are acceptable gaps: we test observable behavior, not implementation details.

import (
	"errors"
	"fmt"
	"os"
	"testing"

	md2pdf "github.com/alnah/go-md2pdf"
	"github.com/alnah/go-md2pdf/internal/config"
)

// ---------------------------------------------------------------------------
// TestExitCodeFor - Error to exit code mapping
// ---------------------------------------------------------------------------

func TestExitCodeFor(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		err  error
		want int
	}{
		// Success
		{"nil error", nil, ExitSuccess},

		// Browser errors (exit 4)
		{"browser connect", md2pdf.ErrBrowserConnect, ExitBrowser},
		{"page create", md2pdf.ErrPageCreate, ExitBrowser},
		{"page load", md2pdf.ErrPageLoad, ExitBrowser},
		{"pdf generation", md2pdf.ErrPDFGeneration, ExitBrowser},
		{"wrapped browser connect", fmt.Errorf("failed: %w", md2pdf.ErrBrowserConnect), ExitBrowser},

		// I/O errors (exit 3)
		{"file not exist", os.ErrNotExist, ExitIO},
		{"permission denied", os.ErrPermission, ExitIO},
		{"read markdown", ErrReadMarkdown, ExitIO},
		{"read css", ErrReadCSS, ExitIO},
		{"write pdf", ErrWritePDF, ExitIO},
		{"no input", ErrNoInput, ExitIO},
		{"wrapped file not exist", fmt.Errorf("reading: %w", os.ErrNotExist), ExitIO},

		// Usage/config/validation errors (exit 2)
		{"config not found", config.ErrConfigNotFound, ExitUsage},
		{"config parse", config.ErrConfigParse, ExitUsage},
		{"field too long", config.ErrFieldTooLong, ExitUsage},
		{"empty markdown", md2pdf.ErrEmptyMarkdown, ExitUsage},
		{"invalid page size", md2pdf.ErrInvalidPageSize, ExitUsage},
		{"invalid orientation", md2pdf.ErrInvalidOrientation, ExitUsage},
		{"invalid margin", md2pdf.ErrInvalidMargin, ExitUsage},
		{"invalid footer position", md2pdf.ErrInvalidFooterPosition, ExitUsage},
		{"invalid watermark color", md2pdf.ErrInvalidWatermarkColor, ExitUsage},
		{"invalid toc depth", md2pdf.ErrInvalidTOCDepth, ExitUsage},
		{"invalid orphans", md2pdf.ErrInvalidOrphans, ExitUsage},
		{"invalid widows", md2pdf.ErrInvalidWidows, ExitUsage},
		{"style not found", md2pdf.ErrStyleNotFound, ExitUsage},
		{"template set not found", md2pdf.ErrTemplateSetNotFound, ExitUsage},
		{"incomplete template set", md2pdf.ErrIncompleteTemplateSet, ExitUsage},
		{"invalid asset path", md2pdf.ErrInvalidAssetPath, ExitUsage},
		{"unsupported shell", ErrUnsupportedShell, ExitUsage},
		{"wrapped config parse", fmt.Errorf("loading: %w", config.ErrConfigParse), ExitUsage},

		// General errors (exit 1)
		{"unknown error", errors.New("something unexpected"), ExitGeneral},
		{"wrapped unknown", fmt.Errorf("context: %w", errors.New("unknown")), ExitGeneral},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := exitCodeFor(tt.err)
			if got != tt.want {
				t.Errorf("exitCodeFor(%v) = %d, want %d", tt.err, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestExitCodeConstants - Unix convention compliance
// ---------------------------------------------------------------------------

func TestExitCodeConstants(t *testing.T) {
	t.Parallel()
	// Verify exit codes follow Unix conventions
	if ExitSuccess != 0 {
		t.Errorf("ExitSuccess = %d, want 0", ExitSuccess)
	}
	if ExitGeneral != 1 {
		t.Errorf("ExitGeneral = %d, want 1", ExitGeneral)
	}
	if ExitUsage != 2 {
		t.Errorf("ExitUsage = %d, want 2", ExitUsage)
	}

	// Verify custom codes are below 126 (Unix convention)
	if ExitIO >= 126 {
		t.Errorf("ExitIO = %d, should be < 126", ExitIO)
	}
	if ExitBrowser >= 126 {
		t.Errorf("ExitBrowser = %d, should be < 126", ExitBrowser)
	}
}
