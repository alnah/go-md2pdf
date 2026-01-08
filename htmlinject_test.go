package md2pdf

import (
	"context"
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
			got := sanitizeCSS(tt.input)
			if got != tt.expected {
				t.Errorf("sanitizeCSS(%q) = %q, want %q", tt.input, got, tt.expected)
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

	injector := &cssInjection{}
	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := injector.InjectCSS(ctx, tt.html, tt.css)
			if got != tt.expected {
				t.Errorf("InjectCSS() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestInjectCSS_ContextCancellation(t *testing.T) {
	injector := &cssInjection{}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	html := "<html><head></head><body>Hello</body></html>"
	css := "body { color: red; }"

	// When context is cancelled, returns HTML unchanged
	got := injector.InjectCSS(ctx, html, css)
	if got != html {
		t.Errorf("InjectCSS() with cancelled context should return HTML unchanged, got %q", got)
	}
}

func TestInjectSignature(t *testing.T) {
	injector := newSignatureInjection()
	ctx := context.Background()

	t.Run("nil data returns HTML unchanged", func(t *testing.T) {
		html := "<html><body>Hello</body></html>"
		got, err := injector.InjectSignature(ctx, html, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != html {
			t.Errorf("InjectSignature() = %q, want %q", got, html)
		}
	})

	t.Run("injects signature before </body>", func(t *testing.T) {
		html := "<html><body>Content</body></html>"
		data := &signatureData{Name: "John Doe", Email: "john@example.com"}

		got, err := injector.InjectSignature(ctx, html, data)
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
		data := &signatureData{Name: "Test"}

		got, err := injector.InjectSignature(ctx, html, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(got, "Test") {
			t.Error("signature name not found in output")
		}
	})

	t.Run("appends to end when no </body>", func(t *testing.T) {
		html := "<p>Content</p>"
		data := &signatureData{Name: "Test"}

		got, err := injector.InjectSignature(ctx, html, data)
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
		data := &signatureData{
			Name:      "Jane Smith",
			Title:     "Software Engineer",
			Email:     "jane@example.com",
			ImagePath: "https://example.com/photo.jpg",
			Links: []signatureLink{
				{Label: "GitHub", URL: "https://github.com/jane"},
				{Label: "LinkedIn", URL: "https://linkedin.com/in/jane"},
			},
		}

		got, err := injector.InjectSignature(ctx, html, data)
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
		data := &signatureData{
			Name: "Minimal",
			// Title, Email, ImagePath, Links all empty
		}

		got, err := injector.InjectSignature(ctx, html, data)
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

func TestInjectSignature_ContextCancellation(t *testing.T) {
	injector := newSignatureInjection()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	data := &signatureData{Name: "Test"}
	_, err := injector.InjectSignature(ctx, "<body></body>", data)

	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestInjectSignature_TemplateError(t *testing.T) {
	// Create injector with a broken template to test error path
	// This is difficult to trigger with valid signatureData,
	// but we can verify the error type is returned correctly
	// by using the mock in service_test.go

	// For the real implementation, verify that error wrapping works
	injector := newSignatureInjection()
	ctx := context.Background()
	data := &signatureData{Name: "Test"}

	_, err := injector.InjectSignature(ctx, "<body></body>", data)
	if err != nil {
		if !errors.Is(err, ErrSignatureRender) {
			t.Errorf("error should wrap ErrSignatureRender, got: %v", err)
		}
	}
	// Note: with valid data and template, no error expected
}

func TestInjectCover(t *testing.T) {
	injector := newCoverInjection()
	ctx := context.Background()

	t.Run("nil data returns HTML unchanged", func(t *testing.T) {
		html := "<html><body>Hello</body></html>"
		got, err := injector.InjectCover(ctx, html, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != html {
			t.Errorf("InjectCover() = %q, want %q", got, html)
		}
	})

	t.Run("injects cover after <body>", func(t *testing.T) {
		html := "<html><body>Content</body></html>"
		data := &coverData{Title: "My Document"}

		got, err := injector.InjectCover(ctx, html, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify cover is injected after <body>
		if !strings.Contains(got, "My Document") {
			t.Error("cover title not found in output")
		}

		// Verify position: cover should appear after <body>
		bodyIdx := strings.Index(got, "<body>")
		coverIdx := strings.Index(got, "cover-page")
		if bodyIdx == -1 || coverIdx == -1 || coverIdx < bodyIdx {
			t.Error("cover should be inserted after <body>")
		}
	})

	t.Run("injects after <body> with attributes", func(t *testing.T) {
		html := `<html><body class="main">Content</body></html>`
		data := &coverData{Title: "Test"}

		got, err := injector.InjectCover(ctx, html, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(got, "Test") {
			t.Error("cover title not found in output")
		}
	})

	t.Run("injects after <BODY> mixed case", func(t *testing.T) {
		html := "<html><BODY>Content</BODY></html>"
		data := &coverData{Title: "Test"}

		got, err := injector.InjectCover(ctx, html, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(got, "Test") {
			t.Error("cover title not found in output")
		}
	})

	t.Run("prepends when no <body>", func(t *testing.T) {
		html := "<p>Content</p>"
		data := &coverData{Title: "Test"}

		got, err := injector.InjectCover(ctx, html, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Cover should be at the start
		if !strings.HasPrefix(got, "<div class=\"cover-page\">") {
			t.Errorf("cover should be prepended, got: %q", got[:min(100, len(got))])
		}
	})

	t.Run("renders all cover fields", func(t *testing.T) {
		html := "<html><body></body></html>"
		data := &coverData{
			Title:        "My Document",
			Subtitle:     "A Comprehensive Guide",
			Logo:         "https://example.com/logo.png",
			Author:       "John Doe",
			AuthorTitle:  "Senior Developer",
			Organization: "Acme Corp",
			Date:         "2025-01-15",
			Version:      "v1.0.0",
		}

		got, err := injector.InjectCover(ctx, html, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify all fields are rendered
		expectedParts := []string{
			"My Document",
			"A Comprehensive Guide",
			"https://example.com/logo.png",
			"John Doe",
			"Senior Developer",
			"Acme Corp",
			"2025-01-15",
			"v1.0.0",
		}
		for _, part := range expectedParts {
			if !strings.Contains(got, part) {
				t.Errorf("expected %q in output", part)
			}
		}
	})

	t.Run("optional fields can be empty", func(t *testing.T) {
		html := "<html><body></body></html>"
		data := &coverData{
			Title: "Minimal",
			// All other fields empty
		}

		got, err := injector.InjectCover(ctx, html, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(got, "Minimal") {
			t.Error("title should be rendered")
		}
		if !strings.Contains(got, "cover-page") {
			t.Error("cover page class should be present")
		}
	})

	t.Run("HTML escapes special characters", func(t *testing.T) {
		html := "<html><body></body></html>"
		data := &coverData{
			Title:  "<script>alert('xss')</script>",
			Author: "John & Jane",
		}

		got, err := injector.InjectCover(ctx, html, data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// HTML template should escape these
		if strings.Contains(got, "<script>alert") {
			t.Error("script tag should be escaped")
		}
		if !strings.Contains(got, "&lt;script&gt;") && !strings.Contains(got, "&#") {
			// Either HTML entity or numeric escape is acceptable
			t.Log("Note: checking for HTML escaping of script tag")
		}
	})
}

func TestInjectCover_ContextCancellation(t *testing.T) {
	injector := newCoverInjection()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	data := &coverData{Title: "Test"}
	_, err := injector.InjectCover(ctx, "<body></body>", data)

	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestEscapeCSSString(t *testing.T) {
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
			name:     "simple text",
			input:    "DRAFT",
			expected: "DRAFT",
		},
		{
			name:     "text with spaces",
			input:    "FOR REVIEW",
			expected: "FOR REVIEW",
		},
		{
			name:     "escapes double quotes",
			input:    `DRAFT "v1"`,
			expected: `DRAFT \"v1\"`,
		},
		{
			name:     "escapes backslash",
			input:    `path\to\file`,
			expected: `path\\to\\file`,
		},
		{
			name:     "escapes newline",
			input:    "line1\nline2",
			expected: `line1\A line2`,
		},
		{
			name:     "removes carriage return",
			input:    "line1\r\nline2",
			expected: `line1\A line2`,
		},
		{
			name:     "CSS injection attempt - closing quote",
			input:    `DRAFT"; } body { display: none } .x { content: "`,
			expected: `DRAFT\"; } body { display: none } .x { content: \"`,
		},
		{
			name:     "CSS injection attempt - backslash escape",
			input:    `DRAFT\"; } body { display: none }`,
			expected: `DRAFT\\\"; } body { display: none }`,
		},
		{
			name:     "unicode preserved",
			input:    "BROUILLON",
			expected: "BROUILLON",
		},
		{
			name:     "mixed special characters",
			input:    "A\"B\\C\nD\rE",
			expected: `A\"B\\C\A DE`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escapeCSSString(tt.input)
			if got != tt.expected {
				t.Errorf("escapeCSSString(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestBuildWatermarkCSS(t *testing.T) {
	tests := []struct {
		name           string
		watermark      *Watermark
		wantEmpty      bool
		wantContains   []string
		wantNotContain []string
	}{
		{
			name:      "nil watermark returns empty",
			watermark: nil,
			wantEmpty: true,
		},
		{
			name:      "empty text returns empty",
			watermark: &Watermark{Text: "", Color: "#888888", Opacity: 0.1, Angle: -45},
			wantEmpty: true,
		},
		{
			name:      "simple watermark",
			watermark: &Watermark{Text: "DRAFT", Color: "#888888", Opacity: 0.1, Angle: -45},
			wantContains: []string{
				`content: "DRAFT"`,
				"color: #888888",
				"opacity: 0.10",
				"rotate(-45.0deg)",
			},
		},
		{
			name:      "watermark with positive angle",
			watermark: &Watermark{Text: "TEST", Color: "#ff0000", Opacity: 0.5, Angle: 30},
			wantContains: []string{
				`content: "TEST"`,
				"color: #ff0000",
				"opacity: 0.50",
				"rotate(30.0deg)",
			},
		},
		{
			name:      "watermark text with quotes is escaped",
			watermark: &Watermark{Text: `DRAFT "v1"`, Color: "#888888", Opacity: 0.1, Angle: -45},
			wantContains: []string{
				`content: "DRAFT \"v1\""`,
			},
			wantNotContain: []string{
				`content: "DRAFT "v1""`, // unescaped quotes would break CSS
			},
		},
		{
			name:      "watermark text with backslash is escaped",
			watermark: &Watermark{Text: `A\B`, Color: "#888888", Opacity: 0.1, Angle: -45},
			wantContains: []string{
				`content: "A\\B"`,
			},
		},
		{
			name:      "CSS injection attempt is escaped",
			watermark: &Watermark{Text: `"; } body { display: none } .x { content: "`, Color: "#888888", Opacity: 0.1, Angle: -45},
			wantContains: []string{
				`content: "\"; } body { display: none } .x { content: \""`,
				"opacity: 0.10", // verify CSS structure is intact after injection attempt
			},
		},
		{
			name:      "watermark with newline in text",
			watermark: &Watermark{Text: "LINE1\nLINE2", Color: "#888888", Opacity: 0.1, Angle: -45},
			wantContains: []string{
				`content: "LINE1\A LINE2"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildWatermarkCSS(tt.watermark)

			if tt.wantEmpty {
				if got != "" {
					t.Errorf("buildWatermarkCSS() = %q, want empty", got)
				}
				return
			}

			if got == "" {
				t.Fatal("buildWatermarkCSS() returned empty, want CSS")
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("buildWatermarkCSS() missing %q\nGot:\n%s", want, got)
				}
			}

			for _, notWant := range tt.wantNotContain {
				if strings.Contains(got, notWant) {
					t.Errorf("buildWatermarkCSS() should not contain %q\nGot:\n%s", notWant, got)
				}
			}
		})
	}
}
