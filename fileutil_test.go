package md2pdf

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateExtension(t *testing.T) {
	tests := []struct {
		name      string
		extension string
		wantErr   error
	}{
		{
			name:      "valid extension md",
			extension: "md",
			wantErr:   nil,
		},
		{
			name:      "valid extension html",
			extension: "html",
			wantErr:   nil,
		},
		{
			name:      "empty extension",
			extension: "",
			wantErr:   ErrExtensionEmpty,
		},
		{
			name:      "forward slash path traversal",
			extension: "../etc/passwd",
			wantErr:   ErrExtensionPathTraversal,
		},
		{
			name:      "backslash path traversal",
			extension: "..\\windows\\system32",
			wantErr:   ErrExtensionPathTraversal,
		},
		{
			name:      "null byte injection",
			extension: "html\x00exe",
			wantErr:   ErrExtensionPathTraversal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateExtension(tt.extension)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("validateExtension(%q) = %v, want %v", tt.extension, err, tt.wantErr)
			}
		})
	}
}

func TestWriteTempFile(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		extension string
	}{
		{
			name:      "markdown file",
			content:   "# Test Markdown",
			extension: "md",
		},
		{
			name:      "html file",
			content:   "<html><body>Test Content</body></html>",
			extension: "html",
		},
		{
			name:      "empty content",
			content:   "",
			extension: "md",
		},
		{
			name:      "unicode content",
			content:   "# Hello World\n\nThis is a test with special characters: café, naïve, résumé",
			extension: "md",
		},
		{
			name:      "unicode html content",
			content:   "<html><body>Hello World</body></html>",
			extension: "html",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, cleanup, err := writeTempFile(tt.content, tt.extension)
			if err != nil {
				t.Fatalf("writeTempFile() error = %v", err)
			}
			defer cleanup()

			// Verify file exists
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Errorf("temp file does not exist at %s", path)
			}

			// Verify path pattern
			if !strings.Contains(path, "md2pdf-") {
				t.Errorf("path %q does not contain prefix 'md2pdf-'", path)
			}
			if !strings.HasSuffix(path, "."+tt.extension) {
				t.Errorf("path %q does not have extension .%s", path, tt.extension)
			}

			// Verify content
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read temp file: %v", err)
			}
			if string(data) != tt.content {
				t.Errorf("file content = %q, want %q", string(data), tt.content)
			}
		})
	}
}

func TestWriteTempFile_Cleanup(t *testing.T) {
	path, cleanup, err := writeTempFile("test content", "md")
	if err != nil {
		t.Fatalf("writeTempFile() error = %v", err)
	}

	// Verify file exists before cleanup
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("temp file does not exist at %s", path)
	}

	// Call cleanup
	cleanup()

	// Verify file is removed after cleanup
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("temp file still exists after cleanup at %s", path)
	}
}

func TestWriteTempFile_InvalidExtension(t *testing.T) {
	tests := []struct {
		name      string
		extension string
		wantErr   error
	}{
		{
			name:      "empty extension",
			extension: "",
			wantErr:   ErrExtensionEmpty,
		},
		{
			name:      "path traversal",
			extension: "../foo",
			wantErr:   ErrExtensionPathTraversal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, cleanup, err := writeTempFile("content", tt.extension)
			if cleanup != nil {
				defer cleanup()
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("writeTempFile() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestWriteTempFile_CreateTempError(t *testing.T) {
	// Save original TMPDIR and restore after test
	originalTmpdir := os.Getenv("TMPDIR")
	defer func() {
		if originalTmpdir == "" {
			os.Unsetenv("TMPDIR")
		} else {
			os.Setenv("TMPDIR", originalTmpdir)
		}
	}()

	// Set TMPDIR to a non-existent directory to trigger CreateTemp failure
	os.Setenv("TMPDIR", "/nonexistent/path/that/does/not/exist")

	_, cleanup, err := writeTempFile("content", "md")
	if cleanup != nil {
		defer cleanup()
	}

	if err == nil {
		t.Fatal("writeTempFile() expected error when TMPDIR is invalid, got nil")
	}

	if !strings.Contains(err.Error(), "creating temp file") {
		t.Errorf("writeTempFile() error = %q, want error containing 'creating temp file'", err.Error())
	}
}

func TestFileExists(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create a test directory
	testDir := filepath.Join(tempDir, "testdir")
	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatalf("failed to create test dir: %v", err)
	}

	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "existing file returns true",
			path: testFile,
			want: true,
		},
		{
			name: "directory returns false",
			path: testDir,
			want: false,
		},
		{
			name: "nonexistent path returns false",
			path: filepath.Join(tempDir, "nonexistent"),
			want: false,
		},
		{
			name: "empty path returns false",
			path: "",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FileExists(tt.path)
			if got != tt.want {
				t.Errorf("FileExists(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestWriteTempFile_LargeContent(t *testing.T) {
	t.Parallel()

	// Test with large content to verify WriteString handles it correctly
	largeContent := strings.Repeat("x", 1024*1024) // 1MB

	path, cleanup, err := writeTempFile(largeContent, "txt")
	if err != nil {
		t.Fatalf("writeTempFile() error = %v", err)
	}
	defer cleanup()

	// Verify file contains all content
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile error = %v", err)
	}
	if len(data) != len(largeContent) {
		t.Errorf("file size = %d, want %d", len(data), len(largeContent))
	}
}

func TestIsFilePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "simple name returns false",
			input: "professional",
			want:  false,
		},
		{
			name:  "relative path with dot-slash returns true",
			input: "./custom.css",
			want:  true,
		},
		{
			name:  "parent path returns true",
			input: "../shared/style.css",
			want:  true,
		},
		{
			name:  "absolute Unix path returns true",
			input: "/absolute/path.css",
			want:  true,
		},
		{
			name:  "Windows path with backslash returns true",
			input: "C:\\windows\\path.css",
			want:  true,
		},
		{
			name:  "hyphenated name returns false",
			input: "my-style",
			want:  false,
		},
		{
			name:  "path with subdirectory returns true",
			input: "sub/dir",
			want:  true,
		},
		{
			name:  "empty string returns false",
			input: "",
			want:  false,
		},
		{
			name:  "name with dots but no slash returns false",
			input: "name.with.dots",
			want:  false,
		},
		{
			name:  "underscore name returns false",
			input: "my_style",
			want:  false,
		},
		{
			name:  "single forward slash returns true",
			input: "/",
			want:  true,
		},
		{
			name:  "single backslash returns true",
			input: "\\",
			want:  true,
		},
		{
			name:  "Windows drive letter path returns true",
			input: "D:/Documents/style.css",
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := IsFilePath(tt.input)
			if got != tt.want {
				t.Errorf("IsFilePath(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
