package main

import (
	"os"
	"strings"
	"testing"
)

func TestSanitizeCSS(t *testing.T) {
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
			name:     "no escape needed",
			input:    "body { color: red; }",
			expected: "body { color: red; }",
		},
		{
			name:     "escapes style close",
			input:    "</style>",
			expected: `<\/style>`,
		},
		{
			name:     "escapes script close",
			input:    "</script>",
			expected: `<\/script>`,
		},
		{
			name:     "multiple occurrences",
			input:    "</a></b>",
			expected: `<\/a><\/b>`,
		},
		{
			name:     "nested sequences",
			input:    "</</style>",
			expected: `<\/<\/style>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeCSS(tt.input)
			if got != tt.expected {
				t.Errorf("SanitizeCSS(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestInjectCSS(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		css      string
		expected string
	}{
		{
			name:     "empty CSS returns HTML unchanged",
			html:     "<html><head></head><body>Hello</body></html>",
			css:      "",
			expected: "<html><head></head><body>Hello</body></html>",
		},
		{
			name:     "injects before </head>",
			html:     "<html><head></head><body>Hello</body></html>",
			css:      "body { color: red; }",
			expected: "<html><head><style>body { color: red; }</style></head><body>Hello</body></html>",
		},
		{
			name:     "injects before </HEAD> mixed case",
			html:     "<html><HEAD></HEAD><body>Hello</body></html>",
			css:      "body { color: red; }",
			expected: "<html><HEAD><style>body { color: red; }</style></HEAD><body>Hello</body></html>",
		},
		{
			name:     "injects after <body> when no </head>",
			html:     "<html><body>Hello</body></html>",
			css:      "body { color: red; }",
			expected: "<html><body><style>body { color: red; }</style>Hello</body></html>",
		},
		{
			name:     "injects after <body> with attributes",
			html:     `<html><body class="main" id="app">Hello</body></html>`,
			css:      "body { color: red; }",
			expected: `<html><body class="main" id="app"><style>body { color: red; }</style>Hello</body></html>`,
		},
		{
			name:     "injects after <BODY> mixed case",
			html:     "<html><BODY>Hello</BODY></html>",
			css:      "body { color: red; }",
			expected: "<html><BODY><style>body { color: red; }</style>Hello</BODY></html>",
		},
		{
			name:     "prepends to bare fragment",
			html:     "<p>Hello</p>",
			css:      "p { color: blue; }",
			expected: "<style>p { color: blue; }</style><p>Hello</p>",
		},
		{
			name:     "sanitizes CSS with closing tags",
			html:     "<html><head></head><body>Hello</body></html>",
			css:      "</style><script>alert('xss')</script>",
			expected: `<html><head><style><\/style><script>alert('xss')<\/script></style></head><body>Hello</body></html>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := InjectCSS(tt.html, tt.css)
			if got != tt.expected {
				t.Errorf("InjectCSS() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestValidateToPDFInputs(t *testing.T) {
	tests := []struct {
		name        string
		htmlContent string
		outputPath  string
		wantErr     error
	}{
		{
			name:        "valid inputs",
			htmlContent: "<html></html>",
			outputPath:  "/tmp/out.pdf",
			wantErr:     nil,
		},
		{
			name:        "empty HTML",
			htmlContent: "",
			outputPath:  "/tmp/out.pdf",
			wantErr:     ErrEmptyHTML,
		},
		{
			name:        "empty output path",
			htmlContent: "<html></html>",
			outputPath:  "",
			wantErr:     ErrEmptyOutputPath,
		},
		{
			name:        "both empty returns ErrEmptyHTML first",
			htmlContent: "",
			outputPath:  "",
			wantErr:     ErrEmptyHTML,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateToPDFInputs(tt.htmlContent, tt.outputPath)
			if err != tt.wantErr {
				t.Errorf("validateToPDFInputs() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestWriteHTMLToTempFile(t *testing.T) {
	content := "<html><body>Test Content</body></html>"

	path, cleanup, err := writeHTMLToTempFile(content)
	if err != nil {
		t.Fatalf("writeHTMLToTempFile() error = %v", err)
	}

	t.Run("file exists", func(t *testing.T) {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("temp file does not exist at %s", path)
		}
	})

	t.Run("file has .html extension pattern", func(t *testing.T) {
		if !strings.Contains(path, "go-md2pdf-") || !strings.HasSuffix(path, ".html") {
			t.Errorf("path %q does not match expected pattern go-md2pdf-*.html", path)
		}
	})

	t.Run("file contains expected content", func(t *testing.T) {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read temp file: %v", err)
		}
		if string(data) != content {
			t.Errorf("file content = %q, want %q", string(data), content)
		}
	})

	t.Run("cleanup removes file", func(t *testing.T) {
		cleanup()
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("temp file still exists after cleanup at %s", path)
		}
	})
}
