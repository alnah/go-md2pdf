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

func TestEmbeddedLoader_LoadTemplateSet(t *testing.T) {
	t.Parallel()

	loader := NewEmbeddedLoader()

	tests := []struct {
		name    string
		setName string
		wantErr error
	}{
		{
			name:    "loads default template set",
			setName: "default",
			wantErr: nil,
		},
		{
			name:    "returns ErrTemplateSetNotFound for nonexistent",
			setName: "nonexistent-template-xyz",
			wantErr: ErrTemplateSetNotFound,
		},
		{
			name:    "returns ErrInvalidAssetName for empty name",
			setName: "",
			wantErr: ErrInvalidAssetName,
		},
		{
			name:    "returns ErrInvalidAssetName for path traversal",
			setName: "../secret",
			wantErr: ErrInvalidAssetName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ts, err := loader.LoadTemplateSet(tt.setName)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("LoadTemplateSet(%q) error = %v, want %v", tt.setName, err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("LoadTemplateSet(%q) unexpected error: %v", tt.setName, err)
			}

			if !strings.Contains(ts.Cover, "cover") {
				t.Errorf("LoadTemplateSet(%q) cover should contain 'cover'", tt.setName)
			}
			if !strings.Contains(ts.Signature, "signature") {
				t.Errorf("LoadTemplateSet(%q) signature should contain 'signature'", tt.setName)
			}
		})
	}
}

func TestEmbeddedLoader_ImplementsAssetLoader(t *testing.T) {
	t.Parallel()

	var _ AssetLoader = (*EmbeddedLoader)(nil)
}
