// Package main provides a CLI tool to convert Markdown files to HTML using Pandoc.
package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// Sentinel errors for path validation failures.
var (
	ErrEmptyPath        = errors.New("path cannot be empty")
	ErrInvalidExtension = errors.New("file must have .md or .markdown extension")
	ErrFileNotFound     = errors.New("file not found")
)

// HTMLConverter abstracts Markdown to HTML conversion to allow different backends.
type HTMLConverter interface {
	ToHTML(path string) (string, error)
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

// ToHTML converts a Markdown file to a standalone HTML5 document.
func (c *PandocConverter) ToHTML(path string) (string, error) {
	if err := validatePath(path); err != nil {
		return "", err
	}

	stdout, stderr, err := c.Runner.Run("pandoc", path, "-t", "html5", "--standalone")
	if err != nil {
		return "", fmt.Errorf("converting %q to HTML: %s: %w", path, stderr, err)
	}

	return stdout, nil
}

// validatePath ensures the path is non-empty, has a Markdown extension, and exists.
func validatePath(path string) error {
	if path == "" {
		return ErrEmptyPath
	}

	ext := filepath.Ext(path)
	if ext != ".md" && ext != ".markdown" {
		return fmt.Errorf("%w: got %q", ErrInvalidExtension, ext)
	}

	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %s", ErrFileNotFound, path)
		}
		return fmt.Errorf("checking file: %w", err)
	}

	return nil
}
