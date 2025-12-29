package main

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
)

// HTMLConverter abstracts Markdown to HTML conversion to allow different backends.
type HTMLConverter interface {
	ToHTML(content string) (string, error)
}

// CommandRunner abstracts command execution to enable testing without real subprocesses.
type CommandRunner interface {
	Run(name string, args ...string) (stdout string, stderr string, err error)
}

// ExecRunner implements CommandRunner using os/exec.
type ExecRunner struct{}

func (r *ExecRunner) Run(name string, args ...string) (string, string, error) {
	cmd := exec.Command(name, args...)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return "", "", fmt.Errorf("creating stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", "", fmt.Errorf("starting command: %w", err)
	}

	stderrContent, err := io.ReadAll(stderrPipe)
	if err != nil {
		return "", "", fmt.Errorf("reading stderr: %w", err)
	}

	err = cmd.Wait()
	return stdout.String(), string(stderrContent), err
}

// PandocConverter converts Markdown to HTML by invoking the Pandoc CLI.
type PandocConverter struct {
	Runner CommandRunner
}

// NewPandocConverter creates a PandocConverter with a real command runner.
func NewPandocConverter() *PandocConverter {
	return &PandocConverter{Runner: &ExecRunner{}}
}

// ToHTML converts Markdown content to a standalone HTML5 document using Pandoc.
// Uses -f markdown-fancy_lists+hard_line_breaks to:
// - Disable automatic conversion of letter markers (A), B), etc.) to numbered lists
// - Treat single newlines as hard line breaks (<br>)
func (c *PandocConverter) ToHTML(content string) (string, error) {
	tmpPath, cleanup, err := writeTempFile(content, "md")
	if err != nil {
		return "", err
	}
	defer cleanup()

	stdout, stderr, err := c.Runner.Run("pandoc", tmpPath, "-f", "markdown-fancy_lists+hard_line_breaks", "-t", "html5", "--standalone")
	if err != nil {
		if stderr != "" {
			return "", fmt.Errorf("converting to HTML: %s: %w", stderr, err)
		}
		return "", fmt.Errorf("converting to HTML: %w", err)
	}

	return stdout, nil
}
