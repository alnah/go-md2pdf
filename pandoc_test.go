package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type MockRunner struct {
	Stdout     string
	Stderr     string
	Err        error
	CalledWith []string
}

func (m *MockRunner) Run(name string, args ...string) (string, string, error) {
	m.CalledWith = append([]string{name}, args...)
	return m.Stdout, m.Stderr, m.Err
}

func TestPandocConverter_ToHTML(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		createFile     bool
		mock           *MockRunner
		wantErr        error
		wantAnyErr     bool
		wantOutput     string
		wantCalledWith []string
	}{
		{
			name:    "empty path returns ErrEmptyPath",
			path:    "",
			mock:    &MockRunner{},
			wantErr: ErrEmptyPath,
		},
		{
			name:    "wrong extension .txt returns ErrInvalidExtension",
			path:    "file.txt",
			mock:    &MockRunner{},
			wantErr: ErrInvalidExtension,
		},
		{
			name:    "no extension returns ErrInvalidExtension",
			path:    "README",
			mock:    &MockRunner{},
			wantErr: ErrInvalidExtension,
		},
		{
			name:    "non-existent .md file returns ErrFileNotFound",
			path:    "does-not-exist.md",
			mock:    &MockRunner{},
			wantErr: ErrFileNotFound,
		},
		{
			name:       "pandoc succeeds returns HTML",
			createFile: true,
			mock: &MockRunner{
				Stdout: "<html><body><h1>Test</h1></body></html>",
			},
			wantOutput:     "<html><body><h1>Test</h1></body></html>",
			wantCalledWith: []string{"pandoc", "", "-t", "html5", "--standalone"},
		},
		{
			name:       "pandoc fails returns error with stderr",
			createFile: true,
			mock: &MockRunner{
				Stderr: "pandoc: unknown option --bad",
				Err:    errors.New("exit status 1"),
			},
			wantAnyErr:     true,
			wantCalledWith: []string{"pandoc", "", "-t", "html5", "--standalone"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.path

			if tt.createFile {
				tmpDir := t.TempDir()
				path = filepath.Join(tmpDir, "test.md")
				if err := writeTestFile(path, "# Test"); err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
				if tt.wantCalledWith != nil {
					tt.wantCalledWith[1] = path
				}
			}

			converter := &PandocConverter{Runner: tt.mock}
			got, err := converter.ToHTML(path)

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

			if tt.wantOutput != "" && got != tt.wantOutput {
				t.Errorf("expected output %q, got %q", tt.wantOutput, got)
			}

			if tt.wantCalledWith != nil {
				if len(tt.mock.CalledWith) != len(tt.wantCalledWith) {
					t.Fatalf("expected %d args, got %d: %v", len(tt.wantCalledWith), len(tt.mock.CalledWith), tt.mock.CalledWith)
				}
				for i, want := range tt.wantCalledWith {
					if tt.mock.CalledWith[i] != want {
						t.Errorf("arg[%d]: expected %q, got %q", i, want, tt.mock.CalledWith[i])
					}
				}
			}
		})
	}
}

func writeTestFile(path, content string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(content)
	return err
}
