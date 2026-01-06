package md2pdf

import (
	"bytes"
	"context"
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

// htmlConverter abstracts Markdown to HTML conversion.
type htmlConverter interface {
	ToHTML(ctx context.Context, content string) (string, error)
}

// goldmarkConverter converts Markdown to HTML using goldmark (pure Go).
type goldmarkConverter struct {
	md goldmark.Markdown
}

// newGoldmarkConverter creates a goldmarkConverter with GFM extensions.
func newGoldmarkConverter() *goldmarkConverter {
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
	return &goldmarkConverter{md: md}
}

// ToHTML converts Markdown content to a standalone HTML5 document.
// Supports context cancellation via goroutine + select pattern since
// Goldmark doesn't natively support context.
func (c *goldmarkConverter) ToHTML(ctx context.Context, content string) (string, error) {
	// Fast path: check context before starting
	if err := ctx.Err(); err != nil {
		return "", err
	}

	type result struct {
		html string
		err  error
	}

	done := make(chan result, 1)

	go func() {
		var buf bytes.Buffer
		if err := c.md.Convert([]byte(content), &buf); err != nil {
			done <- result{err: fmt.Errorf("%w: %v", ErrHTMLConversion, err)}
			return
		}
		done <- result{html: fmt.Sprintf(htmlTemplate, buf.String())}
	}()

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case r := <-done:
		return r.html, r.err
	}
}
