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
			styleName: "fle",
			wantErr:   nil,
		},
		{
			name:      "nonexistent style returns ErrStyleNotFound",
			styleName: "nonexistent",
			wantErr:   ErrStyleNotFound,
		},
		{
			name:      "empty name returns ErrInvalidStyleName",
			styleName: "",
			wantErr:   ErrInvalidStyleName,
		},
		{
			name:      "path traversal with slash returns ErrInvalidStyleName",
			styleName: "../secret",
			wantErr:   ErrInvalidStyleName,
		},
		{
			name:      "path traversal with backslash returns ErrInvalidStyleName",
			styleName: "..\\secret",
			wantErr:   ErrInvalidStyleName,
		},
		{
			name:      "path with dot returns ErrInvalidStyleName",
			styleName: "style.name",
			wantErr:   ErrInvalidStyleName,
		},
		{
			name:      "absolute path returns ErrInvalidStyleName",
			styleName: "/etc/passwd",
			wantErr:   ErrInvalidStyleName,
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

func TestLoadTemplate(t *testing.T) {
	tests := []struct {
		name         string
		templateName string
		wantErr      error
	}{
		{
			name:         "valid template returns content",
			templateName: "signature",
			wantErr:      nil,
		},
		{
			name:         "nonexistent template returns ErrTemplateNotFound",
			templateName: "nonexistent",
			wantErr:      ErrTemplateNotFound,
		},
		{
			name:         "empty name returns ErrInvalidTemplateName",
			templateName: "",
			wantErr:      ErrInvalidTemplateName,
		},
		{
			name:         "path traversal with slash returns ErrInvalidTemplateName",
			templateName: "../secret",
			wantErr:      ErrInvalidTemplateName,
		},
		{
			name:         "path traversal with backslash returns ErrInvalidTemplateName",
			templateName: "..\\secret",
			wantErr:      ErrInvalidTemplateName,
		},
		{
			name:         "path with dot returns ErrInvalidTemplateName",
			templateName: "template.name",
			wantErr:      ErrInvalidTemplateName,
		},
		{
			name:         "absolute path returns ErrInvalidTemplateName",
			templateName: "/etc/passwd",
			wantErr:      ErrInvalidTemplateName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := LoadTemplate(tt.templateName)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("LoadTemplate(%q) error = %v, want %v", tt.templateName, err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("LoadTemplate(%q) unexpected error: %v", tt.templateName, err)
			}

			if content == "" {
				t.Errorf("LoadTemplate(%q) returned empty content", tt.templateName)
			}
		})
	}
}

func TestLoadTemplate_SignatureContent(t *testing.T) {
	content, err := LoadTemplate("signature")
	if err != nil {
		t.Fatalf("LoadTemplate(signature) error: %v", err)
	}

	// Verify template contains expected Go template structure
	expectedParts := []string{
		"signature-block",
		"{{.Name}}",
		"{{.Email}}",
	}

	for _, part := range expectedParts {
		if !strings.Contains(content, part) {
			t.Errorf("signature template should contain %q", part)
		}
	}
}
