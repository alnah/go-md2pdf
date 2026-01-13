package assets

import (
	"embed"
	"fmt"
)

//go:embed styles/*
var styles embed.FS

//go:embed templates/*
var templates embed.FS

// EmbeddedLoader loads assets from embedded filesystem.
// Implements AssetLoader interface.
type EmbeddedLoader struct{}

// NewEmbeddedLoader creates an EmbeddedLoader.
func NewEmbeddedLoader() *EmbeddedLoader {
	return &EmbeddedLoader{}
}

// LoadStyle loads a CSS style from embedded assets by name.
// The name should not include the .css extension.
func (e *EmbeddedLoader) LoadStyle(name string) (string, error) {
	if err := ValidateAssetName(name); err != nil {
		return "", err
	}

	content, err := styles.ReadFile("styles/" + name + ".css")
	if err != nil {
		return "", fmt.Errorf("%w: %q", ErrStyleNotFound, name)
	}

	return string(content), nil
}

// LoadTemplate loads an HTML template from embedded assets by name.
// The name should not include the .html extension.
func (e *EmbeddedLoader) LoadTemplate(name string) (string, error) {
	if err := ValidateAssetName(name); err != nil {
		return "", err
	}

	content, err := templates.ReadFile("templates/" + name + ".html")
	if err != nil {
		return "", fmt.Errorf("%w: %q", ErrTemplateNotFound, name)
	}

	return string(content), nil
}

// Compile-time interface check.
var _ AssetLoader = (*EmbeddedLoader)(nil)
