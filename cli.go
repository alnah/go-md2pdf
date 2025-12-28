package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Sentinel errors for CLI operations.
var (
	ErrInvalidArgs      = errors.New("usage: go-md2pdf <input.md> <output.pdf> [style.css]")
	ErrReadCSS          = errors.New("failed to read CSS file")
	ErrReadMarkdown     = errors.New("failed to read markdown file")
	ErrInvalidExtension = errors.New("file must have .md or .markdown extension")
)

// CLI argument positions.
const (
	minRequiredArgs    = 3
	inputFileArgIndex  = 1
	outputFileArgIndex = 2
	cssFileArgIndex    = 3
)

// Converter is the interface for the conversion service.
type Converter interface {
	Convert(opts ConversionOptions) error
}

// run parses arguments, reads files, and delegates to the conversion service.
func run(args []string, service Converter) error {
	if len(args) < minRequiredArgs {
		return ErrInvalidArgs
	}

	inputPath := args[inputFileArgIndex]
	outputPath := args[outputFileArgIndex]

	// Validate input file extension
	if err := validateMarkdownExtension(inputPath); err != nil {
		return err
	}

	// Read Markdown file
	mdContent, err := readMarkdownFile(inputPath)
	if err != nil {
		return err
	}

	// Read optional CSS
	cssContent, err := resolveStyleArgs(args)
	if err != nil {
		return err
	}

	// Build options and delegate to service
	opts := ConversionOptions{
		MarkdownContent: mdContent,
		OutputPath:      outputPath,
		CSSContent:      cssContent,
	}

	if err := service.Convert(opts); err != nil {
		return err
	}

	fmt.Printf("Created %s\n", outputPath)
	return nil
}

// validateMarkdownExtension checks that the file has a .md or .markdown extension.
func validateMarkdownExtension(path string) error {
	ext := filepath.Ext(path)
	if ext != ".md" && ext != ".markdown" {
		return fmt.Errorf("%w: got %q", ErrInvalidExtension, ext)
	}
	return nil
}

// readMarkdownFile reads the content of a Markdown file.
func readMarkdownFile(path string) (string, error) {
	content, err := os.ReadFile(path) // #nosec G304 -- TODO: add path sanitization when implementing the true CLI
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrReadMarkdown, err)
	}
	return string(content), nil
}

// resolveStyleArgs parses CSS-related arguments.
// Returns (cssContent, error).
// - If CSS file is provided: returns (content, nil)
// - Otherwise: returns ("", nil) for no CSS
func resolveStyleArgs(args []string) (string, error) {
	if len(args) <= cssFileArgIndex {
		return "", nil
	}

	cssBytes, err := os.ReadFile(args[cssFileArgIndex]) // #nosec G304 -- TODO: add path sanitization
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrReadCSS, err)
	}

	return string(cssBytes), nil
}
