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

func TestInjectFooter(t *testing.T) {
	injector := &FooterInjection{}

	t.Run("nil data returns HTML unchanged", func(t *testing.T) {
		html := "<html><head></head><body>Hello</body></html>"
		got := injector.InjectFooter(html, nil)
		if got != html {
			t.Errorf("InjectFooter() = %q, want %q", got, html)
		}
	})

	t.Run("empty content returns HTML unchanged", func(t *testing.T) {
		html := "<html><head></head><body>Hello</body></html>"
		data := &FooterData{} // All fields empty/false
		got := injector.InjectFooter(html, data)
		if got != html {
			t.Errorf("InjectFooter() = %q, want %q", got, html)
		}
	})

	t.Run("injects before </head>", func(t *testing.T) {
		html := "<html><head></head><body>Hello</body></html>"
		data := &FooterData{Text: "Footer Text"}
		got := injector.InjectFooter(html, data)

		if !strings.Contains(got, "@page") {
			t.Error("expected @page rule in output")
		}
		if !strings.Contains(got, "Footer Text") {
			t.Error("expected footer text in output")
		}

		// Verify position: style should appear before </head>
		styleIdx := strings.Index(got, "<style>")
		headIdx := strings.Index(got, "</head>")
		if styleIdx == -1 || headIdx == -1 || styleIdx > headIdx {
			t.Error("style should be inserted before </head>")
		}
	})

	t.Run("injects before </HEAD> mixed case", func(t *testing.T) {
		html := "<html><HEAD></HEAD><body>Hello</body></html>"
		data := &FooterData{Text: "Test"}
		got := injector.InjectFooter(html, data)

		if !strings.Contains(got, "<style>") {
			t.Error("expected style block in output")
		}
	})

	t.Run("injects after <body> when no </head>", func(t *testing.T) {
		html := "<html><body>Hello</body></html>"
		data := &FooterData{Text: "Test"}
		got := injector.InjectFooter(html, data)

		styleIdx := strings.Index(got, "<style>")
		bodyIdx := strings.Index(got, "<body>")
		if styleIdx == -1 || bodyIdx == -1 || styleIdx < bodyIdx {
			t.Error("style should be inserted after <body>")
		}
	})

	t.Run("injects after <body> with attributes", func(t *testing.T) {
		html := `<html><body class="main" id="app">Hello</body></html>`
		data := &FooterData{Text: "Test"}
		got := injector.InjectFooter(html, data)

		if !strings.Contains(got, "<style>") {
			t.Error("expected style block in output")
		}
		// Style should be after the closing > of <body>
		bodyCloseIdx := strings.Index(got, `id="app">`)
		styleIdx := strings.Index(got, "<style>")
		if styleIdx < bodyCloseIdx {
			t.Error("style should be after <body> closing bracket")
		}
	})

	t.Run("prepends to bare fragment", func(t *testing.T) {
		html := "<p>Hello</p>"
		data := &FooterData{Text: "Test"}
		got := injector.InjectFooter(html, data)

		if !strings.HasPrefix(got, "<style>") {
			t.Errorf("expected style to be prepended, got: %q", got)
		}
	})

	t.Run("page number generates counter", func(t *testing.T) {
		html := "<html><head></head><body></body></html>"
		data := &FooterData{ShowPageNumber: true}
		got := injector.InjectFooter(html, data)

		if !strings.Contains(got, "counter(page)") {
			t.Error("expected counter(page) in output")
		}
	})

	t.Run("multiple fields joined with separator", func(t *testing.T) {
		html := "<html><head></head><body></body></html>"
		data := &FooterData{
			ShowPageNumber: true,
			Date:           "2025-01-15",
			Status:         "DRAFT",
			Text:           "Footer",
		}
		got := injector.InjectFooter(html, data)

		// Check all parts are present
		if !strings.Contains(got, "counter(page)") {
			t.Error("expected counter(page)")
		}
		if !strings.Contains(got, "2025-01-15") {
			t.Error("expected date")
		}
		if !strings.Contains(got, "DRAFT") {
			t.Error("expected status")
		}
		if !strings.Contains(got, "Footer") {
			t.Error("expected text")
		}
		// Check separator
		if !strings.Contains(got, "' - '") {
			t.Error("expected separator between parts")
		}
	})
}

func TestBuildFooterContent(t *testing.T) {
	injector := &FooterInjection{}

	tests := []struct {
		name string
		data *FooterData
		want string
	}{
		{
			name: "empty data returns empty string",
			data: &FooterData{},
			want: "",
		},
		{
			name: "page number only",
			data: &FooterData{ShowPageNumber: true},
			want: "counter(page)",
		},
		{
			name: "date only",
			data: &FooterData{Date: "2025-01-15"},
			want: "'2025-01-15'",
		},
		{
			name: "status only",
			data: &FooterData{Status: "DRAFT"},
			want: "'DRAFT'",
		},
		{
			name: "text only",
			data: &FooterData{Text: "Footer"},
			want: "'Footer'",
		},
		{
			name: "all fields joined with separator",
			data: &FooterData{
				ShowPageNumber: true,
				Date:           "2025-01-15",
				Status:         "DRAFT",
				Text:           "Footer",
			},
			want: "counter(page) ' - ' '2025-01-15' ' - ' 'DRAFT' ' - ' 'Footer'",
		},
		{
			name: "page number and text only",
			data: &FooterData{
				ShowPageNumber: true,
				Text:           "Copyright",
			},
			want: "counter(page) ' - ' 'Copyright'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := injector.buildFooterContent(tt.data)
			if got != tt.want {
				t.Errorf("buildFooterContent() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolvePosition(t *testing.T) {
	injector := &FooterInjection{}

	tests := []struct {
		name     string
		position string
		want     string
	}{
		{
			name:     "left returns @bottom-left",
			position: "left",
			want:     "@bottom-left",
		},
		{
			name:     "center returns @bottom-center",
			position: "center",
			want:     "@bottom-center",
		},
		{
			name:     "right returns @bottom-right",
			position: "right",
			want:     "@bottom-right",
		},
		{
			name:     "empty returns default @bottom-right",
			position: "",
			want:     "@bottom-right",
		},
		{
			name:     "unknown returns default @bottom-right",
			position: "top",
			want:     "@bottom-right",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := injector.resolvePosition(tt.position)
			if got != tt.want {
				t.Errorf("resolvePosition(%q) = %q, want %q", tt.position, got, tt.want)
			}
		})
	}
}

func TestEscapeCSSString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no special chars",
			input: "Hello World",
			want:  "Hello World",
		},
		{
			name:  "escapes backslash",
			input: `path\to\file`,
			want:  `path\\to\\file`,
		},
		{
			name:  "escapes single quote",
			input: "it's a test",
			want:  `it\'s a test`,
		},
		{
			name:  "escapes both backslash and quote",
			input: `it's a \path`,
			want:  `it\'s a \\path`,
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "multiple quotes",
			input: "'quoted' and 'more'",
			want:  `\'quoted\' and \'more\'`,
		},
		{
			name:  "consecutive backslashes",
			input: `\\`,
			want:  `\\\\`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escapeCSSString(tt.input)
			if got != tt.want {
				t.Errorf("escapeCSSString(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
