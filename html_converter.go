package main

import (
	"bytes"
	"fmt"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

// htmlTemplate wraps Goldmark's fragment output in a complete HTML5 document.
const htmlTemplate = `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>Document</title>
</head>
<body>
%s
</body>
</html>`

// HTMLConverter abstracts Markdown to HTML conversion.
type HTMLConverter interface {
	ToHTML(content string) (string, error)
}

// GoldmarkConverter converts Markdown to HTML using goldmark (pure Go).
type GoldmarkConverter struct {
	md goldmark.Markdown
}

// NewGoldmarkConverter creates a GoldmarkConverter with GFM extensions.
func NewGoldmarkConverter() *GoldmarkConverter {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,      // Tables, strikethrough, autolinks, task lists
			extension.Footnote, // [^1] footnotes
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(), // Treat newlines as <br>
			html.WithXHTML(),     // Self-closing tags
		),
	)
	return &GoldmarkConverter{md: md}
}

// ToHTML converts Markdown content to a standalone HTML5 document.
func (c *GoldmarkConverter) ToHTML(content string) (string, error) {
	var buf bytes.Buffer
	if err := c.md.Convert([]byte(content), &buf); err != nil {
		return "", fmt.Errorf("converting to HTML: %w", err)
	}
	return fmt.Sprintf(htmlTemplate, buf.String()), nil
}
