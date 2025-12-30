package main

import (
	"testing"
)

func TestNormalizeLineEndings(t *testing.T) {
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
			got := NormalizeLineEndings(tt.input)
			if got != tt.expected {
				t.Errorf("NormalizeLineEndings() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestCompressBlankLines(t *testing.T) {
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
			got := CompressBlankLines(tt.input)
			if got != tt.expected {
				t.Errorf("CompressBlankLines() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestConvertHighlights(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single highlight",
			input:    "This is ==highlighted== text",
			expected: "This is <mark>highlighted</mark> text",
		},
		{
			name:     "multiple highlights",
			input:    "==one== and ==two==",
			expected: "<mark>one</mark> and <mark>two</mark>",
		},
		{
			name:     "empty highlight",
			input:    "empty ==== here",
			expected: "empty <mark></mark> here",
		},
		{
			name:     "highlight with spaces",
			input:    "==hello world==",
			expected: "<mark>hello world</mark>",
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
			expected: "This is <mark>日本語</mark> text",
		},
		{
			name:     "triple equals captures inner equals with trailing",
			input:    "===not highlight===",
			expected: "<mark>=not highlight</mark>=",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertHighlights(tt.input)
			if got != tt.expected {
				t.Errorf("ConvertHighlights() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestCommonMarkPreprocessor_PreprocessMarkdown(t *testing.T) {
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
			name:     "highlights converted",
			input:    "This is ==important== text",
			expected: "This is <mark>important</mark> text",
		},
		{
			name:     "multiple highlights converted",
			input:    "==one== and ==two==",
			expected: "<mark>one</mark> and <mark>two</mark>",
		},
		{
			name:     "multiple blank lines compressed to two",
			input:    "a\n\n\n\n\nb",
			expected: "a\n\nb",
		},
		{
			name:     "full pipeline: normalize, highlight, compress",
			input:    "Title\r\n\r\n\r\n\r\nText with ==highlight==\r\n\r\n\r\nEnd",
			expected: "Title\n\nText with <mark>highlight</mark>\n\nEnd",
		},
	}

	preprocessor := &CommonMarkPreprocessor{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := preprocessor.PreprocessMarkdown(tt.input)
			if got != tt.expected {
				t.Errorf("PreprocessMarkdown():\ngot:  %q\nwant: %q", got, tt.expected)
			}
		})
	}
}
