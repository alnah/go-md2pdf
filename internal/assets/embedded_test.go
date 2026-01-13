package assets

import (
	"errors"
	"strings"
	"testing"
)

func TestNewEmbeddedLoader(t *testing.T) {
	t.Parallel()

	loader := NewEmbeddedLoader()
	if loader == nil {
		t.Fatal("NewEmbeddedLoader() returned nil")
	}
}

func TestEmbeddedLoader_LoadStyle(t *testing.T) {
	t.Parallel()

	loader := NewEmbeddedLoader()

	tests := []struct {
		name        string
		styleName   string
		wantErr     error
		wantContain string
	}{
		{
			name:        "loads creative style",
			styleName:   "creative",
			wantErr:     nil,
			wantContain: "font-family",
		},
		{
			name:      "returns ErrStyleNotFound for nonexistent",
			styleName: "nonexistent-style-xyz",
			wantErr:   ErrStyleNotFound,
		},
		{
			name:      "returns ErrInvalidAssetName for empty name",
			styleName: "",
			wantErr:   ErrInvalidAssetName,
		},
		{
			name:      "returns ErrInvalidAssetName for path traversal",
			styleName: "../secret",
			wantErr:   ErrInvalidAssetName,
		},
		{
			name:      "returns ErrInvalidAssetName for backslash traversal",
			styleName: "..\\secret",
			wantErr:   ErrInvalidAssetName,
		},
		{
			name:      "returns ErrInvalidAssetName for name with dot",
			styleName: "style.name",
			wantErr:   ErrInvalidAssetName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := loader.LoadStyle(tt.styleName)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("LoadStyle(%q) error = %v, want %v", tt.styleName, err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("LoadStyle(%q) unexpected error: %v", tt.styleName, err)
			}

			if tt.wantContain != "" && !strings.Contains(got, tt.wantContain) {
				t.Errorf("LoadStyle(%q) content should contain %q", tt.styleName, tt.wantContain)
			}
		})
	}
}

func TestEmbeddedLoader_LoadTemplate(t *testing.T) {
	t.Parallel()

	loader := NewEmbeddedLoader()

	tests := []struct {
		name         string
		templateName string
		wantErr      error
		wantContain  string
	}{
		{
			name:         "loads cover template",
			templateName: "cover",
			wantErr:      nil,
			wantContain:  "cover",
		},
		{
			name:         "loads signature template",
			templateName: "signature",
			wantErr:      nil,
			wantContain:  "signature",
		},
		{
			name:         "returns ErrTemplateNotFound for nonexistent",
			templateName: "nonexistent-template-xyz",
			wantErr:      ErrTemplateNotFound,
		},
		{
			name:         "returns ErrInvalidAssetName for empty name",
			templateName: "",
			wantErr:      ErrInvalidAssetName,
		},
		{
			name:         "returns ErrInvalidAssetName for path traversal",
			templateName: "../secret",
			wantErr:      ErrInvalidAssetName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := loader.LoadTemplate(tt.templateName)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("LoadTemplate(%q) error = %v, want %v", tt.templateName, err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("LoadTemplate(%q) unexpected error: %v", tt.templateName, err)
			}

			if tt.wantContain != "" && !strings.Contains(got, tt.wantContain) {
				t.Errorf("LoadTemplate(%q) content should contain %q", tt.templateName, tt.wantContain)
			}
		})
	}
}

func TestEmbeddedLoader_ImplementsAssetLoader(t *testing.T) {
	t.Parallel()

	var _ AssetLoader = (*EmbeddedLoader)(nil)
}
