package main

import (
	"os"
	"strings"
	"testing"
)

func TestValidateToPDFInputs(t *testing.T) {
	tests := []struct {
		name        string
		htmlContent string
		outputPath  string
		wantErr     error
	}{
		{
			name:        "valid inputs",
			htmlContent: "<html></html>",
			outputPath:  "/tmp/out.pdf",
			wantErr:     nil,
		},
		{
			name:        "empty HTML",
			htmlContent: "",
			outputPath:  "/tmp/out.pdf",
			wantErr:     ErrEmptyHTML,
		},
		{
			name:        "empty output path",
			htmlContent: "<html></html>",
			outputPath:  "",
			wantErr:     ErrEmptyOutputPath,
		},
		{
			name:        "both empty returns ErrEmptyHTML first",
			htmlContent: "",
			outputPath:  "",
			wantErr:     ErrEmptyHTML,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateToPDFInputs(tt.htmlContent, tt.outputPath)
			if err != tt.wantErr {
				t.Errorf("validateToPDFInputs() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestWriteHTMLToTempFile(t *testing.T) {
	content := "<html><body>Test Content</body></html>"

	path, cleanup, err := writeHTMLToTempFile(content)
	if err != nil {
		t.Fatalf("writeHTMLToTempFile() error = %v", err)
	}

	t.Run("file exists", func(t *testing.T) {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("temp file does not exist at %s", path)
		}
	})

	t.Run("file has .html extension pattern", func(t *testing.T) {
		if !strings.Contains(path, "go-md2pdf-") || !strings.HasSuffix(path, ".html") {
			t.Errorf("path %q does not match expected pattern go-md2pdf-*.html", path)
		}
	})

	t.Run("file contains expected content", func(t *testing.T) {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read temp file: %v", err)
		}
		if string(data) != content {
			t.Errorf("file content = %q, want %q", string(data), content)
		}
	})

	t.Run("cleanup removes file", func(t *testing.T) {
		cleanup()
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("temp file still exists after cleanup at %s", path)
		}
	})
}
