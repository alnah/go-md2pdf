package main

import (
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
