package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Sentinel errors for CLI operations.
var (
	ErrInvalidArgs      = errors.New("usage: go-md2pdf <input.md> <output.pdf> [style.css] [--no-style]")
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

	// Read optional CSS and determine NoStyle
	cssContent, noStyle, err := resolveStyleArgs(args)
	if err != nil {
		return err
	}

	// Build options and delegate to service
	opts := ConversionOptions{
		MarkdownContent: mdContent,
		OutputPath:      outputPath,
		CSSContent:      cssContent,
		NoStyle:         noStyle,
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
// Returns (cssContent, noStyle, error).
// - If --no-style is present: returns ("", true, nil)
// - If CSS file is provided: returns (content, false, nil)
// - Otherwise: returns ("", false, nil) and service will use default
func resolveStyleArgs(args []string) (string, bool, error) {
	if hasNoStyleFlag(args) {
		return "", true, nil
	}

	if len(args) <= cssFileArgIndex {
		return "", false, nil
	}

	cssBytes, err := os.ReadFile(args[cssFileArgIndex]) // #nosec G304 -- TODO: add path sanitization
	if err != nil {
		return "", false, fmt.Errorf("%w: %v", ErrReadCSS, err)
	}

	return string(cssBytes), false, nil
}

// hasNoStyleFlag checks if --no-style flag is present in args.
func hasNoStyleFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--no-style" {
			return true
		}
	}
	return false
}
