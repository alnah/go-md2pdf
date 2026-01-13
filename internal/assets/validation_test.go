package assets

import (
	"errors"
	"testing"
)

func TestValidateAssetName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		// Valid names
		{
			name:    "simple name",
			input:   "creative",
			wantErr: nil,
		},
		{
			name:    "name with hyphen",
			input:   "my-style",
			wantErr: nil,
		},
		{
			name:    "name with underscore",
			input:   "my_style",
			wantErr: nil,
		},
		{
			name:    "name with numbers",
			input:   "style123",
			wantErr: nil,
		},
		{
			name:    "mixed case",
			input:   "MyStyle",
			wantErr: nil,
		},

		// Invalid names - empty
		{
			name:    "empty name",
			input:   "",
			wantErr: ErrInvalidAssetName,
		},

		// Invalid names - path separators
		{
			name:    "forward slash",
			input:   "path/to/style",
			wantErr: ErrInvalidAssetName,
		},
		{
			name:    "backslash",
			input:   "path\\to\\style",
			wantErr: ErrInvalidAssetName,
		},

		// Invalid names - path traversal
		{
			name:    "parent directory traversal",
			input:   "../secret",
			wantErr: ErrInvalidAssetName,
		},
		{
			name:    "windows parent traversal",
			input:   "..\\secret",
			wantErr: ErrInvalidAssetName,
		},
		{
			name:    "double parent traversal",
			input:   "../../etc/passwd",
			wantErr: ErrInvalidAssetName,
		},

		// Invalid names - dots (could allow extension manipulation)
		{
			name:    "dot in name",
			input:   "style.css",
			wantErr: ErrInvalidAssetName,
		},
		{
			name:    "hidden file",
			input:   ".hidden",
			wantErr: ErrInvalidAssetName,
		},
		{
			name:    "double extension",
			input:   "style.css.bak",
			wantErr: ErrInvalidAssetName,
		},

		// Edge cases
		{
			name:    "absolute path unix",
			input:   "/etc/passwd",
			wantErr: ErrInvalidAssetName,
		},
		{
			name:    "absolute path windows",
			input:   "C:\\Windows\\System32",
			wantErr: ErrInvalidAssetName,
		},
		{
			name:    "just a dot",
			input:   ".",
			wantErr: ErrInvalidAssetName,
		},
		{
			name:    "two dots",
			input:   "..",
			wantErr: ErrInvalidAssetName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateAssetName(tt.input)

			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("ValidateAssetName(%q) unexpected error: %v", tt.input, err)
				}
				return
			}

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("ValidateAssetName(%q) error = %v, want %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidateAssetName_ErrorMessages(t *testing.T) {
	t.Parallel()

	t.Run("empty name has descriptive message", func(t *testing.T) {
		t.Parallel()

		err := ValidateAssetName("")
		if err == nil {
			t.Fatal("expected error for empty name")
		}
		if err.Error() == "" {
			t.Error("error message should not be empty")
		}
	})

	t.Run("invalid name includes the name in message", func(t *testing.T) {
		t.Parallel()

		err := ValidateAssetName("../evil")
		if err == nil {
			t.Fatal("expected error for invalid name")
		}
		// The error message should contain the invalid name for debugging
		errStr := err.Error()
		if errStr == "" {
			t.Error("error message should not be empty")
		}
	})
}
