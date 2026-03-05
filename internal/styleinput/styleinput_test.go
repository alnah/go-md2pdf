package styleinput

import "testing"

func TestClassify(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		input        string
		defaultValue string
		allowRawCSS  bool
		wantSource   Source
		wantValue    string
	}{
		{
			name:       "empty with no default",
			input:      "",
			wantSource: SourceNone,
			wantValue:  "",
		},
		{
			name:         "empty falls back to default name",
			input:        "",
			defaultValue: "default",
			wantSource:   SourceName,
			wantValue:    "default",
		},
		{
			name:       "path input",
			input:      "./style.css",
			wantSource: SourceFile,
			wantValue:  "./style.css",
		},
		{
			name:        "raw css when enabled",
			input:       "body { color: red; }",
			allowRawCSS: true,
			wantSource:  SourceRawCSS,
			wantValue:   "body { color: red; }",
		},
		{
			name:       "raw css treated as name when disabled",
			input:      "body { color: red; }",
			wantSource: SourceName,
			wantValue:  "body { color: red; }",
		},
		{
			name:       "name input",
			input:      "technical",
			wantSource: SourceName,
			wantValue:  "technical",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotSource, gotValue := Classify(tt.input, tt.defaultValue, tt.allowRawCSS)
			if gotSource != tt.wantSource {
				t.Errorf("Classify() source = %v, want %v", gotSource, tt.wantSource)
			}
			if gotValue != tt.wantValue {
				t.Errorf("Classify() value = %q, want %q", gotValue, tt.wantValue)
			}
		})
	}
}
