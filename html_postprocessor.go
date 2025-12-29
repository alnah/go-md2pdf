package main

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"strings"

	"github.com/alnah/go-md2pdf/internal/assets"
)

// Sentinel errors for signature injection.
var ErrSignatureRender = errors.New("failed to render signature template")

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

// SignatureData holds signature information for injection into HTML.
// Decoupled from SignatureConfig to keep service independent of config types.
type SignatureData struct {
	Name      string
	Title     string
	Email     string
	ImagePath string
	Links     []SignatureLink
}

// SignatureLink represents a clickable link in the signature block.
type SignatureLink struct {
	Label string
	URL   string
}

// SignatureInjector defines the contract for signature injection into HTML.
type SignatureInjector interface {
	InjectSignature(htmlContent string, data *SignatureData) (string, error)
}

// SignatureInjection renders and injects a signature block into HTML content.
type SignatureInjection struct {
	tmpl *template.Template
}

// NewSignatureInjection creates a SignatureInjection with the embedded template.
// Panics if the template cannot be loaded or parsed (programmer error).
func NewSignatureInjection() *SignatureInjection {
	tmplContent, err := assets.LoadTemplate("signature")
	if err != nil {
		panic("failed to load signature template: " + err.Error())
	}

	tmpl, err := template.New("signature").Parse(tmplContent)
	if err != nil {
		panic("failed to parse signature template: " + err.Error())
	}

	return &SignatureInjection{tmpl: tmpl}
}

// InjectSignature renders the signature template and injects it before </body>.
// If data is nil, returns htmlContent unchanged.
// Returns error if template rendering fails.
func (s *SignatureInjection) InjectSignature(htmlContent string, data *SignatureData) (string, error) {
	if data == nil {
		return htmlContent, nil
	}

	var buf bytes.Buffer
	if err := s.tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("%w: %v", ErrSignatureRender, err)
	}

	signatureHTML := buf.String()
	lowerHTML := strings.ToLower(htmlContent)

	// Try inserting before </body>
	if idx := strings.Index(lowerHTML, "</body>"); idx != -1 {
		return htmlContent[:idx] + signatureHTML + htmlContent[idx:], nil
	}

	// Fallback: append to end
	return htmlContent + signatureHTML, nil
}
