package main

import (
	"errors"
	"os"

	picoloom "github.com/alnah/picoloom/v2"
	"github.com/alnah/picoloom/v2/internal/config"
)

// Error groups keep exit-code policy centralized so new sentinels can be added
// without duplicating branching logic in exitCodeFor.
var (
	browserExitErrors = []error{
		picoloom.ErrBrowserConnect,
		picoloom.ErrPageCreate,
		picoloom.ErrPageLoad,
		picoloom.ErrPDFGeneration,
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
		picoloom.ErrEmptyMarkdown,
		picoloom.ErrInvalidPageSize,
		picoloom.ErrInvalidOrientation,
		picoloom.ErrInvalidMargin,
		picoloom.ErrInvalidFooterPosition,
		picoloom.ErrInvalidWatermarkColor,
		picoloom.ErrInvalidTOCDepth,
		picoloom.ErrInvalidOrphans,
		picoloom.ErrInvalidWidows,
		picoloom.ErrStyleNotFound,
		picoloom.ErrTemplateSetNotFound,
		picoloom.ErrIncompleteTemplateSet,
		picoloom.ErrInvalidAssetPath,
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
