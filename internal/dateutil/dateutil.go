// Package dateutil provides date format parsing utilities.
package dateutil

import (
	"errors"
	"fmt"
	"strings"
)

// ErrInvalidDateFormat indicates an invalid date format string.
var ErrInvalidDateFormat = errors.New("invalid date format")

// MaxDateFormatLength limits format string length to prevent abuse.
const MaxDateFormatLength = 50

// DefaultDateFormat is used when "auto" is specified without a format.
const DefaultDateFormat = "YYYY-MM-DD"

// dateTokens maps user-friendly tokens to Go time format components.
// Ordered by length descending for greedy matching.
var dateTokens = []struct {
	token string
	goFmt string
}{
	{"YYYY", "2006"},
	{"MMMM", "January"},
	{"MMM", "Jan"},
	{"YY", "06"},
	{"MM", "01"},
	{"DD", "02"},
	{"M", "1"},
	{"D", "2"},
}

// DatePresets provides named shortcuts for common date formats.
var DatePresets = map[string]string{
	"iso":      "YYYY-MM-DD",
	"european": "DD/MM/YYYY",
	"us":       "MM/DD/YYYY",
	"long":     "MMMM D, YYYY",
}

// ParseDateFormat converts a user-friendly format string to Go's time format.
// Tokens: YYYY, YY, MMMM, MMM, MM, M, DD, D
// Use brackets to escape literal text: [Date] preserves "Date" literally.
// Any non-token characters outside brackets are preserved as literals.
// Returns ErrInvalidDateFormat if the format is empty, too long, or has unclosed brackets.
func ParseDateFormat(format string) (string, error) {
	if format == "" {
		return "", fmt.Errorf("%w: format cannot be empty", ErrInvalidDateFormat)
	}
	if len(format) > MaxDateFormatLength {
		return "", fmt.Errorf("%w: format exceeds %d characters", ErrInvalidDateFormat, MaxDateFormatLength)
	}

	var result strings.Builder
	result.Grow(len(format) + 10) // Pre-allocate with some buffer

	i := 0
	for i < len(format) {
		// Handle bracket-escaped literal text
		if format[i] == '[' {
			end := strings.Index(format[i+1:], "]")
			if end == -1 {
				return "", fmt.Errorf("%w: unclosed bracket at position %d", ErrInvalidDateFormat, i)
			}
			// Copy content inside brackets literally
			result.WriteString(format[i+1 : i+1+end])
			i += end + 2 // Skip past closing bracket
			continue
		}

		matched := false

		// Try to match tokens (longest first due to slice order)
		for _, t := range dateTokens {
			if strings.HasPrefix(format[i:], t.token) {
				result.WriteString(t.goFmt)
				i += len(t.token)
				matched = true
				break
			}
		}

		if !matched {
			// Preserve literal character
			result.WriteByte(format[i])
			i++
		}
	}

	return result.String(), nil
}
