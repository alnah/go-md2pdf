package md2pdf

import (
	"errors"

	"github.com/alnah/go-md2pdf/internal/assets"
)

// Asset name constants for built-in styles and templates.
const (
	// DefaultStyle is the name of the built-in CSS style.
	DefaultStyle = "default"

	// DefaultTemplateSet is the name of the built-in template set.
	DefaultTemplateSet = "default"
)

// AssetLoader defines the contract for loading CSS styles and HTML templates.
// Implementations may load from filesystem, embedded assets, S3, database, etc.
//
// The library provides NewAssetLoader() for filesystem-based loading with
// fallback to embedded defaults. Implement this interface for custom backends.
type AssetLoader interface {
	// LoadStyle loads a CSS style by name (without .css extension).
	// Returns ErrStyleNotFound if the style doesn't exist.
	LoadStyle(name string) (string, error)

	// LoadTemplateSet loads cover and signature templates by name.
	// Returns ErrTemplateSetNotFound if the template set doesn't exist.
	// Returns ErrIncompleteTemplateSet if required templates are missing.
	LoadTemplateSet(name string) (*TemplateSet, error)
}

// TemplateSet holds HTML templates for document generation.
// A template set contains cover and signature templates that work together.
type TemplateSet struct {
	Name      string // Identifier (name or path)
	Cover     string // Cover page template HTML
	Signature string // Signature block template HTML
}

// NewTemplateSet creates a TemplateSet from cover and signature HTML content.
// This is a convenience constructor for users providing templates directly.
func NewTemplateSet(name, cover, signature string) *TemplateSet {
	return &TemplateSet{
		Name:      name,
		Cover:     cover,
		Signature: signature,
	}
}

// NewAssetLoader creates an AssetLoader for the given base path.
// If basePath is empty, returns a loader using only embedded assets.
// If basePath is set, custom assets take precedence with fallback to embedded.
//
// The basePath directory should contain:
//   - styles/{name}.css for CSS styles
//   - templates/{name}/cover.html and signature.html for template sets
//
// Returns ErrInvalidAssetPath if basePath is set but not a valid, readable directory.
func NewAssetLoader(basePath string) (AssetLoader, error) {
	resolver, err := assets.NewAssetResolver(basePath)
	if err != nil {
		return nil, convertAssetError(err)
	}
	return &assetLoaderAdapter{resolver: resolver}, nil
}

// assetLoaderAdapter wraps internal AssetResolver to return public types.
type assetLoaderAdapter struct {
	resolver *assets.AssetResolver
}

func (a *assetLoaderAdapter) LoadStyle(name string) (string, error) {
	content, err := a.resolver.LoadStyle(name)
	if err != nil {
		return "", convertAssetError(err)
	}
	return content, nil
}

func (a *assetLoaderAdapter) LoadTemplateSet(name string) (*TemplateSet, error) {
	ts, err := a.resolver.LoadTemplateSet(name)
	if err != nil {
		return nil, convertAssetError(err)
	}
	return &TemplateSet{
		Name:      ts.Name,
		Cover:     ts.Cover,
		Signature: ts.Signature,
	}, nil
}

// convertAssetError maps internal asset errors to public errors.
func convertAssetError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case isError(err, assets.ErrStyleNotFound):
		return wrapError(ErrStyleNotFound, err)
	case isError(err, assets.ErrTemplateSetNotFound):
		return wrapError(ErrTemplateSetNotFound, err)
	case isError(err, assets.ErrIncompleteTemplateSet):
		return wrapError(ErrIncompleteTemplateSet, err)
	case isError(err, assets.ErrInvalidBasePath):
		return wrapError(ErrInvalidAssetPath, err)
	case isError(err, assets.ErrPathTraversal):
		return wrapError(ErrInvalidAssetPath, err)
	case isError(err, assets.ErrInvalidAssetName):
		return wrapError(ErrStyleNotFound, err) // Invalid name means not found
	default:
		return err
	}
}

// isError checks if err wraps or equals target using errors.Is semantics.
func isError(err, target error) bool {
	return errors.Is(err, target)
}

// wrapError creates a new error that wraps the original with a public sentinel.
// The resulting error preserves the original message via Error() and supports
// errors.Is() matching against the public sentinel via Unwrap().
func wrapError(sentinel, original error) error {
	return &wrappedAssetError{sentinel: sentinel, original: original}
}

type wrappedAssetError struct {
	sentinel error
	original error
}

func (e *wrappedAssetError) Error() string {
	return e.original.Error()
}

// Unwrap returns the public sentinel for errors.Is() matching.
// Internal errors are not exposed since they're in internal/ packages.
func (e *wrappedAssetError) Unwrap() error {
	return e.sentinel
}
