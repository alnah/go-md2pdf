// Package main provides a CLI tool to convert Markdown files to HTML using Pandoc.
package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// Sentinel errors for Pandoc conversion failures.
var (
	ErrEmptyContent = errors.New("markdown content cannot be empty")
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
// Uses -f markdown-fancy_lists to disable automatic conversion of letter markers
// (A), B), etc.) to numbered lists, preserving the original text.
func (c *PandocConverter) ToHTML(content string) (string, error) {
	if content == "" {
		return "", ErrEmptyContent
	}

	tmpPath, cleanup, err := writeTempMarkdown(content)
	if err != nil {
		return "", err
	}
	defer cleanup()

	stdout, stderr, err := c.Runner.Run("pandoc", tmpPath, "-f", "markdown-fancy_lists", "-t", "html5", "--standalone")
	if err != nil {
		return "", fmt.Errorf("converting to HTML: %s: %w", stderr, err)
	}

	return stdout, nil
}

// writeTempMarkdown creates a temporary file with Markdown content.
// Returns the file path and a cleanup function to remove the file.
func writeTempMarkdown(content string) (path string, cleanup func(), err error) {
	tmpFile, err := os.CreateTemp("", "go-md2pdf-*.md")
	if err != nil {
		return "", nil, fmt.Errorf("creating temp file: %w", err)
	}

	path = tmpFile.Name()
	cleanup = func() { _ = os.Remove(path) }

	if _, err := tmpFile.WriteString(content); err != nil {
		_ = tmpFile.Close()
		cleanup()
		return "", nil, fmt.Errorf("writing temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("closing temp file: %w", err)
	}

	return path, cleanup, nil
}
