// Package assets provides embedded CSS styles for PDF generation.
package assets

import (
	"embed"
	"errors"
	"fmt"
	"strings"
)

//go:embed styles/*
var styles embed.FS

// Sentinel errors for assets operations.
var (
	ErrStyleNotFound    = errors.New("style not found")
	ErrInvalidStyleName = errors.New("invalid style name")
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
