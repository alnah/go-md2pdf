package main

import (
	"path/filepath"
	"strings"
)

const (
	canonicalCLIName = "picoloom"
	legacyCLIName    = "md2pdf"
)

// displayCLIName returns the name to show in user-facing messages.
// Prefer the new canonical name, but preserve a known invoked alias when
// the process was launched through it.
func displayCLIName(argv0 string) string {
	base := filepath.Base(argv0)
	switch base {
	case canonicalCLIName, legacyCLIName:
		return base
	default:
		return canonicalCLIName
	}
}

func envCLIName(env *Environment) string {
	if env != nil && env.CLIName != "" {
		return env.CLIName
	}
	return canonicalCLIName
}

func canonicalizeShellFuncName(cliName string) string {
	return strings.ReplaceAll(cliName, "-", "_")
}
