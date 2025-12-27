package main

import "strings"

// CSSInjector defines the contract for CSS injection into HTML.
type CSSInjector interface {
	InjectCSS(htmlContent, cssContent string) string
}

// CSSInjection injects CSS as a <style> block into HTML content.
type CSSInjection struct{}

// InjectCSS inserts a <style> block into HTML content.
// Tries </head> first, then <body>, then prepends to the HTML.
// CSS content is sanitized to prevent injection attacks.
func (s *CSSInjection) InjectCSS(htmlContent, cssContent string) string {
	if cssContent == "" {
		return htmlContent
	}

	sanitizedCSS := SanitizeCSS(cssContent)
	styleBlock := "<style>" + sanitizedCSS + "</style>"
	lowerHTML := strings.ToLower(htmlContent)

	// Try inserting before </head>
	if idx := strings.Index(lowerHTML, "</head>"); idx != -1 {
		return htmlContent[:idx] + styleBlock + htmlContent[idx:]
	}

	// Try inserting after <body>
	if idx := strings.Index(lowerHTML, "<body"); idx != -1 {
		// Find the closing > of <body...>
		closeIdx := strings.Index(htmlContent[idx:], ">")
		if closeIdx != -1 {
			insertPos := idx + closeIdx + 1
			return htmlContent[:insertPos] + styleBlock + htmlContent[insertPos:]
		}
	}

	// Fallback: prepend
	return styleBlock + htmlContent
}

// SanitizeCSS escapes sequences that could break out of a <style> block.
// Prevents CSS injection by escaping </style> and similar closing sequences.
func SanitizeCSS(css string) string {
	// Escape </ sequences to prevent closing the style tag prematurely
	return strings.ReplaceAll(css, "</", `<\/`)
}
