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

// run executes the main conversion pipeline.
// Accepts converters as interfaces to enable testing with mocks.
func run(args []string, preprocessor MarkdownPreprocessor, htmlConverter HTMLConverter, cssInjector CSSInjector, pdfConverter PDFConverter) error {
	if len(args) < minRequiredArgs {
		return ErrInvalidArgs
	}

	inputPath := args[inputFileArgIndex]
	outputPath := args[outputFileArgIndex]

	// Validate input file extension
	if err := validateMarkdownExtension(inputPath); err != nil {
		return err
	}

	// Read and preprocess Markdown
	mdContent, err := readMarkdownFile(inputPath)
	if err != nil {
		return err
	}
	mdContent = preprocessor.PreprocessMarkdown(mdContent)

	// Read optional CSS
	cssContent, err := readOptionalCSS(args)
	if err != nil {
		return err
	}

	// Convert Markdown to HTML
	htmlContent, err := htmlConverter.ToHTML(mdContent)
	if err != nil {
		return err
	}

	// Inject CSS into HTML
	htmlContent = cssInjector.InjectCSS(htmlContent, cssContent)

	// Convert HTML to PDF
	if err := pdfConverter.ToPDF(htmlContent, outputPath); err != nil {
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

// readOptionalCSS reads CSS content from the file path in args, if provided.
// Returns empty string if no CSS argument is present.
func readOptionalCSS(args []string) (string, error) {
	if len(args) <= cssFileArgIndex {
		return "", nil
	}

	cssBytes, err := os.ReadFile(args[cssFileArgIndex])
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrReadCSS, err)
	}

	return string(cssBytes), nil
}
