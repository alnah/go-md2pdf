package md2pdf

import (
	"context"
	"testing"
)

func TestNormalizeLineEndings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "LF unchanged",
			input:    "line1\nline2\nline3",
			expected: "line1\nline2\nline3",
		},
		{
			name:     "CRLF to LF",
			input:    "line1\r\nline2\r\nline3",
			expected: "line1\nline2\nline3",
		},
		{
			name:     "CR to LF",
			input:    "line1\rline2\rline3",
			expected: "line1\nline2\nline3",
		},
		{
			name:     "mixed line endings",
			input:    "line1\r\nline2\rline3\nline4",
			expected: "line1\nline2\nline3\nline4",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := normalizeLineEndings(tt.input)
			if got != tt.expected {
				t.Errorf("normalizeLineEndings() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestCompressBlankLines(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single blank line unchanged",
			input:    "line1\n\nline2",
			expected: "line1\n\nline2",
		},
		{
			name:     "two blank lines compressed to two newlines",
			input:    "line1\n\n\nline2",
			expected: "line1\n\nline2",
		},
		{
			name:     "three blank lines compressed to two",
			input:    "line1\n\n\n\nline2",
			expected: "line1\n\nline2",
		},
		{
			name:     "five blank lines compressed to two",
			input:    "line1\n\n\n\n\n\nline2",
			expected: "line1\n\nline2",
		},
		{
			name:     "multiple groups compressed",
			input:    "a\n\n\n\nb\n\n\n\n\nc",
			expected: "a\n\nb\n\nc",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := compressBlankLines(tt.input)
			if got != tt.expected {
				t.Errorf("compressBlankLines() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestConvertHighlights(t *testing.T) {
	t.Parallel()

	// Helper to build expected output with placeholders
	mark := func(s string) string {
		return MarkStartPlaceholder + s + MarkEndPlaceholder
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single highlight",
			input:    "This is ==highlighted== text",
			expected: "This is " + mark("highlighted") + " text",
		},
		{
			name:     "multiple highlights",
			input:    "==one== and ==two==",
			expected: mark("one") + " and " + mark("two"),
		},
		{
			name:     "empty highlight",
			input:    "empty ==== here",
			expected: "empty " + mark("") + " here",
		},
		{
			name:     "highlight with spaces",
			input:    "==hello world==",
			expected: mark("hello world"),
		},
		{
			name:     "no highlights",
			input:    "plain text",
			expected: "plain text",
		},
		{
			name:     "unclosed highlight unchanged",
			input:    "==unclosed",
			expected: "==unclosed",
		},
		{
			name:     "unicode highlight",
			input:    "This is ==日本語== text",
			expected: "This is " + mark("日本語") + " text",
		},
		{
			name:     "triple equals captures inner equals with trailing",
			input:    "===not highlight===",
			expected: mark("=not highlight") + "=",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := convertHighlights(tt.input)
			if got != tt.expected {
				t.Errorf("convertHighlights() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestConvertMarkPlaceholders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single placeholder",
			input:    "text " + MarkStartPlaceholder + "highlighted" + MarkEndPlaceholder + " more",
			expected: "text <mark>highlighted</mark> more",
		},
		{
			name:     "multiple placeholders",
			input:    MarkStartPlaceholder + "one" + MarkEndPlaceholder + " and " + MarkStartPlaceholder + "two" + MarkEndPlaceholder,
			expected: "<mark>one</mark> and <mark>two</mark>",
		},
		{
			name:     "no placeholders",
			input:    "plain text without markers",
			expected: "plain text without markers",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "nested in HTML",
			input:    "<p>" + MarkStartPlaceholder + "important" + MarkEndPlaceholder + "</p>",
			expected: "<p><mark>important</mark></p>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := convertMarkPlaceholders(tt.input)
			if got != tt.expected {
				t.Errorf("convertMarkPlaceholders() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestCommonMarkPreprocessor_PreprocessMarkdown(t *testing.T) {
	t.Parallel()

	// Helper to build expected output with placeholders
	mark := func(s string) string {
		return MarkStartPlaceholder + s + MarkEndPlaceholder
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "plain text unchanged",
			input:    "Hello world",
			expected: "Hello world",
		},
		{
			name:     "CRLF normalized to LF",
			input:    "line1\r\nline2\r\nline3",
			expected: "line1\nline2\nline3",
		},
		{
			name:     "CR normalized to LF",
			input:    "line1\rline2\rline3",
			expected: "line1\nline2\nline3",
		},
		{
			name:     "highlights converted to placeholders",
			input:    "This is ==important== text",
			expected: "This is " + mark("important") + " text",
		},
		{
			name:     "multiple highlights converted to placeholders",
			input:    "==one== and ==two==",
			expected: mark("one") + " and " + mark("two"),
		},
		{
			name:     "multiple blank lines compressed to two",
			input:    "a\n\n\n\n\nb",
			expected: "a\n\nb",
		},
		{
			name:     "full pipeline: normalize, highlight, compress",
			input:    "Title\r\n\r\n\r\n\r\nText with ==highlight==\r\n\r\n\r\nEnd",
			expected: "Title\n\nText with " + mark("highlight") + "\n\nEnd",
		},
	}

	preprocessor := &commonMarkPreprocessor{}
	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := preprocessor.PreprocessMarkdown(ctx, tt.input)
			if got != tt.expected {
				t.Errorf("PreprocessMarkdown():\ngot:  %q\nwant: %q", got, tt.expected)
			}
		})
	}
}
