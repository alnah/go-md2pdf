package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type MockRenderer struct {
	Result     []byte
	Err        error
	CalledWith string
}

func (m *MockRenderer) RenderFromFile(filePath string) ([]byte, error) {
	m.CalledWith = filePath
	return m.Result, m.Err
}

func TestChromeConverter_ToPDF(t *testing.T) {
	tests := []struct {
		name       string
		html       string
		mock       *MockRenderer
		wantErr    error
		wantAnyErr bool
	}{
		{
			name: "successful render writes PDF",
			html: "<html><body>Test</body></html>",
			mock: &MockRenderer{
				Result: []byte("%PDF-1.4 fake pdf content"),
			},
		},
		{
			name: "renderer error propagates",
			html: "<html></html>",
			mock: &MockRenderer{
				Err: errors.New("chrome crashed"),
			},
			wantAnyErr: true,
		},
		{
			name: "empty HTML is valid",
			html: "",
			mock: &MockRenderer{
				Result: []byte("%PDF-1.4"),
			},
		},
		{
			name: "unicode content succeeds",
			html: "<html><body>Bonjour le monde</body></html>",
			mock: &MockRenderer{
				Result: []byte("%PDF-1.4 unicode"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			outputPath := filepath.Join(tmpDir, "output.pdf")

			converter := NewChromeConverterWith(tt.mock)
			err := converter.ToPDF(tt.html, outputPath)

			if tt.wantAnyErr || tt.wantErr != nil {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify PDF was written
			data, err := os.ReadFile(outputPath)
			if err != nil {
				t.Fatalf("failed to read output file: %v", err)
			}
			if string(data) != string(tt.mock.Result) {
				t.Errorf("expected output %q, got %q", tt.mock.Result, data)
			}

			// Verify renderer was called with temp file
			if !strings.Contains(tt.mock.CalledWith, "go-md2pdf-") {
				t.Errorf("expected temp file path with 'go-md2pdf-', got %q", tt.mock.CalledWith)
			}
		})
	}
}

func TestChromeConverter_ToPDF_WriteError(t *testing.T) {
	mock := &MockRenderer{
		Result: []byte("%PDF-1.4"),
	}

	converter := NewChromeConverterWith(mock)
	err := converter.ToPDF("<html></html>", "/nonexistent/directory/output.pdf")

	if !errors.Is(err, ErrWritePDF) {
		t.Errorf("expected ErrWritePDF, got %v", err)
	}
}

func TestNewChromeConverter(t *testing.T) {
	converter := NewChromeConverter()

	if converter.Renderer == nil {
		t.Fatal("expected non-nil Renderer")
	}

	// Verify it's a ChromeDPRenderer with correct timeout
	renderer, ok := converter.Renderer.(*ChromeDPRenderer)
	if !ok {
		t.Fatalf("expected *ChromeDPRenderer, got %T", converter.Renderer)
	}
	if renderer.Timeout != defaultTimeout {
		t.Errorf("expected timeout %v, got %v", defaultTimeout, renderer.Timeout)
	}
}

func TestNewChromeConverterWith(t *testing.T) {
	mock := &MockRenderer{}
	converter := NewChromeConverterWith(mock)

	if converter.Renderer != mock {
		t.Error("expected Renderer to be the provided mock")
	}
}

func TestNewChromeConverterWith_NilRenderer(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil renderer")
		}
	}()

	NewChromeConverterWith(nil)
}
