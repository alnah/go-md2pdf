package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrintUsage(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	printUsage(&buf)
	output := buf.String()

	requiredStrings := []string{
		"Usage: md2pdf",
		"Commands:",
		"convert",
		"version",
		"help",
	}

	for _, s := range requiredStrings {
		if !strings.Contains(output, s) {
			t.Errorf("printUsage output should contain %q", s)
		}
	}
}

func TestPrintConvertUsage(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	printConvertUsage(&buf)
	output := buf.String()

	// Check for flag group headers
	flagGroups := []string{
		"Author:",
		"Document:",
		"Page:",
		"Footer:",
		"Cover:",
		"Signature:",
		"Table of Contents:",
		"Watermark:",
		"Page Breaks:",
		"Styling:",
	}

	for _, group := range flagGroups {
		if !strings.Contains(output, group) {
			t.Errorf("printConvertUsage output should contain group header %q", group)
		}
	}

	// Check for new author flags
	authorFlags := []string{
		"--author-name",
		"--author-title",
		"--author-email",
		"--author-org",
	}

	for _, flag := range authorFlags {
		if !strings.Contains(output, flag) {
			t.Errorf("printConvertUsage output should contain %q", flag)
		}
	}

	// Check for new document flags
	documentFlags := []string{
		"--doc-title",
		"--doc-subtitle",
		"--doc-version",
		"--doc-date",
	}

	for _, flag := range documentFlags {
		if !strings.Contains(output, flag) {
			t.Errorf("printConvertUsage output should contain %q", flag)
		}
	}

	// Check for watermark shorthand flags
	wmFlags := []string{
		"--wm-text",
		"--wm-color",
		"--wm-opacity",
		"--wm-angle",
	}

	for _, flag := range wmFlags {
		if !strings.Contains(output, flag) {
			t.Errorf("printConvertUsage output should contain %q", flag)
		}
	}
}

func TestRunHelp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		args         []string
		wantInStdout []string
		wantInStderr []string
	}{
		{
			name:         "no args shows main usage",
			args:         []string{},
			wantInStdout: []string{"Usage: md2pdf", "Commands:"},
		},
		{
			name:         "convert shows convert help",
			args:         []string{"convert"},
			wantInStdout: []string{"Usage: md2pdf convert", "Author:", "Document:"},
		},
		{
			name:         "version shows version help",
			args:         []string{"version"},
			wantInStdout: []string{"Usage: md2pdf version"},
		},
		{
			name:         "help shows help help",
			args:         []string{"help"},
			wantInStdout: []string{"Usage: md2pdf help"},
		},
		{
			name:         "unknown command shows error",
			args:         []string{"unknown"},
			wantInStderr: []string{"Unknown command: unknown"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stdout, stderr bytes.Buffer
			deps := &Dependencies{
				Stdout: &stdout,
				Stderr: &stderr,
			}

			runHelp(tt.args, deps)

			stdoutStr := stdout.String()
			stderrStr := stderr.String()

			for _, want := range tt.wantInStdout {
				if !strings.Contains(stdoutStr, want) {
					t.Errorf("stdout should contain %q, got %q", want, stdoutStr)
				}
			}

			for _, want := range tt.wantInStderr {
				if !strings.Contains(stderrStr, want) {
					t.Errorf("stderr should contain %q, got %q", want, stderrStr)
				}
			}
		})
	}
}
