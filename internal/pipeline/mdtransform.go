package pipeline

import (
	"context"
	"regexp"
	"strings"
)

// Highlight placeholders use Unicode Private Use Area characters.
// These are guaranteed to not conflict with any standard characters
// and will pass through Goldmark unchanged (no WithUnsafe needed).
// Post-processing converts these to <mark> tags after HTML generation.
const (
	MarkStartPlaceholder = "\uE000" // U+E000: Private Use Area start
	MarkEndPlaceholder   = "\uE001" // U+E001: Private Use Area end
)

// Precompiled regex patterns for performance.
var (
	// Line ending normalization
	crlfOrCR = regexp.MustCompile(`\r\n?`)

	// Compress multiple blank lines to max 2
	multipleBlankLines = regexp.MustCompile(`\n{3,}`)

	// Highlight syntax ==text==
	highlightPattern = regexp.MustCompile(`==(.*?)==`)

	// YAML frontmatter at document start.
	// Matches paired --- delimiters on their own lines.
	// Pattern requires both delimiters to avoid stripping incomplete blocks.
	// Only matches at start of document (after optional horizontal whitespace).
	// Uses [ \t]* instead of \s* to avoid matching across blank lines,
	// consistent with real-world frontmatter parsers (Jekyll, Hugo).
	yamlFrontmatter = regexp.MustCompile(`(?s)^[ \t]*---\s*\n.*?\n---\s*\n`)
)

// MarkdownPreprocessor defines the contract for markdown preprocessing.
type MarkdownPreprocessor interface {
	PreprocessMarkdown(ctx context.Context, content string) string
}

// CommonMarkPreprocessor applies transformations before CommonMark conversion.
type CommonMarkPreprocessor struct{}

// PreprocessMarkdown applies all transformations to prepare Markdown for conversion.
func (p *CommonMarkPreprocessor) PreprocessMarkdown(ctx context.Context, content string) string {
	// Check for cancellation before processing
	if ctx.Err() != nil {
		return content
	}

	content = normalizeLineEndings(content)
	content = stripFrontmatter(content)
	content = convertHighlights(content)
	content = compressBlankLines(content)
	return content
}

// normalizeLineEndings converts \r\n and \r to \n.
func normalizeLineEndings(content string) string {
	return crlfOrCR.ReplaceAllString(content, "\n")
}

// stripFrontmatter removes YAML frontmatter from the start of markdown content.
// YAML frontmatter is identified by paired --- delimiters on their own lines.
// If frontmatter is malformed (missing opening or closing delimiter), it is
// left intact to avoid data loss. Only well-formed frontmatter at the document
// start is removed.
func stripFrontmatter(content string) string {
	return yamlFrontmatter.ReplaceAllLiteralString(content, "")
}

// compressBlankLines limits consecutive blank lines to 2 maximum.
func compressBlankLines(content string) string {
	return multipleBlankLines.ReplaceAllString(content, "\n\n")
}

// convertHighlights transforms ==text== to placeholder markers.
// The placeholders are converted to <mark> tags after Goldmark processing
// via convertMarkPlaceholders. This avoids needing html.WithUnsafe().
func convertHighlights(content string) string {
	return highlightPattern.ReplaceAllString(content, MarkStartPlaceholder+"$1"+MarkEndPlaceholder)
}

// ConvertMarkPlaceholders converts placeholder markers to <mark> tags.
// Called after Goldmark HTML conversion to finalize highlight markup.
// This is the second half of the ==highlight== feature, keeping Goldmark
// secure (no WithUnsafe) while still supporting inline HTML marks.
func ConvertMarkPlaceholders(content string) string {
	return strings.ReplaceAll(
		strings.ReplaceAll(content, MarkStartPlaceholder, "<mark>"),
		MarkEndPlaceholder, "</mark>",
	)
}
