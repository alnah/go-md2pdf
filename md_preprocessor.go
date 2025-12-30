package main

import (
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

// MarkdownPreprocessor defines the contract for markdown preprocessing.
type MarkdownPreprocessor interface {
	PreprocessMarkdown(content string) string
}

// CommonMarkPreprocessor applies transformations before CommonMark conversion.
type CommonMarkPreprocessor struct{}

// PreprocessMarkdown applies all transformations to prepare Markdown for conversion.
func (p *CommonMarkPreprocessor) PreprocessMarkdown(content string) string {
	content = NormalizeLineEndings(content)
	content = ConvertHighlights(content)
	content = CompressBlankLines(content)
	return content
}

// NormalizeLineEndings converts \r\n and \r to \n.
func NormalizeLineEndings(content string) string {
	return crlfOrCR.ReplaceAllString(content, "\n")
}

// CompressBlankLines limits consecutive blank lines to 2 maximum.
func CompressBlankLines(content string) string {
	return multipleBlankLines.ReplaceAllString(content, "\n\n")
}

// ConvertHighlights transforms ==text== to <mark>text</mark>.
func ConvertHighlights(content string) string {
	return highlightPattern.ReplaceAllString(content, "<mark>$1</mark>")
}
