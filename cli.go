package main

import (
	"errors"
	"fmt"
	"os"
)

// Sentinel errors for CLI operations.
var (
	ErrInvalidArgs = errors.New("usage: md2pdf <input.md> <output.pdf> [style.css]")
	ErrReadCSS     = errors.New("failed to read CSS file")
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
func run(args []string, htmlConverter HTMLConverter, pdfConverter PDFConverter) error {
	if len(args) < minRequiredArgs {
		return ErrInvalidArgs
	}

	inputPath := args[inputFileArgIndex]
	outputPath := args[outputFileArgIndex]

	cssContent, err := readOptionalCSS(args)
	if err != nil {
		return err
	}

	htmlContent, err := htmlConverter.ToHTML(inputPath)
	if err != nil {
		return err
	}

	if err := pdfConverter.ToPDF(htmlContent, cssContent, outputPath); err != nil {
		return err
	}

	fmt.Printf("Created %s\n", outputPath)
	return nil
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
