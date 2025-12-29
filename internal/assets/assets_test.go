package assets

import (
	"errors"
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
			styleName: "default",
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
