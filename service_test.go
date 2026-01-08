package md2pdf

import (
	"context"
	"errors"
	"testing"
)

// Mock implementations for testing.

type mockPreprocessor struct {
	called bool
	input  string
	output string
}

func (m *mockPreprocessor) PreprocessMarkdown(ctx context.Context, content string) string {
	m.called = true
	m.input = content
	if m.output != "" {
		return m.output
	}
	return content
}

type mockHTMLConverter struct {
	called bool
	input  string
	output string
	err    error
}

func (m *mockHTMLConverter) ToHTML(ctx context.Context, content string) (string, error) {
	m.called = true
	m.input = content
	if m.err != nil {
		return "", m.err
	}
	if m.output != "" {
		return m.output, nil
	}
	return "<html>" + content + "</html>", nil
}

type mockCSSInjector struct {
	called    bool
	inputHTML string
	inputCSS  string
	output    string
}

func (m *mockCSSInjector) InjectCSS(ctx context.Context, htmlContent, cssContent string) string {
	m.called = true
	m.inputHTML = htmlContent
	m.inputCSS = cssContent
	if m.output != "" {
		return m.output
	}
	return htmlContent
}

type mockPDFConverter struct {
	called    bool
	inputHTML string
	inputOpts *pdfOptions
	output    []byte
	err       error
}

func (m *mockPDFConverter) ToPDF(ctx context.Context, htmlContent string, opts *pdfOptions) ([]byte, error) {
	m.called = true
	m.inputHTML = htmlContent
	m.inputOpts = opts
	if m.err != nil {
		return nil, m.err
	}
	if m.output != nil {
		return m.output, nil
	}
	return []byte("%PDF-1.4 mock"), nil
}

func (m *mockPDFConverter) Close() error {
	return nil
}

type mockSignatureInjector struct {
	called    bool
	inputHTML string
	inputData *signatureData
	output    string
	err       error
}

func (m *mockSignatureInjector) InjectSignature(ctx context.Context, htmlContent string, data *signatureData) (string, error) {
	m.called = true
	m.inputHTML = htmlContent
	m.inputData = data
	if m.err != nil {
		return "", m.err
	}
	if m.output != "" {
		return m.output, nil
	}
	return htmlContent, nil
}

// Test options for dependency injection (not exported).

func withPreprocessor(p markdownPreprocessor) Option {
	return func(s *Service) {
		s.preprocessor = p
	}
}

func withHTMLConverter(c htmlConverter) Option {
	return func(s *Service) {
		s.htmlConverter = c
	}
}

func withCSSInjector(c cssInjector) Option {
	return func(s *Service) {
		s.cssInjector = c
	}
}

func withSignatureInjector(i signatureInjector) Option {
	return func(s *Service) {
		s.signatureInjector = i
	}
}

func withPDFConverter(c pdfConverter) Option {
	return func(s *Service) {
		s.pdfConverter = c
	}
}

func TestValidateInput(t *testing.T) {
	service := New()
	defer service.Close()

	tests := []struct {
		name    string
		input   Input
		wantErr error
	}{
		{
			name:    "valid input",
			input:   Input{Markdown: "# Hello"},
			wantErr: nil,
		},
		{
			name:    "empty markdown",
			input:   Input{Markdown: ""},
			wantErr: ErrEmptyMarkdown,
		},
		{
			name:    "with CSS",
			input:   Input{Markdown: "# Hello", CSS: "body { color: red; }"},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateInput(tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("validateInput() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConvert_Success(t *testing.T) {
	preprocessor := &mockPreprocessor{output: "preprocessed"}
	htmlConv := &mockHTMLConverter{output: "<html>converted</html>"}
	cssInj := &mockCSSInjector{output: "<html>with-css</html>"}
	sigInjector := &mockSignatureInjector{output: "<html>with-sig</html>"}
	pdfConv := &mockPDFConverter{output: []byte("%PDF-1.4 test")}

	service := New(
		withPreprocessor(preprocessor),
		withHTMLConverter(htmlConv),
		withCSSInjector(cssInj),
		withSignatureInjector(sigInjector),
		withPDFConverter(pdfConv),
	)
	defer service.Close()

	input := Input{
		Markdown: "# Hello",
		CSS:      "body {}",
	}

	ctx := context.Background()
	result, err := service.Convert(ctx, input)
	if err != nil {
		t.Fatalf("Convert() unexpected error: %v", err)
	}

	if string(result) != "%PDF-1.4 test" {
		t.Errorf("Convert() result = %q, want %q", result, "%PDF-1.4 test")
	}

	// Verify pipeline was called in order with correct inputs
	if !preprocessor.called {
		t.Error("preprocessor was not called")
	}
	if preprocessor.input != "# Hello" {
		t.Errorf("preprocessor input = %q, want %q", preprocessor.input, "# Hello")
	}

	if !htmlConv.called {
		t.Error("htmlConverter was not called")
	}
	if htmlConv.input != "preprocessed" {
		t.Errorf("htmlConverter input = %q, want %q", htmlConv.input, "preprocessed")
	}

	if !cssInj.called {
		t.Error("cssInjector was not called")
	}
	if cssInj.inputHTML != "<html>converted</html>" {
		t.Errorf("cssInjector inputHTML = %q, want %q", cssInj.inputHTML, "<html>converted</html>")
	}
	if cssInj.inputCSS != "body {}" {
		t.Errorf("cssInjector inputCSS = %q, want %q", cssInj.inputCSS, "body {}")
	}

	if !sigInjector.called {
		t.Error("signatureInjector was not called")
	}
	if sigInjector.inputHTML != "<html>with-css</html>" {
		t.Errorf("signatureInjector inputHTML = %q, want %q", sigInjector.inputHTML, "<html>with-css</html>")
	}

	if !pdfConv.called {
		t.Error("pdfConverter was not called")
	}
	if pdfConv.inputHTML != "<html>with-sig</html>" {
		t.Errorf("pdfConverter inputHTML = %q, want %q", pdfConv.inputHTML, "<html>with-sig</html>")
	}
}

func TestConvert_ValidationError(t *testing.T) {
	service := New()
	defer service.Close()

	ctx := context.Background()
	_, err := service.Convert(ctx, Input{Markdown: ""})

	if !errors.Is(err, ErrEmptyMarkdown) {
		t.Errorf("Convert() error = %v, want %v", err, ErrEmptyMarkdown)
	}
}

func TestConvert_HTMLConverterError(t *testing.T) {
	htmlErr := errors.New("pandoc failed")

	service := New(
		withPreprocessor(&mockPreprocessor{}),
		withHTMLConverter(&mockHTMLConverter{err: htmlErr}),
		withCSSInjector(&mockCSSInjector{}),
		withSignatureInjector(&mockSignatureInjector{}),
		withPDFConverter(&mockPDFConverter{}),
	)
	defer service.Close()

	ctx := context.Background()
	_, err := service.Convert(ctx, Input{Markdown: "# Hello"})

	if err == nil {
		t.Fatal("Convert() expected error, got nil")
	}
	if !errors.Is(err, htmlErr) {
		t.Errorf("Convert() error should wrap %v, got %v", htmlErr, err)
	}
}

func TestConvert_PDFConverterError(t *testing.T) {
	pdfErr := errors.New("chrome failed")

	service := New(
		withPreprocessor(&mockPreprocessor{}),
		withHTMLConverter(&mockHTMLConverter{}),
		withCSSInjector(&mockCSSInjector{}),
		withSignatureInjector(&mockSignatureInjector{}),
		withPDFConverter(&mockPDFConverter{err: pdfErr}),
	)
	defer service.Close()

	ctx := context.Background()
	_, err := service.Convert(ctx, Input{Markdown: "# Hello"})

	if err == nil {
		t.Fatal("Convert() expected error, got nil")
	}
	if !errors.Is(err, pdfErr) {
		t.Errorf("Convert() error should wrap %v, got %v", pdfErr, err)
	}
}

func TestConvert_SignatureInjectorError(t *testing.T) {
	sigErr := errors.New("signature template failed")

	service := New(
		withPreprocessor(&mockPreprocessor{}),
		withHTMLConverter(&mockHTMLConverter{}),
		withCSSInjector(&mockCSSInjector{}),
		withSignatureInjector(&mockSignatureInjector{err: sigErr}),
		withPDFConverter(&mockPDFConverter{}),
	)
	defer service.Close()

	ctx := context.Background()
	_, err := service.Convert(ctx, Input{Markdown: "# Hello"})

	if err == nil {
		t.Fatal("Convert() expected error, got nil")
	}
	if !errors.Is(err, sigErr) {
		t.Errorf("Convert() error should wrap %v, got %v", sigErr, err)
	}
}

func TestConvert_NoCSSByDefault(t *testing.T) {
	cssInj := &mockCSSInjector{}

	service := New(
		withPreprocessor(&mockPreprocessor{}),
		withHTMLConverter(&mockHTMLConverter{}),
		withCSSInjector(cssInj),
		withSignatureInjector(&mockSignatureInjector{}),
		withPDFConverter(&mockPDFConverter{}),
	)
	defer service.Close()

	ctx := context.Background()
	_, err := service.Convert(ctx, Input{Markdown: "# Hello"})

	if err != nil {
		t.Fatalf("Convert() unexpected error: %v", err)
	}

	if cssInj.inputCSS != "" {
		t.Errorf("cssInjector should receive empty CSS by default, got %q", cssInj.inputCSS)
	}
}

func TestNew(t *testing.T) {
	service := New()
	defer service.Close()

	if service.preprocessor == nil {
		t.Error("preprocessor is nil")
	}
	if service.htmlConverter == nil {
		t.Error("htmlConverter is nil")
	}
	if service.cssInjector == nil {
		t.Error("cssInjector is nil")
	}
	if service.signatureInjector == nil {
		t.Error("signatureInjector is nil")
	}
	if service.pdfConverter == nil {
		t.Error("pdfConverter is nil")
	}
}

func TestWithTimeout(t *testing.T) {
	service := New(WithTimeout(60 * defaultTimeout))
	defer service.Close()

	if service.cfg.timeout != 60*defaultTimeout {
		t.Errorf("timeout = %v, want %v", service.cfg.timeout, 60*defaultTimeout)
	}
}

func TestService_Close(t *testing.T) {
	service := New()

	// Close should not error
	if err := service.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Double close should also not error
	if err := service.Close(); err != nil {
		t.Errorf("Close() second call error = %v", err)
	}
}

func TestToSignatureData(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		result := toSignatureData(nil)
		if result != nil {
			t.Error("expected nil for nil input")
		}
	})

	t.Run("converts all fields", func(t *testing.T) {
		sig := &Signature{
			Name:      "John Doe",
			Title:     "Developer",
			Email:     "john@example.com",
			ImagePath: "/path/to/image.png",
			Links: []Link{
				{Label: "GitHub", URL: "https://github.com/john"},
			},
		}

		result := toSignatureData(sig)

		if result.Name != sig.Name {
			t.Errorf("Name = %q, want %q", result.Name, sig.Name)
		}
		if result.Title != sig.Title {
			t.Errorf("Title = %q, want %q", result.Title, sig.Title)
		}
		if result.Email != sig.Email {
			t.Errorf("Email = %q, want %q", result.Email, sig.Email)
		}
		if result.ImagePath != sig.ImagePath {
			t.Errorf("ImagePath = %q, want %q", result.ImagePath, sig.ImagePath)
		}
		if len(result.Links) != 1 {
			t.Fatalf("Links count = %d, want 1", len(result.Links))
		}
		if result.Links[0].Label != "GitHub" || result.Links[0].URL != "https://github.com/john" {
			t.Errorf("Links[0] = %+v, want {GitHub, https://github.com/john}", result.Links[0])
		}
	})
}

func TestToFooterData(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		result := toFooterData(nil)
		if result != nil {
			t.Error("expected nil for nil input")
		}
	})

	t.Run("converts all fields", func(t *testing.T) {
		footer := &Footer{
			Position:       "center",
			ShowPageNumber: true,
			Date:           "2025-01-15",
			Status:         "DRAFT",
			Text:           "Footer",
		}

		result := toFooterData(footer)

		if result.Position != footer.Position {
			t.Errorf("Position = %q, want %q", result.Position, footer.Position)
		}
		if result.ShowPageNumber != footer.ShowPageNumber {
			t.Errorf("ShowPageNumber = %v, want %v", result.ShowPageNumber, footer.ShowPageNumber)
		}
		if result.Date != footer.Date {
			t.Errorf("Date = %q, want %q", result.Date, footer.Date)
		}
		if result.Status != footer.Status {
			t.Errorf("Status = %q, want %q", result.Status, footer.Status)
		}
		if result.Text != footer.Text {
			t.Errorf("Text = %q, want %q", result.Text, footer.Text)
		}
	})
}

func TestToCoverData(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		result := toCoverData(nil)
		if result != nil {
			t.Error("expected nil for nil input")
		}
	})

	t.Run("converts all fields", func(t *testing.T) {
		cover := &Cover{
			Title:        "My Document",
			Subtitle:     "A Comprehensive Guide",
			Logo:         "/path/to/logo.png",
			Author:       "John Doe",
			AuthorTitle:  "Senior Developer",
			Organization: "Acme Corp",
			Date:         "2025-01-15",
			Version:      "v1.0.0",
		}

		result := toCoverData(cover)

		if result.Title != cover.Title {
			t.Errorf("Title = %q, want %q", result.Title, cover.Title)
		}
		if result.Subtitle != cover.Subtitle {
			t.Errorf("Subtitle = %q, want %q", result.Subtitle, cover.Subtitle)
		}
		if result.Logo != cover.Logo {
			t.Errorf("Logo = %q, want %q", result.Logo, cover.Logo)
		}
		if result.Author != cover.Author {
			t.Errorf("Author = %q, want %q", result.Author, cover.Author)
		}
		if result.AuthorTitle != cover.AuthorTitle {
			t.Errorf("AuthorTitle = %q, want %q", result.AuthorTitle, cover.AuthorTitle)
		}
		if result.Organization != cover.Organization {
			t.Errorf("Organization = %q, want %q", result.Organization, cover.Organization)
		}
		if result.Date != cover.Date {
			t.Errorf("Date = %q, want %q", result.Date, cover.Date)
		}
		if result.Version != cover.Version {
			t.Errorf("Version = %q, want %q", result.Version, cover.Version)
		}
	})

	t.Run("empty fields preserved", func(t *testing.T) {
		cover := &Cover{
			Title: "Only Title",
			// All other fields empty
		}

		result := toCoverData(cover)

		if result.Title != "Only Title" {
			t.Errorf("Title = %q, want %q", result.Title, "Only Title")
		}
		if result.Subtitle != "" {
			t.Errorf("Subtitle = %q, want empty", result.Subtitle)
		}
		if result.Logo != "" {
			t.Errorf("Logo = %q, want empty", result.Logo)
		}
		if result.Author != "" {
			t.Errorf("Author = %q, want empty", result.Author)
		}
	})
}
