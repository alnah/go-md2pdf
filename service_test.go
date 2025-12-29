package main

import (
	"errors"
	"testing"
)

// Mock implementations for testing.

type mockPreprocessor struct {
	called bool
	input  string
	output string
}

func (m *mockPreprocessor) PreprocessMarkdown(content string) string {
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

func (m *mockHTMLConverter) ToHTML(content string) (string, error) {
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

func (m *mockCSSInjector) InjectCSS(htmlContent, cssContent string) string {
	m.called = true
	m.inputHTML = htmlContent
	m.inputCSS = cssContent
	if m.output != "" {
		return m.output
	}
	return htmlContent
}

type mockPDFConverter struct {
	called     bool
	inputHTML  string
	outputPath string
	err        error
}

func (m *mockPDFConverter) ToPDF(htmlContent, outputPath string) error {
	m.called = true
	m.inputHTML = htmlContent
	m.outputPath = outputPath
	return m.err
}

type mockSignatureInjector struct {
	called    bool
	inputHTML string
	inputData *SignatureData
	output    string
	err       error
}

func (m *mockSignatureInjector) InjectSignature(htmlContent string, data *SignatureData) (string, error) {
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

func TestValidateOptions(t *testing.T) {
	service := &ConversionService{}

	tests := []struct {
		name    string
		opts    ConversionOptions
		wantErr error
	}{
		{
			name: "valid options",
			opts: ConversionOptions{
				MarkdownContent: "# Hello",
				OutputPath:      "out.pdf",
			},
			wantErr: nil,
		},
		{
			name: "empty markdown",
			opts: ConversionOptions{
				MarkdownContent: "",
				OutputPath:      "out.pdf",
			},
			wantErr: ErrEmptyMarkdown,
		},
		{
			name: "empty output path",
			opts: ConversionOptions{
				MarkdownContent: "# Hello",
				OutputPath:      "",
			},
			wantErr: ErrEmptyOutput,
		},
		{
			name: "both empty returns ErrEmptyMarkdown first",
			opts: ConversionOptions{
				MarkdownContent: "",
				OutputPath:      "",
			},
			wantErr: ErrEmptyMarkdown,
		},
		{
			name: "CSSContent is valid",
			opts: ConversionOptions{
				MarkdownContent: "# Hello",
				OutputPath:      "out.pdf",
				CSSContent:      "body { color: red; }",
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateOptions(tt.opts)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("validateOptions() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestResolveCSS(t *testing.T) {
	service := &ConversionService{}

	tests := []struct {
		name string
		opts ConversionOptions
		want string
	}{
		{
			name: "empty CSSContent returns empty string",
			opts: ConversionOptions{},
			want: "",
		},
		{
			name: "custom CSS returns CSSContent",
			opts: ConversionOptions{CSSContent: "body { color: red; }"},
			want: "body { color: red; }",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := service.resolveCSS(tt.opts)
			if got != tt.want {
				t.Errorf("resolveCSS() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConvert_Success(t *testing.T) {
	preprocessor := &mockPreprocessor{output: "preprocessed"}
	htmlConverter := &mockHTMLConverter{output: "<html>converted</html>"}
	cssInjector := &mockCSSInjector{output: "<html>with-css</html>"}
	signatureInjector := &mockSignatureInjector{output: "<html>with-sig</html>"}
	pdfConverter := &mockPDFConverter{}

	service := NewConversionServiceWith(preprocessor, htmlConverter, cssInjector, signatureInjector, pdfConverter)

	opts := ConversionOptions{
		MarkdownContent: "# Hello",
		OutputPath:      "out.pdf",
		CSSContent:      "body {}",
	}

	err := service.Convert(opts)
	if err != nil {
		t.Fatalf("Convert() unexpected error: %v", err)
	}

	// Verify pipeline was called in order with correct inputs
	if !preprocessor.called {
		t.Error("preprocessor was not called")
	}
	if preprocessor.input != "# Hello" {
		t.Errorf("preprocessor input = %q, want %q", preprocessor.input, "# Hello")
	}

	if !htmlConverter.called {
		t.Error("htmlConverter was not called")
	}
	if htmlConverter.input != "preprocessed" {
		t.Errorf("htmlConverter input = %q, want %q", htmlConverter.input, "preprocessed")
	}

	if !cssInjector.called {
		t.Error("cssInjector was not called")
	}
	if cssInjector.inputHTML != "<html>converted</html>" {
		t.Errorf("cssInjector inputHTML = %q, want %q", cssInjector.inputHTML, "<html>converted</html>")
	}
	if cssInjector.inputCSS != "body {}" {
		t.Errorf("cssInjector inputCSS = %q, want %q", cssInjector.inputCSS, "body {}")
	}

	if !signatureInjector.called {
		t.Error("signatureInjector was not called")
	}
	if signatureInjector.inputHTML != "<html>with-css</html>" {
		t.Errorf("signatureInjector inputHTML = %q, want %q", signatureInjector.inputHTML, "<html>with-css</html>")
	}

	if !pdfConverter.called {
		t.Error("pdfConverter was not called")
	}
	if pdfConverter.inputHTML != "<html>with-sig</html>" {
		t.Errorf("pdfConverter inputHTML = %q, want %q", pdfConverter.inputHTML, "<html>with-sig</html>")
	}
	if pdfConverter.outputPath != "out.pdf" {
		t.Errorf("pdfConverter outputPath = %q, want %q", pdfConverter.outputPath, "out.pdf")
	}
}

func TestConvert_ValidationError(t *testing.T) {
	service := &ConversionService{}

	opts := ConversionOptions{
		MarkdownContent: "",
		OutputPath:      "out.pdf",
	}

	err := service.Convert(opts)
	if !errors.Is(err, ErrEmptyMarkdown) {
		t.Errorf("Convert() error = %v, want %v", err, ErrEmptyMarkdown)
	}
}

func TestConvert_HTMLConverterError(t *testing.T) {
	htmlErr := errors.New("pandoc failed")
	htmlConverter := &mockHTMLConverter{err: htmlErr}

	service := NewConversionServiceWith(
		&mockPreprocessor{},
		htmlConverter,
		&mockCSSInjector{},
		&mockSignatureInjector{},
		&mockPDFConverter{},
	)

	opts := ConversionOptions{
		MarkdownContent: "# Hello",
		OutputPath:      "out.pdf",
	}

	err := service.Convert(opts)
	if err == nil {
		t.Fatal("Convert() expected error, got nil")
	}
	if !errors.Is(err, htmlErr) {
		t.Errorf("Convert() error should wrap %v, got %v", htmlErr, err)
	}
}

func TestConvert_PDFConverterError(t *testing.T) {
	pdfErr := errors.New("chrome failed")
	pdfConverter := &mockPDFConverter{err: pdfErr}

	service := NewConversionServiceWith(
		&mockPreprocessor{},
		&mockHTMLConverter{},
		&mockCSSInjector{},
		&mockSignatureInjector{},
		pdfConverter,
	)

	opts := ConversionOptions{
		MarkdownContent: "# Hello",
		OutputPath:      "out.pdf",
	}

	err := service.Convert(opts)
	if err == nil {
		t.Fatal("Convert() expected error, got nil")
	}
	if !errors.Is(err, pdfErr) {
		t.Errorf("Convert() error should wrap %v, got %v", pdfErr, err)
	}
}

func TestConvert_SignatureInjectorError(t *testing.T) {
	sigErr := errors.New("signature template failed")
	signatureInjector := &mockSignatureInjector{err: sigErr}

	service := NewConversionServiceWith(
		&mockPreprocessor{},
		&mockHTMLConverter{},
		&mockCSSInjector{},
		signatureInjector,
		&mockPDFConverter{},
	)

	opts := ConversionOptions{
		MarkdownContent: "# Hello",
		OutputPath:      "out.pdf",
	}

	err := service.Convert(opts)
	if err == nil {
		t.Fatal("Convert() expected error, got nil")
	}
	if !errors.Is(err, sigErr) {
		t.Errorf("Convert() error should wrap %v, got %v", sigErr, err)
	}
}

func TestConvert_NoCSSByDefault(t *testing.T) {
	cssInjector := &mockCSSInjector{}

	service := NewConversionServiceWith(
		&mockPreprocessor{},
		&mockHTMLConverter{},
		cssInjector,
		&mockSignatureInjector{},
		&mockPDFConverter{},
	)

	opts := ConversionOptions{
		MarkdownContent: "# Hello",
		OutputPath:      "out.pdf",
	}

	err := service.Convert(opts)
	if err != nil {
		t.Fatalf("Convert() unexpected error: %v", err)
	}

	if cssInjector.inputCSS != "" {
		t.Errorf("cssInjector should receive empty CSS by default, got %q", cssInjector.inputCSS)
	}
}

func TestNewConversionServiceWith(t *testing.T) {
	preprocessor := &mockPreprocessor{}
	htmlConverter := &mockHTMLConverter{}
	cssInjector := &mockCSSInjector{}
	signatureInjector := &mockSignatureInjector{}
	pdfConverter := &mockPDFConverter{}

	service := NewConversionServiceWith(preprocessor, htmlConverter, cssInjector, signatureInjector, pdfConverter)

	if service.preprocessor != preprocessor {
		t.Error("preprocessor not set correctly")
	}
	if service.htmlConverter != htmlConverter {
		t.Error("htmlConverter not set correctly")
	}
	if service.cssInjector != cssInjector {
		t.Error("cssInjector not set correctly")
	}
	if service.signatureInjector != signatureInjector {
		t.Error("signatureInjector not set correctly")
	}
	if service.pdfConverter != pdfConverter {
		t.Error("pdfConverter not set correctly")
	}
}

func TestNewConversionServiceWith_NilDependencies(t *testing.T) {
	tests := []struct {
		name              string
		preprocessor      MarkdownPreprocessor
		htmlConverter     HTMLConverter
		cssInjector       CSSInjector
		signatureInjector SignatureInjector
		pdfConverter      PDFConverter
		wantPanic         string
	}{
		{
			name:              "nil preprocessor",
			preprocessor:      nil,
			htmlConverter:     &mockHTMLConverter{},
			cssInjector:       &mockCSSInjector{},
			signatureInjector: &mockSignatureInjector{},
			pdfConverter:      &mockPDFConverter{},
			wantPanic:         "nil preprocessor provided to ConversionService",
		},
		{
			name:              "nil htmlConverter",
			preprocessor:      &mockPreprocessor{},
			htmlConverter:     nil,
			cssInjector:       &mockCSSInjector{},
			signatureInjector: &mockSignatureInjector{},
			pdfConverter:      &mockPDFConverter{},
			wantPanic:         "nil htmlConverter provided to ConversionService",
		},
		{
			name:              "nil cssInjector",
			preprocessor:      &mockPreprocessor{},
			htmlConverter:     &mockHTMLConverter{},
			cssInjector:       nil,
			signatureInjector: &mockSignatureInjector{},
			pdfConverter:      &mockPDFConverter{},
			wantPanic:         "nil cssInjector provided to ConversionService",
		},
		{
			name:              "nil signatureInjector",
			preprocessor:      &mockPreprocessor{},
			htmlConverter:     &mockHTMLConverter{},
			cssInjector:       &mockCSSInjector{},
			signatureInjector: nil,
			pdfConverter:      &mockPDFConverter{},
			wantPanic:         "nil signatureInjector provided to ConversionService",
		},
		{
			name:              "nil pdfConverter",
			preprocessor:      &mockPreprocessor{},
			htmlConverter:     &mockHTMLConverter{},
			cssInjector:       &mockCSSInjector{},
			signatureInjector: &mockSignatureInjector{},
			pdfConverter:      nil,
			wantPanic:         "nil pdfConverter provided to ConversionService",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if r == nil {
					t.Fatal("expected panic, got none")
				}
				if r != tt.wantPanic {
					t.Errorf("panic = %q, want %q", r, tt.wantPanic)
				}
			}()
			NewConversionServiceWith(tt.preprocessor, tt.htmlConverter, tt.cssInjector, tt.signatureInjector, tt.pdfConverter)
		})
	}
}
