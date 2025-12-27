package main

import (
	"fmt"
	"os"
)

// Exit codes and CLI argument positions.
const (
	exitSuccess        = 0
	exitError          = 1
	minRequiredArgs    = 3
	inputFileArgIndex  = 1
	outputFileArgIndex = 2
	cssFileArgIndex    = 3
)

func main() {
	if len(os.Args) < minRequiredArgs {
		fmt.Fprintln(os.Stderr, "usage: md2pdf <input.md> <output.pdf> [style.css]")
		os.Exit(exitError)
	}

	inputPath := os.Args[inputFileArgIndex]
	outputPath := os.Args[outputFileArgIndex]

	// Convert Markdown to HTML
	pandoc := NewPandocConverter()
	html, err := pandoc.ToHTML(inputPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitError)
	}

	// Read optional CSS file
	var cssContent string
	if len(os.Args) > cssFileArgIndex {
		cssBytes, err := os.ReadFile(os.Args[cssFileArgIndex])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(exitError)
		}
		cssContent = string(cssBytes)
	}

	// Convert HTML to PDF
	chrome := NewChromeConverter()
	if err := chrome.ToPDF(html, cssContent, outputPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitError)
	}

	fmt.Printf("Created %s\n", outputPath)
}
