package md2pdf

import (
	"fmt"
	"strings"
	"time"

	"github.com/alnah/go-md2pdf/internal/dateutil"
)

// ResolveDate handles "auto" and "auto:FORMAT" syntax for date values.
// - "auto" → current date in YYYY-MM-DD format
// - "auto:FORMAT" → current date in custom format (e.g., "auto:DD/MM/YYYY")
// - "auto:preset" → current date using named preset (iso, european, us, long)
// - any other value → returned unchanged (passthrough)
//
// The time parameter allows injecting a fixed time for testing.
func ResolveDate(value string, t time.Time) (string, error) {
	lower := strings.ToLower(value)

	// Not an auto value - passthrough
	if !strings.HasPrefix(lower, "auto") {
		return value, nil
	}

	// Exact "auto" - use default format
	if lower == "auto" {
		goFmt, err := dateutil.ParseDateFormat(dateutil.DefaultDateFormat)
		if err != nil {
			return "", err
		}
		return t.Format(goFmt), nil
	}

	// Must be "auto:something"
	if !strings.HasPrefix(lower, "auto:") {
		return "", fmt.Errorf("%w: invalid auto syntax %q, use \"auto\" or \"auto:FORMAT\"", dateutil.ErrInvalidDateFormat, value)
	}

	// Extract format part (preserve original case for format tokens)
	formatPart := value[5:] // Skip "auto:"
	if formatPart == "" {
		return "", fmt.Errorf("%w: format cannot be empty after \"auto:\"", dateutil.ErrInvalidDateFormat)
	}

	// Check for preset (case-insensitive)
	if preset, ok := dateutil.DatePresets[strings.ToLower(formatPart)]; ok {
		formatPart = preset
	}

	// Parse and apply format
	goFmt, err := dateutil.ParseDateFormat(formatPart)
	if err != nil {
		return "", err
	}

	return t.Format(goFmt), nil
}
