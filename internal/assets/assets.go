// Package assets provides CSS styles and HTML templates for PDF generation.
// Assets can be loaded from embedded files or custom filesystem paths.
package assets

// defaultLoader is the package-level embedded loader for backward compatibility.
var defaultLoader = NewEmbeddedLoader()

// LoadStyle loads a CSS file by name using the default embedded loader.
// The name should not include the .css extension or path components.
// Returns ErrStyleNotFound if the style does not exist.
// Returns ErrInvalidAssetName if the name contains path separators or traversal.
func LoadStyle(name string) (string, error) {
	return defaultLoader.LoadStyle(name)
}

// LoadTemplate loads an HTML template by name using the default embedded loader.
// The name should not include the .html extension or path components.
// Returns ErrTemplateNotFound if the template does not exist.
// Returns ErrInvalidAssetName if the name contains path separators or traversal.
func LoadTemplate(name string) (string, error) {
	return defaultLoader.LoadTemplate(name)
}
