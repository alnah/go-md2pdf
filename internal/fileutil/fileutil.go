// Package fileutil provides file and path utility functions.
package fileutil

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// Sentinel errors for file utility operations.
var (
	ErrExtensionEmpty         = errors.New("extension cannot be empty")
	ErrExtensionPathTraversal = errors.New("extension contains path separator or null byte")
)

// WriteTempFile creates a temporary file with the given content and extension.
// Returns the file path and a cleanup function to remove the file.
func WriteTempFile(content, extension string) (path string, cleanup func(), err error) {
	if err := ValidateExtension(extension); err != nil {
		return "", nil, err
	}

	tmpFile, err := os.CreateTemp("", "md2pdf-*."+extension)
	if err != nil {
		return "", nil, fmt.Errorf("creating temp file: %w", err)
	}

	path = tmpFile.Name()
	cleanup = func() { _ = os.Remove(path) }

	if _, writeErr := tmpFile.WriteString(content); writeErr != nil {
		_ = tmpFile.Close()
		cleanup()
		return "", nil, fmt.Errorf("writing temp file: %w", writeErr)
	}

	if closeErr := tmpFile.Close(); closeErr != nil {
		cleanup()
		return "", nil, fmt.Errorf("closing temp file: %w", closeErr)
	}

	return path, cleanup, nil
}

// ValidateExtension checks that the extension is safe for use in temp file names.
func ValidateExtension(extension string) error {
	if extension == "" {
		return ErrExtensionEmpty
	}
	if strings.ContainsAny(extension, "/\\\x00") {
		return ErrExtensionPathTraversal
	}
	return nil
}

// FileExists returns true if the path exists and is a regular file.
func FileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// IsFilePath returns true if the string looks like a file path rather than a name.
// A string containing path separators (/, \) is treated as a path.
//
// Examples:
//   - "professional" -> false (name)
//   - "./custom.css" -> true (relative path)
//   - "../shared/style.css" -> true (parent path)
//   - "/absolute/path.css" -> true (absolute)
//   - "C:\windows\path.css" -> true (Windows)
//   - "my-style" -> false (hyphenated name)
//   - "sub/dir" -> true (contains separator)
func IsFilePath(s string) bool {
	return strings.ContainsAny(s, "/\\")
}

// IsURL returns true if the string looks like a URL.
func IsURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}
