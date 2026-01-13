package assets

// AssetLoader defines the contract for loading CSS styles and HTML templates.
// Implementations may load from embedded assets, filesystem, S3, database, etc.
type AssetLoader interface {
	// LoadStyle loads a CSS style by name (without .css extension).
	// Returns ErrStyleNotFound if the style doesn't exist.
	// Returns ErrInvalidAssetName if the name contains invalid characters.
	LoadStyle(name string) (string, error)

	// LoadTemplateSet loads a set of HTML templates by name.
	// A template set contains cover.html and signature.html in a named directory.
	// Returns ErrTemplateSetNotFound if the template set doesn't exist.
	// Returns ErrIncompleteTemplateSet if required templates are missing.
	// Returns ErrInvalidAssetName if the name contains invalid characters.
	LoadTemplateSet(name string) (*TemplateSet, error)
}
