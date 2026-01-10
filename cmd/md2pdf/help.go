package main

import (
	"fmt"
	"io"
)

// printUsage prints the main usage message.
func printUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: md2pdf <command> [flags] [args]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  convert    Convert markdown files to PDF")
	fmt.Fprintln(w, "  version    Show version information")
	fmt.Fprintln(w, "  help       Show help for a command")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Run 'md2pdf help <command>' for details on a specific command.")
}

// printConvertUsage prints usage for the convert command.
func printConvertUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: md2pdf convert <input> [flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Convert markdown files to PDF.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Arguments:")
	fmt.Fprintln(w, "  input    Markdown file or directory (optional if config has input.defaultDir)")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Input/Output:")
	fmt.Fprintln(w, "  -o, --output <path>       Output file or directory")
	fmt.Fprintln(w, "  -c, --config <name>       Config file name or path")
	fmt.Fprintln(w, "  -w, --workers <n>         Parallel workers (0 = auto)")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Author:")
	fmt.Fprintln(w, "      --author-name <s>     Author name")
	fmt.Fprintln(w, "      --author-title <s>    Author professional title")
	fmt.Fprintln(w, "      --author-email <s>    Author email")
	fmt.Fprintln(w, "      --author-org <s>      Organization name")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Document:")
	fmt.Fprintln(w, "      --doc-title <s>       Document title (\"\" = auto from H1)")
	fmt.Fprintln(w, "      --doc-subtitle <s>    Document subtitle")
	fmt.Fprintln(w, "      --doc-version <s>     Version string")
	fmt.Fprintln(w, "      --doc-date <s>        Date: \"auto\", \"auto:FORMAT\", or literal")
	fmt.Fprintln(w, "                            Tokens: YYYY, YY, MMMM, MMM, MM, M, DD, D")
	fmt.Fprintln(w, "                            Presets (case-insensitive): iso, european, us, long")
	fmt.Fprintln(w, "                            Use [text] to escape literals: [Date]: YYYY")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Page:")
	fmt.Fprintln(w, "  -p, --page-size <s>       Page size: letter, a4, legal")
	fmt.Fprintln(w, "      --orientation <s>     Orientation: portrait, landscape")
	fmt.Fprintln(w, "      --margin <f>          Margin in inches (0.25-3.0)")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Footer:")
	fmt.Fprintln(w, "      --footer-position <s> Position: left, center, right")
	fmt.Fprintln(w, "      --footer-text <s>     Custom footer text")
	fmt.Fprintln(w, "      --footer-page-number  Show page numbers")
	fmt.Fprintln(w, "      --no-footer           Disable footer")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Cover:")
	fmt.Fprintln(w, "      --cover-logo <path>   Logo path or URL")
	fmt.Fprintln(w, "      --no-cover            Disable cover page")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Signature:")
	fmt.Fprintln(w, "      --sig-image <path>    Signature image path")
	fmt.Fprintln(w, "      --no-signature        Disable signature block")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Table of Contents:")
	fmt.Fprintln(w, "      --toc-title <s>       TOC heading text")
	fmt.Fprintln(w, "      --toc-depth <n>       Max heading depth (1-6)")
	fmt.Fprintln(w, "      --no-toc              Disable table of contents")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Watermark:")
	fmt.Fprintln(w, "      --wm-text <s>         Watermark text")
	fmt.Fprintln(w, "      --wm-color <s>        Watermark color (hex)")
	fmt.Fprintln(w, "      --wm-opacity <f>      Watermark opacity (0.0-1.0)")
	fmt.Fprintln(w, "      --wm-angle <f>        Watermark angle in degrees")
	fmt.Fprintln(w, "      --no-watermark        Disable watermark")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Page Breaks:")
	fmt.Fprintln(w, "      --break-before <s>    Break before headings: h1,h2,h3")
	fmt.Fprintln(w, "      --orphans <n>         Min lines at page bottom (1-5)")
	fmt.Fprintln(w, "      --widows <n>          Min lines at page top (1-5)")
	fmt.Fprintln(w, "      --no-page-breaks      Disable page break features")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Styling:")
	fmt.Fprintln(w, "      --css <path>          External CSS file")
	fmt.Fprintln(w, "      --no-style            Disable CSS styling")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Output Control:")
	fmt.Fprintln(w, "  -q, --quiet               Only show errors")
	fmt.Fprintln(w, "  -v, --verbose             Show detailed timing")
}

// runHelp prints help for a specific command.
func runHelp(args []string, deps *Dependencies) {
	if len(args) == 0 {
		printUsage(deps.Stdout)
		return
	}

	switch args[0] {
	case "convert":
		printConvertUsage(deps.Stdout)
	case "version":
		fmt.Fprintln(deps.Stdout, "Usage: md2pdf version")
		fmt.Fprintln(deps.Stdout)
		fmt.Fprintln(deps.Stdout, "Show version information.")
	case "help":
		fmt.Fprintln(deps.Stdout, "Usage: md2pdf help [command]")
		fmt.Fprintln(deps.Stdout)
		fmt.Fprintln(deps.Stdout, "Show help for a command.")
	default:
		fmt.Fprintf(deps.Stderr, "Unknown command: %s\n", args[0])
		printUsage(deps.Stderr)
	}
}
