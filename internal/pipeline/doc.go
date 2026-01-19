// Package pipeline implements the Markdown-to-HTML conversion pipeline.
//
// This package handles preprocessing, HTML conversion, and HTML injection stages:
//   - Markdown preprocessing (line normalization, highlight syntax)
//   - Markdown to HTML conversion via Goldmark
//   - CSS injection into HTML documents
//   - Cover page injection
//   - Table of contents generation and injection
//   - Signature block injection
//
// PDF generation is handled separately by the root md2pdf package using
// headless Chrome (go-rod). This separation keeps the pipeline focused on
// document structure and content, while PDF rendering handles page layout,
// margins, and browser-based rendering concerns.
package pipeline
