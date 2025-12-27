package main

import (
	"regexp"
	"strings"
)

// Precompiled regex patterns for performance.
var (
	// Line ending normalization
	crlfOrCR = regexp.MustCompile(`\r\n?`)

	// Compress multiple blank lines to max 2
	multipleBlankLines = regexp.MustCompile(`\n{3,}`)

	// Highlight syntax ==text==
	highlightPattern = regexp.MustCompile(`==(.*?)==`)

	// Fenced code block delimiter (backticks or tildes)
	fencedCodeBlock = regexp.MustCompile("^(```|~~~)")

	// Header pattern (ATX style)
	headerPattern = regexp.MustCompile(`^#{1,6}\s`)

	// Blockquote pattern
	blockquotePattern = regexp.MustCompile(`^>`)

	// List item patterns (unordered and ordered)
	unorderedListPattern = regexp.MustCompile(`^[-*+]\s`)
	orderedListPattern   = regexp.MustCompile(`^[0-9]+\.\s`)

	// List in blockquote patterns
	blockquoteUnorderedList = regexp.MustCompile(`^>\s*[-*+]\s`)
	blockquoteOrderedList   = regexp.MustCompile(`^>\s*[0-9]+\.\s`)

	// Indented code block (4 spaces or tab)
	indentedCodeBlock = regexp.MustCompile(`^(    |\t)`)
)

// MarkdownPreprocessor defines the contract for markdown preprocessing.
type MarkdownPreprocessor interface {
	PreprocessMarkdown(content string) string
}

// CommonMarkToPandocPreprocessor transforms CommonMark content for Pandoc compatibility.
type CommonMarkToPandocPreprocessor struct{}

// PreprocessMarkdown applies all transformations to prepare Markdown for Pandoc.
// Order matters: normalize line endings first, then spacing fixes, then syntax conversions.
// Makes multiple passes for modularity and clarity; acceptable for typical document sizes.
func (p *CommonMarkToPandocPreprocessor) PreprocessMarkdown(content string) string {
	content = NormalizeLineEndings(content)
	content = EnsureBlankBeforeHeaders(content)
	content = EnsureBlankBeforeBlockquotes(content)
	content = EnsureBlankBeforeLists(content)
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

// EnsureBlankBeforeHeaders adds a blank line before ATX headers (#, ##, etc.)
// if the previous line is non-empty. Skips content inside code blocks.
func EnsureBlankBeforeHeaders(content string) string {
	return processLinesWithCodeBlockAwareness(content, func(prev, current string) string {
		if headerPattern.MatchString(current) && prev != "" && !isBlankLine(prev) {
			return "\n" + current
		}
		return current
	})
}

// EnsureBlankBeforeBlockquotes adds a blank line before blockquotes (>)
// if the previous line is non-empty and not itself a blockquote.
// Skips content inside code blocks.
func EnsureBlankBeforeBlockquotes(content string) string {
	return processLinesWithCodeBlockAwareness(content, func(prev, current string) string {
		if blockquotePattern.MatchString(current) &&
			prev != "" &&
			!isBlankLine(prev) &&
			!blockquotePattern.MatchString(prev) {
			return "\n" + current
		}
		return current
	})
}

// EnsureBlankBeforeLists adds a blank line before list items (-, *, +, 1.)
// if the previous line is text (not a list item, blank, or header).
// Also handles lists inside blockquotes: "> text\n> - item" becomes "> text\n>\n> - item".
// Skips content inside code blocks.
func EnsureBlankBeforeLists(content string) string {
	return processLinesWithCodeBlockAwareness(content, func(prev, current string) string {
		// Handle lists inside blockquotes
		if isBlockquoteList(current) && blockquotePattern.MatchString(prev) && !isBlockquoteList(prev) && !isBlankLine(prev) {
			// Previous is blockquote text, current is blockquote list
			// Insert ">\" between them
			return ">\n" + current
		}

		// Handle regular lists
		if isListItem(current) && prev != "" && !isBlankLine(prev) && !isListItem(prev) && !headerPattern.MatchString(prev) {
			return "\n" + current
		}

		return current
	})
}

// processLinesWithCodeBlockAwareness processes each line with a callback,
// but skips lines inside fenced code blocks.
func processLinesWithCodeBlockAwareness(content string, process func(prev, current string) string) string {
	lines := strings.Split(content, "\n")
	result := make([]string, 0, len(lines))

	inCodeBlock := false
	var previousLine string

	for i, line := range lines {
		// Track fenced code blocks
		if fencedCodeBlock.MatchString(line) {
			inCodeBlock = !inCodeBlock
		}

		// Skip processing inside code blocks or indented code blocks
		if inCodeBlock || indentedCodeBlock.MatchString(line) {
			result = append(result, line)
			previousLine = line
			continue
		}

		// First line has no previous
		if i == 0 {
			result = append(result, line)
			previousLine = line
			continue
		}

		processed := process(previousLine, line)
		if strings.HasPrefix(processed, "\n") {
			// Insert blank line before current line
			result = append(result, "")
			result = append(result, processed[1:])
		} else {
			result = append(result, processed)
		}

		// Use original line (not processed) to detect structure in next iteration.
		// This ensures we match against the original Markdown syntax, not inserted blank lines.
		previousLine = line
	}

	return strings.Join(result, "\n")
}

// isBlankLine returns true if the line is empty or contains only whitespace.
func isBlankLine(line string) bool {
	return strings.TrimSpace(line) == ""
}

// isListItem returns true if the line starts with a list marker (-, *, +, or 1.).
func isListItem(line string) bool {
	return unorderedListPattern.MatchString(line) || orderedListPattern.MatchString(line)
}

// isBlockquoteList returns true if the line is a list item inside a blockquote.
func isBlockquoteList(line string) bool {
	return blockquoteUnorderedList.MatchString(line) || blockquoteOrderedList.MatchString(line)
}
