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

func TestEnsureBlankBeforeHeaders(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "h1 after text gets blank line",
			input:    "text\n# Header",
			expected: "text\n\n# Header",
		},
		{
			name:     "h2 after text gets blank line",
			input:    "text\n## Header",
			expected: "text\n\n## Header",
		},
		{
			name:     "h6 after text gets blank line",
			input:    "text\n###### Header",
			expected: "text\n\n###### Header",
		},
		{
			name:     "already has blank line unchanged",
			input:    "text\n\n# Header",
			expected: "text\n\n# Header",
		},
		{
			name:     "header at start unchanged",
			input:    "# Header\ntext",
			expected: "# Header\ntext",
		},
		{
			name:     "header inside fenced code block unchanged",
			input:    "```\ncode\n# Not a header\nmore code\n```",
			expected: "```\ncode\n# Not a header\nmore code\n```",
		},
		{
			name:     "header inside tilde fenced code block unchanged",
			input:    "~~~\ncode\n# Not a header\nmore code\n~~~",
			expected: "~~~\ncode\n# Not a header\nmore code\n~~~",
		},
		{
			name:     "header after fenced code block gets blank line",
			input:    "```\ncode\n```\n# Header",
			expected: "```\ncode\n```\n\n# Header",
		},
		{
			name:     "unbalanced code fence treats rest as code",
			input:    "```\ncode\n# Header inside\nmore code",
			expected: "```\ncode\n# Header inside\nmore code",
		},
		{
			name:     "header inside indented code block unchanged",
			input:    "text\n    # indented code\nmore",
			expected: "text\n    # indented code\nmore",
		},
		{
			name:     "consecutive headers get blank lines",
			input:    "# H1\n## H2\n### H3",
			expected: "# H1\n\n## H2\n\n### H3",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EnsureBlankBeforeHeaders(tt.input)
			if got != tt.expected {
				t.Errorf("EnsureBlankBeforeHeaders() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestEnsureBlankBeforeBlockquotes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "blockquote after text gets blank line",
			input:    "text\n> quote",
			expected: "text\n\n> quote",
		},
		{
			name:     "consecutive blockquotes unchanged",
			input:    "> line1\n> line2\n> line3",
			expected: "> line1\n> line2\n> line3",
		},
		{
			name:     "already has blank line unchanged",
			input:    "text\n\n> quote",
			expected: "text\n\n> quote",
		},
		{
			name:     "blockquote at start unchanged",
			input:    "> quote\ntext",
			expected: "> quote\ntext",
		},
		{
			name:     "blockquote inside fenced code block unchanged",
			input:    "```\n> not a quote\n```",
			expected: "```\n> not a quote\n```",
		},
		{
			name:     "blockquote inside tilde fenced code block unchanged",
			input:    "~~~\n> not a quote\n~~~",
			expected: "~~~\n> not a quote\n~~~",
		},
		{
			name:     "multiple separate blockquotes",
			input:    "text1\n> quote1\ntext2\n> quote2",
			expected: "text1\n\n> quote1\ntext2\n\n> quote2",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EnsureBlankBeforeBlockquotes(tt.input)
			if got != tt.expected {
				t.Errorf("EnsureBlankBeforeBlockquotes() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestEnsureBlankBeforeLists(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "unordered list with dash after text",
			input:    "text\n- item",
			expected: "text\n\n- item",
		},
		{
			name:     "unordered list with asterisk after text",
			input:    "text\n* item",
			expected: "text\n\n* item",
		},
		{
			name:     "unordered list with plus after text",
			input:    "text\n+ item",
			expected: "text\n\n+ item",
		},
		{
			name:     "ordered list after text",
			input:    "text\n1. item",
			expected: "text\n\n1. item",
		},
		{
			name:     "consecutive list items unchanged",
			input:    "- item1\n- item2\n- item3",
			expected: "- item1\n- item2\n- item3",
		},
		{
			name:     "already has blank line unchanged",
			input:    "text\n\n- item",
			expected: "text\n\n- item",
		},
		{
			name:     "list after header unchanged",
			input:    "# Header\n- item",
			expected: "# Header\n- item",
		},
		{
			name:     "list inside fenced code block unchanged",
			input:    "```\n- not a list\n```",
			expected: "```\n- not a list\n```",
		},
		{
			name:     "list inside tilde fenced code block unchanged",
			input:    "~~~\n- not a list\n~~~",
			expected: "~~~\n- not a list\n~~~",
		},
		{
			name:     "ordered list multi-digit after text",
			input:    "text\n10. tenth item",
			expected: "text\n\n10. tenth item",
		},
		{
			name:     "blockquote list after blockquote text",
			input:    "> text\n> - item",
			expected: "> text\n>\n> - item",
		},
		{
			name:     "blockquote ordered list after blockquote text",
			input:    "> text\n> 1. item",
			expected: "> text\n>\n> 1. item",
		},
		{
			name:     "consecutive blockquote list items unchanged",
			input:    "> - item1\n> - item2",
			expected: "> - item1\n> - item2",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EnsureBlankBeforeLists(tt.input)
			if got != tt.expected {
				t.Errorf("EnsureBlankBeforeLists() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestIsBlankLine(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"empty string", "", true},
		{"spaces only", "   ", true},
		{"tabs only", "\t\t", true},
		{"mixed whitespace", " \t ", true},
		{"text", "hello", false},
		{"text with spaces", "  hello  ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isBlankLine(tt.input)
			if got != tt.expected {
				t.Errorf("isBlankLine(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestIsListItem(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"dash list", "- item", true},
		{"asterisk list", "* item", true},
		{"plus list", "+ item", true},
		{"ordered list single digit", "1. item", true},
		{"ordered list multi digit", "123. item", true},
		{"ordered list 10", "10. item", true},
		{"ordered list 99", "99. item", true},
		{"not a list plain text", "text", false},
		{"not a list no space after dash", "-item", false},
		{"not a list no space after number", "1.item", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isListItem(tt.input)
			if got != tt.expected {
				t.Errorf("isListItem(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestIsBlockquoteList(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"blockquote dash list", "> - item", true},
		{"blockquote asterisk list", "> * item", true},
		{"blockquote plus list", "> + item", true},
		{"blockquote ordered list", "> 1. item", true},
		{"blockquote with extra space", ">  - item", true},
		{"plain blockquote", "> text", false},
		{"plain list", "- item", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isBlockquoteList(tt.input)
			if got != tt.expected {
				t.Errorf("isBlockquoteList(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestPreprocessMarkdown(t *testing.T) {
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
			name:     "CRLF normalized and header spaced",
			input:    "text\r\n# Header",
			expected: "text\n\n# Header",
		},
		{
			name:     "highlights converted",
			input:    "This is ==important==",
			expected: "This is <mark>important</mark>",
		},
		{
			name:     "multiple blank lines compressed",
			input:    "a\n\n\n\n\nb",
			expected: "a\n\nb",
		},
		{
			name:     "complex document",
			input:    "Intro\n# Title\nParagraph\n> Quote\nMore text\n- List item\n\n\n\n\nEnd",
			expected: "Intro\n\n# Title\nParagraph\n\n> Quote\nMore text\n\n- List item\n\nEnd",
		},
		{
			name:     "code block content preserved",
			input:    "text\n```\n# not header\n> not quote\n- not list\n```\nafter",
			expected: "text\n```\n# not header\n> not quote\n- not list\n```\nafter",
		},
		{
			name:     "tilde code block content preserved",
			input:    "text\n~~~\n# not header\n> not quote\n- not list\n~~~\nafter",
			expected: "text\n~~~\n# not header\n> not quote\n- not list\n~~~\nafter",
		},
		{
			name:     "full integration",
			input:    "Title\r\n# Header\r\nParagraph with ==highlight==\r\n> Quote\r\n> - list in quote\r\n\r\n\r\n\r\n\r\nEnd",
			expected: "Title\n\n# Header\nParagraph with <mark>highlight</mark>\n\n> Quote\n>\n> - list in quote\n\nEnd",
		},
	}

	preprocessor := &CommonMarkToPandocPreprocessor{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := preprocessor.PreprocessMarkdown(tt.input)
			if got != tt.expected {
				t.Errorf("PreprocessMarkdown():\ngot:\n%q\nwant:\n%q", got, tt.expected)
			}
		})
	}
}
