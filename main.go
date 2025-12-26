package main

import (
	"fmt"
	"os"
)

// Exit codes and CLI argument positions.
const (
	exitSuccess       = 0
	exitError         = 1
	minRequiredArgs   = 2
	inputFileArgIndex = 1
)

func main() {
	if len(os.Args) < minRequiredArgs {
		fmt.Fprintln(os.Stderr, "usage: md2pdf <file.md>")
		os.Exit(exitError)
	}

	converter := NewPandocConverter()
	html, err := converter.ToHTML(os.Args[inputFileArgIndex])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitError)
	}

	fmt.Print(html)
}
