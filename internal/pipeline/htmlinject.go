package pipeline

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html"
	"html/template"
	"regexp"
	"strconv"
	"strings"
)

// Sentinel errors for template rendering.
var (
	ErrSignatureRender = errors.New("signature template rendering failed")
	ErrCoverRender     = errors.New("cover template rendering failed")
)

// CSSInjector defines the contract for CSS injection into HTML.
type CSSInjector interface {
	InjectCSS(ctx context.Context, htmlContent, cssContent string) string
}

// CSSInjection injects CSS as a <style> block into HTML content.
type CSSInjection struct{}

// InjectCSS inserts a <style> block into HTML content.
// Tries </head> first, then <body>, then prepends to the HTML.
// CSS content is sanitized to prevent injection attacks.
func (s *CSSInjection) InjectCSS(ctx context.Context, htmlContent, cssContent string) string {
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

// SignatureData holds signature information for injection into HTML.
type SignatureData struct {
	Name         string
	Title        string
	Email        string
	Organization string
	ImagePath    string
	Links        []SignatureLink
	// Extended metadata fields
	Phone      string
	Address    string
	Department string
}

// SignatureLink represents a clickable link in the signature block.
type SignatureLink struct {
	Label string
	URL   string
}

// SignatureInjector defines the contract for signature injection into HTML.
type SignatureInjector interface {
	InjectSignature(ctx context.Context, htmlContent string, data *SignatureData) (string, error)
}

// SignatureInjection renders and injects a signature block into HTML content.
type SignatureInjection struct {
	tmpl *template.Template
}

// NewSignatureInjection creates a SignatureInjection from template content.
// Returns error if the template cannot be parsed.
func NewSignatureInjection(tmplContent string) (*SignatureInjection, error) {
	tmpl, err := template.New("signature").Parse(tmplContent)
	if err != nil {
		return nil, fmt.Errorf("parsing signature template: %w", err)
	}

	return &SignatureInjection{tmpl: tmpl}, nil
}

// InjectSignature renders the signature template and injects it before </body>.
// If data is nil, returns htmlContent unchanged.
// Returns error if template rendering fails.
func (s *SignatureInjection) InjectSignature(ctx context.Context, htmlContent string, data *SignatureData) (string, error) {
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

// CoverData holds cover page information for injection into HTML.
type CoverData struct {
	Title        string
	Subtitle     string
	Logo         string
	Author       string
	AuthorTitle  string
	Organization string
	Date         string
	Version      string
	// Extended metadata fields
	ClientName   string
	ProjectName  string
	DocumentType string
	DocumentID   string
	Description  string
	Department   string // From author config (DRY)
}

// CoverInjector defines the contract for cover injection into HTML.
type CoverInjector interface {
	InjectCover(ctx context.Context, htmlContent string, data *CoverData) (string, error)
}

// CoverInjection renders and injects a cover page into HTML content.
type CoverInjection struct {
	tmpl *template.Template
}

// NewCoverInjection creates a CoverInjection from template content.
// Returns error if the template cannot be parsed.
func NewCoverInjection(tmplContent string) (*CoverInjection, error) {
	tmpl, err := template.New("cover").Parse(tmplContent)
	if err != nil {
		return nil, fmt.Errorf("parsing cover template: %w", err)
	}

	return &CoverInjection{tmpl: tmpl}, nil
}

// InjectCover renders the cover template and injects it after <body>.
// If data is nil, returns htmlContent unchanged.
// Returns error if template rendering fails.
func (c *CoverInjection) InjectCover(ctx context.Context, htmlContent string, data *CoverData) (string, error) {
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

// FooterData holds footer configuration for injection into HTML.
type FooterData struct {
	Position       string // "left", "center", "right" (default: "right")
	ShowPageNumber bool
	Date           string
	Status         string
	Text           string
	DocumentID     string // Document reference number
}

// TOCData holds TOC configuration for injection.
type TOCData struct {
	Title    string
	MinDepth int // Minimum heading level (default: 2, skips H1)
	MaxDepth int // Maximum heading level (default: 3)
}

// TOCInjector defines the contract for TOC injection into HTML.
type TOCInjector interface {
	InjectTOC(ctx context.Context, htmlContent string, data *TOCData) (string, error)
}

// headingInfo represents an extracted heading from HTML.
type headingInfo struct {
	Level int    // 1-6
	ID    string // anchor ID
	Text  string // heading text content
}

// headingPattern matches h1-h6 tags with id attribute.
// Captures: 1=level, 2=id, 3=inner HTML (may contain inline tags)
var headingPattern = regexp.MustCompile(`(?is)<h([1-6])[^>]*\bid="([^"]*)"[^>]*>(.*?)</h[1-6]>`)

// htmlTagPattern matches HTML tags for stripping from heading text.
var htmlTagPattern = regexp.MustCompile(`<[^>]*>`)

// stripHTMLTags removes HTML tags from a string, decodes HTML entities,
// and trims whitespace. Decoding entities is essential to avoid double-encoding
// when the text is later escaped for HTML output (e.g., in TOC generation).
func stripHTMLTags(s string) string {
	s = htmlTagPattern.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	return strings.TrimSpace(s)
}

// extractHeadings parses HTML and returns headings between minDepth and maxDepth.
// Headings without IDs are skipped.
func extractHeadings(htmlContent string, minDepth, maxDepth int) []headingInfo {
	matches := headingPattern.FindAllStringSubmatch(htmlContent, -1)
	if len(matches) == 0 {
		return nil
	}

	var headings []headingInfo
	for _, m := range matches {
		level, _ := strconv.Atoi(m[1])
		if level < minDepth || level > maxDepth {
			continue
		}
		headings = append(headings, headingInfo{
			Level: level,
			ID:    m[2],
			Text:  stripHTMLTags(m[3]),
		})
	}
	return headings
}

// numberingState tracks hierarchical numbering for TOC entries.
// Supports normalization (first heading becomes level 1) and gap skipping.
type numberingState struct {
	counters     [6]int // counters[0] = level 1 count, etc.
	minLevelSeen int    // for normalization (0 = not set)
	lastLevel    int    // for tracking parent relationships
}

// newNumberingState creates a new numbering state.
func newNumberingState() *numberingState {
	return &numberingState{minLevelSeen: 0, lastLevel: 0}
}

// next returns the next number string and effective depth for the given heading level.
// Handles normalization and gap skipping.
// The effective depth is used for nesting decisions in TOC generation.
func (n *numberingState) next(level int) (numStr string, effectiveDepth int) {
	// Initialize minLevelSeen on first heading
	if n.minLevelSeen == 0 {
		n.minLevelSeen = level
	}

	// Calculate effective depth (1-based, normalized)
	effectiveDepth = level - n.minLevelSeen + 1
	if effectiveDepth < 1 {
		effectiveDepth = 1
	}

	// Handle gap skipping: if we jump levels, treat as direct child
	// E.g., H1 -> H3 becomes depth 1 -> depth 2 (not depth 3)
	if n.lastLevel > 0 && effectiveDepth > n.lastLevel+1 {
		effectiveDepth = n.lastLevel + 1
	}

	// Reset deeper level counters
	for i := effectiveDepth; i < 6; i++ {
		n.counters[i] = 0
	}

	// Increment current level
	n.counters[effectiveDepth-1]++
	n.lastLevel = effectiveDepth

	// Build number string: "1.2.3."
	var parts []string
	for i := 0; i < effectiveDepth; i++ {
		parts = append(parts, strconv.Itoa(n.counters[i]))
	}
	return strings.Join(parts, ".") + ".", effectiveDepth
}

// generateNumberedTOC creates HTML for a numbered table of contents.
// Uses <div> elements instead of <ul>/<li> to avoid CSS list-style conflicts.
func generateNumberedTOC(headings []headingInfo, title string) string {
	if len(headings) == 0 {
		return ""
	}

	var buf strings.Builder
	buf.WriteString(`<nav class="toc">`)

	if title != "" {
		buf.WriteString(`<h2 class="toc-title">`)
		buf.WriteString(html.EscapeString(title))
		buf.WriteString(`</h2>`)
	}

	buf.WriteString(`<div class="toc-list">`)

	numbering := newNumberingState()

	for _, h := range headings {
		// Get number and effective depth (handles normalization and gap skipping)
		num, effectiveDepth := numbering.next(h.Level)

		// Calculate indentation: (depth - 1) * 1.5em
		indent := float64(effectiveDepth-1) * 1.5

		// Write the TOC item
		buf.WriteString(`<div class="toc-item"`)
		if indent > 0 {
			buf.WriteString(fmt.Sprintf(` style="padding-left:%.1fem"`, indent))
		}
		buf.WriteString(`><a href="#`)
		buf.WriteString(html.EscapeString(h.ID))
		buf.WriteString(`">`)
		buf.WriteString(num)
		buf.WriteString(` `)
		buf.WriteString(html.EscapeString(h.Text))
		buf.WriteString(`</a></div>`)
	}

	buf.WriteString(`</div></nav>`)
	return buf.String()
}

// TOCInjection implements TOCInjector.
type TOCInjection struct{}

// NewTOCInjection creates a new TOC injector.
func NewTOCInjection() *TOCInjection {
	return &TOCInjection{}
}

// InjectTOC extracts headings and injects a numbered TOC after the cover page.
// If data is nil, returns htmlContent unchanged.
func (t *TOCInjection) InjectTOC(ctx context.Context, htmlContent string, data *TOCData) (string, error) {
	if data == nil {
		return htmlContent, nil
	}

	// Check for cancellation
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	// Extract headings
	headings := extractHeadings(htmlContent, data.MinDepth, data.MaxDepth)
	if len(headings) == 0 {
		return htmlContent, nil
	}

	// Generate TOC HTML
	tocHTML := generateNumberedTOC(headings, data.Title)
	if tocHTML == "" {
		return htmlContent, nil
	}

	lowerHTML := strings.ToLower(htmlContent)

	// Try inserting after cover page marker.
	// Note: We use <span data-cover-end> instead of <!-- cover-end --> comment
	// because html/template strips HTML comments for security reasons.
	coverEndPattern := regexp.MustCompile(`(?i)</div>\s*</section>\s*<span[^>]*data-cover-end[^>]*>\s*</span>`)
	if loc := coverEndPattern.FindStringIndex(htmlContent); loc != nil {
		insertPos := loc[1]
		return htmlContent[:insertPos] + tocHTML + htmlContent[insertPos:], nil
	}

	// Fallback: insert after <body> tag
	if idx := strings.Index(lowerHTML, "<body"); idx != -1 {
		closeIdx := strings.Index(htmlContent[idx:], ">")
		if closeIdx != -1 {
			insertPos := idx + closeIdx + 1
			return htmlContent[:insertPos] + tocHTML + htmlContent[insertPos:], nil
		}
	}

	// Last fallback: prepend
	return tocHTML + htmlContent, nil
}
