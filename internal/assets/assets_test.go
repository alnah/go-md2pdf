package assets

import (
	"errors"
	"strings"
	"testing"
)

func TestLoadStyle(t *testing.T) {
	tests := []struct {
		name      string
		styleName string
		wantErr   error
	}{
		{
			name:      "valid style returns content",
			styleName: "creative",
			wantErr:   nil,
		},
		{
			name:      "default style returns content",
			styleName: DefaultStyleName,
			wantErr:   nil,
		},
		{
			name:      "nonexistent style returns ErrStyleNotFound",
			styleName: "nonexistent",
			wantErr:   ErrStyleNotFound,
		},
		{
			name:      "empty name returns ErrInvalidAssetName",
			styleName: "",
			wantErr:   ErrInvalidAssetName,
		},
		{
			name:      "path traversal with slash returns ErrInvalidAssetName",
			styleName: "../secret",
			wantErr:   ErrInvalidAssetName,
		},
		{
			name:      "path traversal with backslash returns ErrInvalidAssetName",
			styleName: "..\\secret",
			wantErr:   ErrInvalidAssetName,
		},
		{
			name:      "path with dot returns ErrInvalidAssetName",
			styleName: "style.name",
			wantErr:   ErrInvalidAssetName,
		},
		{
			name:      "absolute path returns ErrInvalidAssetName",
			styleName: "/etc/passwd",
			wantErr:   ErrInvalidAssetName,
		},
		{
			name:      "valid name with hyphen",
			styleName: "my-style",
			wantErr:   ErrStyleNotFound, // valid name but doesn't exist
		},
		{
			name:      "valid name with underscore",
			styleName: "my_style",
			wantErr:   ErrStyleNotFound, // valid name but doesn't exist
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := LoadStyle(tt.styleName)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("LoadStyle(%q) error = %v, want %v", tt.styleName, err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("LoadStyle(%q) unexpected error: %v", tt.styleName, err)
			}

			if content == "" {
				t.Errorf("LoadStyle(%q) returned empty content", tt.styleName)
			}
		})
	}
}

func TestLoadTemplateSet(t *testing.T) {
	tests := []struct {
		name    string
		setName string
		wantErr error
	}{
		{
			name:    "valid template set returns content",
			setName: "default",
			wantErr: nil,
		},
		{
			name:    "nonexistent template set returns ErrTemplateSetNotFound",
			setName: "nonexistent",
			wantErr: ErrTemplateSetNotFound,
		},
		{
			name:    "empty name returns ErrInvalidAssetName",
			setName: "",
			wantErr: ErrInvalidAssetName,
		},
		{
			name:    "path traversal with slash returns ErrInvalidAssetName",
			setName: "../secret",
			wantErr: ErrInvalidAssetName,
		},
		{
			name:    "path traversal with backslash returns ErrInvalidAssetName",
			setName: "..\\secret",
			wantErr: ErrInvalidAssetName,
		},
		{
			name:    "path with dot returns ErrInvalidAssetName",
			setName: "template.name",
			wantErr: ErrInvalidAssetName,
		},
		{
			name:    "absolute path returns ErrInvalidAssetName",
			setName: "/etc/passwd",
			wantErr: ErrInvalidAssetName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, err := LoadTemplateSet(tt.setName)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("LoadTemplateSet(%q) error = %v, want %v", tt.setName, err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("LoadTemplateSet(%q) unexpected error: %v", tt.setName, err)
			}

			if ts.Cover == "" {
				t.Errorf("LoadTemplateSet(%q) returned empty cover", tt.setName)
			}
			if ts.Signature == "" {
				t.Errorf("LoadTemplateSet(%q) returned empty signature", tt.setName)
			}
		})
	}
}

func TestLoadTemplateSet_SignatureContent(t *testing.T) {
	ts, err := LoadTemplateSet("default")
	if err != nil {
		t.Fatalf("LoadTemplateSet(default) error: %v", err)
	}

	// Verify signature template contains expected Go template structure
	expectedParts := []string{
		"signature-block",
		"{{.Name}}",
		"{{.Email}}",
	}

	for _, part := range expectedParts {
		if !strings.Contains(ts.Signature, part) {
			t.Errorf("signature template should contain %q", part)
		}
	}
}
