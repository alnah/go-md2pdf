package main

import (
	"errors"
	"os"

	md2pdf "github.com/alnah/go-md2pdf"
	"github.com/alnah/go-md2pdf/internal/config"
)

// Error groups keep exit-code policy centralized so new sentinels can be added
// without duplicating branching logic in exitCodeFor.
var (
	browserExitErrors = []error{
		md2pdf.ErrBrowserConnect,
		md2pdf.ErrPageCreate,
		md2pdf.ErrPageLoad,
		md2pdf.ErrPDFGeneration,
	}
	ioExitErrors = []error{
		os.ErrNotExist,
		os.ErrPermission,
		ErrReadMarkdown,
		ErrReadCSS,
		ErrWritePDF,
		ErrNoInput,
	}
	usageExitErrors = []error{
		config.ErrConfigNotFound,
		config.ErrConfigParse,
		config.ErrFieldTooLong,
		ErrConfigCommandUsage,
		ErrConfigInitNeedsTTY,
		ErrConfigInitExists,
		ErrConfigInitBusy,
		md2pdf.ErrEmptyMarkdown,
		md2pdf.ErrInvalidPageSize,
		md2pdf.ErrInvalidOrientation,
		md2pdf.ErrInvalidMargin,
		md2pdf.ErrInvalidFooterPosition,
		md2pdf.ErrInvalidWatermarkColor,
		md2pdf.ErrInvalidTOCDepth,
		md2pdf.ErrInvalidOrphans,
		md2pdf.ErrInvalidWidows,
		md2pdf.ErrStyleNotFound,
		md2pdf.ErrTemplateSetNotFound,
		md2pdf.ErrIncompleteTemplateSet,
		md2pdf.ErrInvalidAssetPath,
		ErrUnsupportedShell,
	}
)

// Exit codes for md2pdf CLI.
// Follows Unix conventions: 0=success, 1=general, 2=usage, and custom codes < 126.
const (
	ExitSuccess = 0 // Successful conversion
	ExitGeneral = 1 // General/unexpected error
	ExitUsage   = 2 // Invalid flags, config, or validation
	ExitIO      = 3 // File not found, permission denied
	ExitBrowser = 4 // Browser/Chrome errors
)

// exitCodeFor returns the appropriate exit code for an error.
// It uses errors.Is to check wrapped errors, so callers must use fmt.Errorf("%w", err).
func exitCodeFor(err error) int {
	if err == nil {
		return ExitSuccess
	}

	if matchesAny(err, browserExitErrors) {
		return ExitBrowser
	}

	if matchesAny(err, ioExitErrors) {
		return ExitIO
	}

	if matchesAny(err, usageExitErrors) {
		return ExitUsage
	}

	return ExitGeneral
}

// matchesAny keeps wrapped-error matching uniform across exit-code categories.
func matchesAny(err error, candidates []error) bool {
	for _, target := range candidates {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}
