package assets

// AssetLoader defines the contract for loading CSS styles and HTML templates.
// Implementations may load from embedded assets, filesystem, S3, database, etc.
type AssetLoader interface {
	// LoadStyle loads a CSS style by name (without .css extension).
	// Returns ErrStyleNotFound if the style doesn't exist.
	// Returns ErrInvalidAssetName if the name contains invalid characters.
	LoadStyle(name string) (string, error)

	// LoadTemplate loads an HTML template by name (without .html extension).
	// Returns ErrTemplateNotFound if the template doesn't exist.
	// Returns ErrInvalidAssetName if the name contains invalid characters.
	LoadTemplate(name string) (string, error)
}
