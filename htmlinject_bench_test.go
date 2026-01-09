//go:build bench

package md2pdf

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// BenchmarkInjectCSS benchmarks CSS injection into HTML.
// Critical for styling as it's called on every conversion.
func BenchmarkInjectCSS(b *testing.B) {
	injector := &cssInjection{}
	ctx := context.Background()

	smallHTML := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body><h1>Hello</h1></body>
</html>`

	largeHTML := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>` + strings.Repeat("<p>Paragraph content here.</p>\n", 500) + `</body>
</html>`

	smallCSS := "body { margin: 0; }"
	largeCSS := strings.Repeat(".class-name { color: red; font-size: 14px; margin: 10px; }\n", 100)

	inputs := []struct {
		name string
		html string
		css  string
	}{
		{"small_html_small_css", smallHTML, smallCSS},
		{"small_html_large_css", smallHTML, largeCSS},
		{"large_html_small_css", largeHTML, smallCSS},
		{"large_html_large_css", largeHTML, largeCSS},
		{"no_head_tag", "<body><p>Content</p></body>", smallCSS},
		{"empty_css", smallHTML, ""},
	}

	for _, input := range inputs {
		b.Run(input.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				result := injector.InjectCSS(ctx, input.html, input.css)
				_ = result
			}
		})
	}
}

// BenchmarkSanitizeCSS benchmarks CSS sanitization.
// Tests escaping of potentially dangerous sequences.
func BenchmarkSanitizeCSS(b *testing.B) {
	inputs := []struct {
		name string
		css  string
	}{
		{"clean", strings.Repeat(".class { color: red; }\n", 50)},
		{"with_escapes", strings.Repeat(".class { content: '</style>'; }\n", 50)},
		{"large_clean", strings.Repeat(".class { color: red; font-size: 14px; }\n", 500)},
	}

	for _, input := range inputs {
		b.Run(input.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				result := sanitizeCSS(input.css)
				_ = result
			}
		})
	}
}

// BenchmarkInjectSignature benchmarks signature block injection.
func BenchmarkInjectSignature(b *testing.B) {
	injector := newSignatureInjection()
	ctx := context.Background()

	html := generateTestHTML(100)

	signatures := []struct {
		name string
		data *signatureData
	}{
		{"nil", nil},
		{"minimal", &signatureData{Name: "John Doe"}},
		{"full", &signatureData{
			Name:         "John Doe",
			Title:        "Senior Engineer",
			Email:        "john@example.com",
			Organization: "Example Corp",
			ImagePath:    "/path/to/signature.png",
			Links: []signatureLink{
				{Label: "LinkedIn", URL: "https://linkedin.com/in/johndoe"},
				{Label: "GitHub", URL: "https://github.com/johndoe"},
			},
		}},
	}

	for _, sig := range signatures {
		b.Run(sig.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				result, err := injector.InjectSignature(ctx, html, sig.data)
				if err != nil {
					b.Fatal(err)
				}
				_ = result
			}
		})
	}
}

// BenchmarkInjectCover benchmarks cover page injection.
func BenchmarkInjectCover(b *testing.B) {
	injector := newCoverInjection()
	ctx := context.Background()

	html := generateTestHTML(100)

	covers := []struct {
		name string
		data *coverData
	}{
		{"nil", nil},
		{"minimal", &coverData{Title: "Document Title"}},
		{"full", &coverData{
			Title:        "Comprehensive Guide",
			Subtitle:     "A Deep Dive into Topics",
			Logo:         "https://example.com/logo.png",
			Author:       "John Doe",
			AuthorTitle:  "Senior Engineer",
			Organization: "Example Corporation",
			Date:         "2025-01-08",
			Version:      "1.0.0",
		}},
	}

	for _, cover := range covers {
		b.Run(cover.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				result, err := injector.InjectCover(ctx, html, cover.data)
				if err != nil {
					b.Fatal(err)
				}
				_ = result
			}
		})
	}
}

// BenchmarkExtractHeadings benchmarks heading extraction from HTML.
// Critical for TOC generation.
func BenchmarkExtractHeadings(b *testing.B) {
	htmls := []struct {
		name    string
		content string
		depth   int
	}{
		{"few_headings", generateHTMLWithHeadings(10), 3},
		{"many_headings", generateHTMLWithHeadings(100), 3},
		{"deep_headings", generateHTMLWithHeadings(50), 6},
		{"shallow_headings", generateHTMLWithHeadings(50), 1},
	}

	for _, h := range htmls {
		b.Run(h.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				result := extractHeadings(h.content, h.depth)
				_ = result
			}
		})
	}
}

// BenchmarkGenerateNumberedTOC benchmarks TOC HTML generation.
func BenchmarkGenerateNumberedTOC(b *testing.B) {
	headingCounts := []int{5, 20, 50, 100}

	for _, count := range headingCounts {
		headings := generateHeadingInfos(count)
		b.Run(fmt.Sprintf("headings_%d", count), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				result := generateNumberedTOC(headings, "Table of Contents")
				_ = result
			}
		})
	}
}

// BenchmarkInjectTOC benchmarks full TOC injection.
func BenchmarkInjectTOC(b *testing.B) {
	injector := newTOCInjection()
	ctx := context.Background()

	htmls := []struct {
		name string
		html string
		data *tocData
	}{
		{"nil_data", generateHTMLWithHeadings(20), nil},
		{"shallow", generateHTMLWithHeadings(20), &tocData{Title: "Contents", MaxDepth: 2}},
		{"deep", generateHTMLWithHeadings(50), &tocData{Title: "Table of Contents", MaxDepth: 6}},
		{"no_title", generateHTMLWithHeadings(20), &tocData{Title: "", MaxDepth: 3}},
	}

	for _, h := range htmls {
		b.Run(h.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				result, err := injector.InjectTOC(ctx, h.html, h.data)
				if err != nil {
					b.Fatal(err)
				}
				_ = result
			}
		})
	}
}

// BenchmarkBuildWatermarkCSS benchmarks watermark CSS generation.
func BenchmarkBuildWatermarkCSS(b *testing.B) {
	watermarks := []struct {
		name string
		data *Watermark
	}{
		{"nil", nil},
		{"simple", &Watermark{Text: "DRAFT", Color: "#888888", Opacity: 0.1, Angle: -45}},
		{"long_text", &Watermark{Text: "CONFIDENTIAL DOCUMENT", Color: "#ff0000", Opacity: 0.2, Angle: -30}},
	}

	for _, wm := range watermarks {
		b.Run(wm.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				result := buildWatermarkCSS(wm.data)
				_ = result
			}
		})
	}
}

// BenchmarkBuildPageBreaksCSS benchmarks page breaks CSS generation.
func BenchmarkBuildPageBreaksCSS(b *testing.B) {
	configs := []struct {
		name string
		data *PageBreaks
	}{
		{"nil", nil},
		{"defaults", &PageBreaks{Orphans: 2, Widows: 2}},
		{"all_breaks", &PageBreaks{BeforeH1: true, BeforeH2: true, BeforeH3: true, Orphans: 3, Widows: 3}},
	}

	for _, cfg := range configs {
		b.Run(cfg.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				result := buildPageBreaksCSS(cfg.data)
				_ = result
			}
		})
	}
}

// BenchmarkEscapeCSSString benchmarks CSS string escaping.
func BenchmarkEscapeCSSString(b *testing.B) {
	inputs := []struct {
		name  string
		value string
	}{
		{"clean", "DRAFT"},
		{"with_quotes", `Text with "quotes"`},
		{"with_backslash", `Path\to\file`},
		{"with_newlines", "Line1\nLine2\r\nLine3"},
		{"with_percent", "100% Complete"},
		{"complex", `"Complex\nString" with 100% escapes`},
	}

	for _, input := range inputs {
		b.Run(input.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				result := escapeCSSString(input.value)
				_ = result
			}
		})
	}
}

// BenchmarkStripHTMLTags benchmarks HTML tag stripping.
func BenchmarkStripHTMLTags(b *testing.B) {
	inputs := []struct {
		name  string
		value string
	}{
		{"no_tags", "Plain text content"},
		{"simple_tags", "<strong>Bold</strong> and <em>italic</em>"},
		{"nested_tags", "<div><span><a href='#'>Link</a></span></div>"},
		{"many_tags", strings.Repeat("<span>text</span>", 50)},
	}

	for _, input := range inputs {
		b.Run(input.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				result := stripHTMLTags(input.value)
				_ = result
			}
		})
	}
}

// Helper functions

func generateTestHTML(paragraphs int) string {
	var sb strings.Builder
	sb.WriteString(`<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<h1>Document Title</h1>
`)
	for i := 0; i < paragraphs; i++ {
		sb.WriteString(fmt.Sprintf("<p>Paragraph %d with some content.</p>\n", i+1))
	}
	sb.WriteString("</body>\n</html>")
	return sb.String()
}

func generateHTMLWithHeadings(count int) string {
	var sb strings.Builder
	sb.WriteString(`<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
`)
	for i := 0; i < count; i++ {
		level := (i % 6) + 1
		id := fmt.Sprintf("heading-%d", i)
		sb.WriteString(fmt.Sprintf(`<h%d id="%s">Heading %d</h%d>`, level, id, i+1, level))
		sb.WriteString("\n<p>Some content under this heading.</p>\n")
	}
	sb.WriteString("</body>\n</html>")
	return sb.String()
}

func generateHeadingInfos(count int) []headingInfo {
	headings := make([]headingInfo, count)
	for i := 0; i < count; i++ {
		headings[i] = headingInfo{
			Level: (i % 3) + 1, // Levels 1-3
			ID:    fmt.Sprintf("heading-%d", i),
			Text:  fmt.Sprintf("Heading Number %d", i+1),
		}
	}
	return headings
}
