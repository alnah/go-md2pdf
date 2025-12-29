// Package assets provides embedded CSS styles and HTML templates for PDF generation.
package assets

import (
	"embed"
	"errors"
	"fmt"
	"strings"
)

//go:embed styles/*
var styles embed.FS

//go:embed templates/*
var templates embed.FS

// Sentinel errors for assets operations.
var (
	ErrStyleNotFound       = errors.New("style not found")
	ErrInvalidStyleName    = errors.New("invalid style name")
	ErrTemplateNotFound    = errors.New("template not found")
	ErrInvalidTemplateName = errors.New("invalid template name")
)

// LoadStyle loads a CSS file from embedded assets by name.
// The name should not include the .css extension or path components.
// Returns ErrStyleNotFound if the style does not exist.
// Returns ErrInvalidStyleName if the name contains path separators or traversal.
func LoadStyle(name string) (string, error) {
	if err := validateStyleName(name); err != nil {
		return "", err
	}

	content, err := styles.ReadFile("styles/" + name + ".css")
	if err != nil {
		return "", fmt.Errorf("%w: %q", ErrStyleNotFound, name)
	}

	return string(content), nil
}

// validateStyleName checks that the style name is safe and doesn't contain
// path traversal or separator characters.
func validateStyleName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: empty name", ErrInvalidStyleName)
	}
	if strings.ContainsAny(name, "/\\.") {
		return fmt.Errorf("%w: %q", ErrInvalidStyleName, name)
	}
	return nil
}

// LoadTemplate loads an HTML template from embedded assets by name.
// The name should not include the .html extension or path components.
// Returns ErrTemplateNotFound if the template does not exist.
// Returns ErrInvalidTemplateName if the name contains path separators or traversal.
func LoadTemplate(name string) (string, error) {
	if err := validateTemplateName(name); err != nil {
		return "", err
	}

	content, err := templates.ReadFile("templates/" + name + ".html")
	if err != nil {
		return "", fmt.Errorf("%w: %q", ErrTemplateNotFound, name)
	}

	return string(content), nil
}

// validateTemplateName checks that the template name is safe and doesn't contain
// path traversal or separator characters.
func validateTemplateName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: empty name", ErrInvalidTemplateName)
	}
	if strings.ContainsAny(name, "/\\.") {
		return fmt.Errorf("%w: %q", ErrInvalidTemplateName, name)
	}
	return nil
}
