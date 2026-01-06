package md2pdf

import (
	"context"
	"regexp"
)

// Precompiled regex patterns for performance.
var (
	// Line ending normalization
	crlfOrCR = regexp.MustCompile(`\r\n?`)

	// Compress multiple blank lines to max 2
	multipleBlankLines = regexp.MustCompile(`\n{3,}`)

	// Highlight syntax ==text==
	highlightPattern = regexp.MustCompile(`==(.*?)==`)
)

// markdownPreprocessor defines the contract for markdown preprocessing.
type markdownPreprocessor interface {
	PreprocessMarkdown(ctx context.Context, content string) string
}

// commonMarkPreprocessor applies transformations before CommonMark conversion.
type commonMarkPreprocessor struct{}

// PreprocessMarkdown applies all transformations to prepare Markdown for conversion.
func (p *commonMarkPreprocessor) PreprocessMarkdown(ctx context.Context, content string) string {
	// Check for cancellation before processing
	if ctx.Err() != nil {
		return content
	}

	content = normalizeLineEndings(content)
	content = convertHighlights(content)
	content = compressBlankLines(content)
	return content
}

// normalizeLineEndings converts \r\n and \r to \n.
func normalizeLineEndings(content string) string {
	return crlfOrCR.ReplaceAllString(content, "\n")
}

// compressBlankLines limits consecutive blank lines to 2 maximum.
func compressBlankLines(content string) string {
	return multipleBlankLines.ReplaceAllString(content, "\n\n")
}

// convertHighlights transforms ==text== to <mark>text</mark>.
func convertHighlights(content string) string {
	return highlightPattern.ReplaceAllString(content, "<mark>$1</mark>")
}
