package md2pdf

import (
	"fmt"
	"strings"
)

// defaultFontFamily is the standard font stack for PDF footers and generated content.
const defaultFontFamily = "sans-serif"

// watermarkFontSize is the font size for watermark text overlay.
const watermarkFontSize = "8rem"

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
