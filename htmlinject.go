package md2pdf

import (
	"bytes"
	"context"
	"fmt"
	"html"
	"html/template"
	"regexp"
	"strconv"
	"strings"

	"github.com/alnah/go-md2pdf/internal/assets"
)

// defaultFontFamily is the standard font stack for PDF footers and generated content.
const defaultFontFamily = "'Inter', sans-serif"

// watermarkFontSize is the font size for watermark text overlay.
const watermarkFontSize = "8rem"

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
	Name         string
	Title        string
	Email        string
	Organization string
	ImagePath    string
	Links        []signatureLink
	// Extended metadata fields
	Phone      string
	Address    string
	Department string
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

// newSignatureInjection creates a signatureInjection using the provided AssetLoader.
// Returns error if the template cannot be loaded or parsed.
func newSignatureInjection(loader assets.AssetLoader) (*signatureInjection, error) {
	tmplContent, err := loader.LoadTemplate("signature")
	if err != nil {
		return nil, fmt.Errorf("loading signature template: %w", err)
	}

	tmpl, err := template.New("signature").Parse(tmplContent)
	if err != nil {
		return nil, fmt.Errorf("parsing signature template: %w", err)
	}

	return &signatureInjection{tmpl: tmpl}, nil
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
	// Extended metadata fields
	ClientName   string
	ProjectName  string
	DocumentType string
	DocumentID   string
	Description  string
	Department   string // From author config (DRY)
}

// coverInjector defines the contract for cover injection into HTML.
type coverInjector interface {
	InjectCover(ctx context.Context, htmlContent string, data *coverData) (string, error)
}

// coverInjection renders and injects a cover page into HTML content.
type coverInjection struct {
	tmpl *template.Template
}

// newCoverInjection creates a coverInjection using the provided AssetLoader.
// Returns error if the template cannot be loaded or parsed.
func newCoverInjection(loader assets.AssetLoader) (*coverInjection, error) {
	tmplContent, err := loader.LoadTemplate("cover")
	if err != nil {
		return nil, fmt.Errorf("loading cover template: %w", err)
	}

	tmpl, err := template.New("cover").Parse(tmplContent)
	if err != nil {
		return nil, fmt.Errorf("parsing cover template: %w", err)
	}

	return &coverInjection{tmpl: tmpl}, nil
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
	DocumentID     string // Document reference number
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
  font-size: %s;
  font-weight: bold;
  color: %s;
  opacity: %.2f;
  z-index: -1;
  pointer-events: none;
  white-space: nowrap;
  font-family: %s;
}
`, escapeCSSString(breakURLPattern(w.Text)), w.Angle, watermarkFontSize, w.Color, w.Opacity, defaultFontFamily)
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

// breakURLPattern replaces ALL dots with a Unicode lookalike (ONE DOT LEADER U+2024)
// to prevent PDF viewers from auto-detecting URLs and making them clickable.
// The character ․ looks identical to . but is not recognized as a URL separator.
//
// Note: This affects all dots unconditionally, including version numbers (1.0.0),
// abbreviations (e.g.), and decimal numbers. This is intentional - the U+2024
// character is visually indistinguishable from a period in rendered output.
func breakURLPattern(text string) string {
	return strings.ReplaceAll(text, ".", "\u2024")
}

// buildPageBreaksCSS generates CSS for page break control.
// Always includes hardcoded rules for heading protection (break-after/inside: avoid).
// Configurable rules for page breaks before h1/h2/h3 and orphan/widow control.
func buildPageBreaksCSS(pb *PageBreaks) string {
	var buf strings.Builder

	buf.WriteString(`
/* Page breaks: always active - prevent heading alone at page bottom */
h1, h2, h3, h4, h5, h6 {
  break-after: avoid;
  page-break-after: avoid;
  break-inside: avoid;
  page-break-inside: avoid;
}
`)

	// Resolve orphans/widows (0 means use default)
	orphans := DefaultOrphans
	widows := DefaultWidows
	if pb != nil {
		if pb.Orphans > 0 {
			orphans = pb.Orphans
		}
		if pb.Widows > 0 {
			widows = pb.Widows
		}
	}

	buf.WriteString(fmt.Sprintf(`
/* Page breaks: orphan/widow control */
p, li, dd, dt, blockquote {
  orphans: %d;
  widows: %d;
}
`, orphans, widows))

	// Configurable page breaks before headings
	if pb != nil && pb.BeforeH1 {
		buf.WriteString(`
/* Page breaks: before H1 */
h1 {
  break-before: page;
  page-break-before: always;
}
/* Exception: no break before first H1 if it's first element in body */
body > h1:first-child {
  break-before: auto;
  page-break-before: auto;
}
`)
	}

	if pb != nil && pb.BeforeH2 {
		buf.WriteString(`
/* Page breaks: before H2 */
h2 {
  break-before: page;
  page-break-before: always;
}
`)
	}

	if pb != nil && pb.BeforeH3 {
		buf.WriteString(`
/* Page breaks: before H3 */
h3 {
  break-before: page;
  page-break-before: always;
}
`)
	}

	return buf.String()
}

// tocData holds TOC configuration for injection.
type tocData struct {
	Title    string
	MinDepth int // Minimum heading level (default: 2, skips H1)
	MaxDepth int // Maximum heading level (default: 3)
}

// tocInjector defines the contract for TOC injection into HTML.
type tocInjector interface {
	InjectTOC(ctx context.Context, htmlContent string, data *tocData) (string, error)
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

// tocInjection implements tocInjector.
type tocInjection struct{}

// newTOCInjection creates a new TOC injector.
func newTOCInjection() *tocInjection {
	return &tocInjection{}
}

// InjectTOC extracts headings and injects a numbered TOC after the cover page.
// If data is nil, returns htmlContent unchanged.
func (t *tocInjection) InjectTOC(ctx context.Context, htmlContent string, data *tocData) (string, error) {
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
