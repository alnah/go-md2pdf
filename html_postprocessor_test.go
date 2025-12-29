package main

import (
	"errors"
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
		{
			name:     "case variation STYLE",
			input:    "</STYLE>",
			expected: `<\/STYLE>`,
		},
		{
			name:     "case variation Script",
			input:    "</Script>",
			expected: `<\/Script>`,
		},
		{
			name:     "mixed case sTyLe",
			input:    "</sTyLe>",
			expected: `<\/sTyLe>`,
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
		{
			name:     "unicode in CSS content property",
			html:     "<html><head></head><body>Hello</body></html>",
			css:      `.icon::before { content: ""; }`,
			expected: `<html><head><style>.icon::before { content: ""; }</style></head><body>Hello</body></html>`,
		},
		{
			name:     "unicode in HTML preserved",
			html:     "<html><head></head><body>Bonjour le monde</body></html>",
			css:      "body { color: red; }",
			expected: "<html><head><style>body { color: red; }</style></head><body>Bonjour le monde</body></html>",
		},
	}

	injector := &CSSInjection{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := injector.InjectCSS(tt.html, tt.css)
			if got != tt.expected {
				t.Errorf("InjectCSS() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestInjectSignature(t *testing.T) {
	injector := NewSignatureInjection()

	t.Run("nil data returns HTML unchanged", func(t *testing.T) {
		html := "<html><body>Hello</body></html>"
		got, err := injector.InjectSignature(html, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != html {
			t.Errorf("InjectSignature() = %q, want %q", got, html)
		}
	})

	t.Run("injects signature before </body>", func(t *testing.T) {
		html := "<html><body>Content</body></html>"
		data := &SignatureData{Name: "John Doe", Email: "john@example.com"}

		got, err := injector.InjectSignature(html, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify signature is injected before </body>
		if !strings.Contains(got, "John Doe") {
			t.Error("signature name not found in output")
		}
		if !strings.Contains(got, "john@example.com") {
			t.Error("signature email not found in output")
		}

		// Verify position: signature should appear before </body>
		sigIdx := strings.Index(got, "signature-block")
		bodyIdx := strings.Index(got, "</body>")
		if sigIdx == -1 || bodyIdx == -1 || sigIdx > bodyIdx {
			t.Error("signature should be inserted before </body>")
		}
	})

	t.Run("injects before </BODY> mixed case", func(t *testing.T) {
		html := "<html><BODY>Content</BODY></html>"
		data := &SignatureData{Name: "Test"}

		got, err := injector.InjectSignature(html, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(got, "Test") {
			t.Error("signature name not found in output")
		}
	})

	t.Run("appends to end when no </body>", func(t *testing.T) {
		html := "<p>Content</p>"
		data := &SignatureData{Name: "Test"}

		got, err := injector.InjectSignature(html, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Signature should be at the end
		if !strings.HasSuffix(got, "</div>\n") {
			t.Errorf("signature should be appended at end, got: %q", got)
		}
	})

	t.Run("renders all signature fields", func(t *testing.T) {
		html := "<html><body></body></html>"
		data := &SignatureData{
			Name:      "Jane Smith",
			Title:     "Software Engineer",
			Email:     "jane@example.com",
			ImagePath: "https://example.com/photo.jpg",
			Links: []SignatureLink{
				{Label: "GitHub", URL: "https://github.com/jane"},
				{Label: "LinkedIn", URL: "https://linkedin.com/in/jane"},
			},
		}

		got, err := injector.InjectSignature(html, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify all fields are rendered
		expectedParts := []string{
			"Jane Smith",
			"Software Engineer",
			"jane@example.com",
			"https://example.com/photo.jpg",
			"GitHub",
			"https://github.com/jane",
			"LinkedIn",
			"https://linkedin.com/in/jane",
		}
		for _, part := range expectedParts {
			if !strings.Contains(got, part) {
				t.Errorf("expected %q in output", part)
			}
		}
	})

	t.Run("optional fields can be empty", func(t *testing.T) {
		html := "<html><body></body></html>"
		data := &SignatureData{
			Name: "Minimal",
			// Title, Email, ImagePath, Links all empty
		}

		got, err := injector.InjectSignature(html, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(got, "Minimal") {
			t.Error("name should be rendered")
		}
		// Should not contain empty <em> or <a> tags for missing fields
		if strings.Contains(got, "<em></em>") {
			t.Error("empty title should not render empty <em> tag")
		}
	})
}

func TestInjectSignature_TemplateError(t *testing.T) {
	// Create injector with a broken template to test error path
	// This is difficult to trigger with valid SignatureData,
	// but we can verify the error type is returned correctly
	// by using the mock in service_test.go

	// For the real implementation, verify that error wrapping works
	injector := NewSignatureInjection()
	data := &SignatureData{Name: "Test"}

	_, err := injector.InjectSignature("<body></body>", data)
	if err != nil {
		if !errors.Is(err, ErrSignatureRender) {
			t.Errorf("error should wrap ErrSignatureRender, got: %v", err)
		}
	}
	// Note: with valid data and template, no error expected
}
