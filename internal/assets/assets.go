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

// LoadTemplateSet loads a template set by name using the default embedded loader.
// The name identifies a directory containing cover.html and signature.html.
// Returns ErrTemplateSetNotFound if the template set does not exist.
// Returns ErrIncompleteTemplateSet if required templates are missing.
// Returns ErrInvalidAssetName if the name contains path separators or traversal.
func LoadTemplateSet(name string) (*TemplateSet, error) {
	return defaultLoader.LoadTemplateSet(name)
}
