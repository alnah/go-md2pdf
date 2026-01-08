package md2pdf

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"strings"

	"github.com/alnah/go-md2pdf/internal/assets"
)

// defaultFontFamily is the standard font stack for PDF footers and generated content.
const defaultFontFamily = "'Inter', sans-serif"

// cssInjector defines the contract for CSS injection into HTML.
type cssInjector interface {
	InjectCSS(ctx context.Context, htmlContent, cssContent string) string
}

// cssInjection injects CSS as a <style> block into HTML content.
type cssInjection struct{}

// InjectCSS inserts a <style> block into HTML content.
// Tries </head> first, then <body>, then prepends to the HTML.
// CSS content is sanitized to prevent injection attacks.
func (s *cssInjection) InjectCSS(ctx context.Context, htmlContent, cssContent string) string {
	if cssContent == "" {
		return htmlContent
	}

	// Check for cancellation
	if ctx.Err() != nil {
		return htmlContent
	}

	sanitizedCSS := sanitizeCSS(cssContent)
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

// sanitizeCSS escapes sequences that could break out of a <style> block.
// Prevents CSS injection by escaping </style> and similar closing sequences.
func sanitizeCSS(css string) string {
	// Escape </ sequences to prevent closing the style tag prematurely
	return strings.ReplaceAll(css, "</", `<\/`)
}

// signatureData holds signature information for injection into HTML.
// This is the internal type used by the injector.
type signatureData struct {
	Name      string
	Title     string
	Email     string
	ImagePath string
	Links     []signatureLink
}

// signatureLink represents a clickable link in the signature block.
type signatureLink struct {
	Label string
	URL   string
}

// signatureInjector defines the contract for signature injection into HTML.
type signatureInjector interface {
	InjectSignature(ctx context.Context, htmlContent string, data *signatureData) (string, error)
}

// signatureInjection renders and injects a signature block into HTML content.
type signatureInjection struct {
	tmpl *template.Template
}

// newSignatureInjection creates a signatureInjection with the embedded template.
// Panics if the template cannot be loaded or parsed (programmer error).
func newSignatureInjection() *signatureInjection {
	tmplContent, err := assets.LoadTemplate("signature")
	if err != nil {
		panic("failed to load signature template: " + err.Error())
	}

	tmpl, err := template.New("signature").Parse(tmplContent)
	if err != nil {
		panic("failed to parse signature template: " + err.Error())
	}

	return &signatureInjection{tmpl: tmpl}
}

// InjectSignature renders the signature template and injects it before </body>.
// If data is nil, returns htmlContent unchanged.
// Returns error if template rendering fails.
func (s *signatureInjection) InjectSignature(ctx context.Context, htmlContent string, data *signatureData) (string, error) {
	if data == nil {
		return htmlContent, nil
	}

	// Check for cancellation
	if ctx.Err() != nil {
		return "", ctx.Err()
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

// coverData holds cover page information for injection into HTML.
// This is the internal type used by the injector.
type coverData struct {
	Title        string
	Subtitle     string
	Logo         string
	Author       string
	AuthorTitle  string
	Organization string
	Date         string
	Version      string
}

// coverInjector defines the contract for cover injection into HTML.
type coverInjector interface {
	InjectCover(ctx context.Context, htmlContent string, data *coverData) (string, error)
}

// coverInjection renders and injects a cover page into HTML content.
type coverInjection struct {
	tmpl *template.Template
}

// newCoverInjection creates a coverInjection with the embedded template.
// Panics if the template cannot be loaded or parsed (programmer error).
func newCoverInjection() *coverInjection {
	tmplContent, err := assets.LoadTemplate("cover")
	if err != nil {
		panic("failed to load cover template: " + err.Error())
	}

	tmpl, err := template.New("cover").Parse(tmplContent)
	if err != nil {
		panic("failed to parse cover template: " + err.Error())
	}

	return &coverInjection{tmpl: tmpl}
}

// InjectCover renders the cover template and injects it after <body>.
// If data is nil, returns htmlContent unchanged.
// Returns error if template rendering fails.
func (c *coverInjection) InjectCover(ctx context.Context, htmlContent string, data *coverData) (string, error) {
	if data == nil {
		return htmlContent, nil
	}

	// Check for cancellation
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	var buf bytes.Buffer
	if err := c.tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("%w: %v", ErrCoverRender, err)
	}

	coverHTML := buf.String()
	lowerHTML := strings.ToLower(htmlContent)

	// Try inserting after <body>
	if idx := strings.Index(lowerHTML, "<body"); idx != -1 {
		// Find the closing > of <body...>
		closeIdx := strings.Index(htmlContent[idx:], ">")
		if closeIdx != -1 {
			insertPos := idx + closeIdx + 1
			return htmlContent[:insertPos] + coverHTML + htmlContent[insertPos:], nil
		}
	}

	// Fallback: prepend
	return coverHTML + htmlContent, nil
}

// footerData holds footer configuration for injection into HTML.
// This is the internal type used by the injector.
type footerData struct {
	Position       string // "left", "center", "right" (default: "right")
	ShowPageNumber bool
	Date           string
	Status         string
	Text           string
}

// buildWatermarkCSS generates CSS for a diagonal background watermark.
// The watermark uses position:fixed to appear on all pages when printed.
func buildWatermarkCSS(w *Watermark) string {
	if w == nil || w.Text == "" {
		return ""
	}

	return fmt.Sprintf(`
/* Watermark */
body::before {
  content: "%s";
  position: fixed;
  top: 50%%;
  left: 50%%;
  transform: translate(-50%%, -50%%) rotate(%.1fdeg);
  font-size: 8rem;
  font-weight: bold;
  color: %s;
  opacity: %.2f;
  z-index: -1;
  pointer-events: none;
  white-space: nowrap;
  font-family: %s;
}
`, escapeCSSString(w.Text), w.Angle, w.Color, w.Opacity, defaultFontFamily)
}

// escapeCSSString escapes a string for safe use in CSS content property.
// Prevents CSS injection by escaping backslashes, quotes, newlines, and
// percent signs (to avoid fmt.Sprintf format string issues).
func escapeCSSString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\A `)
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, `%`, `%%`)
	return s
}
