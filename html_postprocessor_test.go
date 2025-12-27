package main

import "testing"

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
