package pipeline

import (
	"net/url"
	"path/filepath"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// RewriteRelativePaths converts relative image and link paths to absolute file:// URLs.
// If sourceDir is empty, returns the HTML unchanged.
//
// Rewrites:
//   - img[src]: relative paths to images
//   - a[href]: relative file paths (not anchors, not URLs)
//
// Does NOT rewrite (by design):
//   - video, audio, source elements (PDFs don't support media)
//   - srcset attributes (complex format, out of scope)
//   - CSS url() references (out of scope)
//   - script[src] (security)
//   - Absolute paths or URLs (already resolved)
func RewriteRelativePaths(htmlContent, sourceDir string) (string, error) {
	if sourceDir == "" {
		return htmlContent, nil
	}

	// Make sourceDir absolute for consistent path resolution
	absSourceDir, err := filepath.Abs(sourceDir)
	if err != nil {
		return "", err
	}

	// Parse HTML - detect if full document or fragment
	doc, isFragment, err := parseHTML(htmlContent)
	if err != nil {
		return "", err
	}

	// Rewrite paths in the document tree
	rewriteNode(doc, absSourceDir)

	// Render back to string
	return renderHTML(doc, isFragment)
}

// parseHTML parses HTML content, handling both full documents and fragments.
// Returns the parsed node, whether it was a fragment, and any error.
func parseHTML(content string) (*html.Node, bool, error) {
	trimmed := strings.TrimSpace(content)

	// Full document: starts with <!DOCTYPE or <html
	if strings.HasPrefix(strings.ToLower(trimmed), "<!doctype") ||
		strings.HasPrefix(strings.ToLower(trimmed), "<html") {
		doc, err := html.Parse(strings.NewReader(content))
		return doc, false, err
	}

	// Fragment: parse with body context to avoid wrapping
	context := &html.Node{
		Type:     html.ElementNode,
		DataAtom: atom.Body,
		Data:     "body",
	}
	nodes, err := html.ParseFragment(strings.NewReader(content), context)
	if err != nil {
		return nil, true, err
	}

	// Wrap nodes in a container for uniform traversal
	container := &html.Node{Type: html.DocumentNode}
	for _, n := range nodes {
		container.AppendChild(n)
	}

	return container, true, nil
}

// renderHTML renders the document back to string.
// For fragments, only renders the children (avoids adding <html><body> wrapper).
func renderHTML(doc *html.Node, isFragment bool) (string, error) {
	var buf strings.Builder

	if isFragment {
		// Render each child directly
		for c := doc.FirstChild; c != nil; c = c.NextSibling {
			if err := html.Render(&buf, c); err != nil {
				return "", err
			}
		}
		return buf.String(), nil
	}

	// Full document: render normally
	if err := html.Render(&buf, doc); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// rewriteNode traverses the DOM and rewrites relative paths.
func rewriteNode(n *html.Node, sourceDir string) {
	if n.Type == html.ElementNode {
		switch n.Data {
		case "img":
			rewriteAttr(n, "src", sourceDir)
		case "a":
			rewriteAttr(n, "href", sourceDir)
			// Note: video, audio, source intentionally NOT rewritten (PDFs don't support media)
			// Note: srcset intentionally NOT rewritten (complex format, out of scope)
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		rewriteNode(c, sourceDir)
	}
}

// rewriteAttr rewrites a single attribute if it's a relative path.
func rewriteAttr(n *html.Node, attrName, sourceDir string) {
	for i, attr := range n.Attr {
		if attr.Key != attrName {
			continue
		}
		if !isRelativePath(attr.Val) {
			continue
		}

		absPath := filepath.Join(sourceDir, attr.Val)

		// Security: validate path is under sourceDir (prevent traversal)
		if !isPathUnderDir(absPath, sourceDir) {
			continue // Skip rewriting, leave original path
		}

		// Convert to file:// URL (handles Windows paths correctly)
		n.Attr[i].Val = pathToFileURL(absPath)
	}
}

// isRelativePath returns true if the path should be rewritten.
func isRelativePath(path string) bool {
	if path == "" {
		return false
	}

	// Skip URLs (http, https, file, data, protocol-relative)
	if strings.HasPrefix(path, "http://") ||
		strings.HasPrefix(path, "https://") ||
		strings.HasPrefix(path, "file://") ||
		strings.HasPrefix(path, "data:") ||
		strings.HasPrefix(path, "//") {
		return false
	}

	// Skip anchors
	if strings.HasPrefix(path, "#") {
		return false
	}

	// Skip absolute paths
	if filepath.IsAbs(path) {
		return false
	}

	return true
}

// isPathUnderDir checks if absPath is under dir (prevents path traversal).
func isPathUnderDir(absPath, dir string) bool {
	cleanPath := filepath.Clean(absPath)
	cleanDir := filepath.Clean(dir)

	// Ensure dir ends with separator for correct prefix matching
	if !strings.HasSuffix(cleanDir, string(filepath.Separator)) {
		cleanDir += string(filepath.Separator)
	}

	// Path is under dir if it starts with dir/ or equals dir
	return strings.HasPrefix(cleanPath+string(filepath.Separator), cleanDir)
}

// pathToFileURL converts an absolute path to a file:// URL.
// Handles both Unix and Windows paths correctly.
func pathToFileURL(absPath string) string {
	// filepath.ToSlash handles Windows backslashes
	u := url.URL{
		Scheme: "file",
		Path:   filepath.ToSlash(absPath),
	}
	return u.String()
}
