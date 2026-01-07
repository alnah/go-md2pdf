package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"testing"

	md2pdf "github.com/alnah/go-md2pdf"
)

// wrongTypeConverter is a Converter that is NOT *md2pdf.Service.
type wrongTypeConverter struct{}

func (w *wrongTypeConverter) Convert(_ context.Context, _ md2pdf.Input) ([]byte, error) {
	return []byte("%PDF-1.4 mock"), nil
}

func TestPoolAdapter_Release_WrongType(t *testing.T) {
	// Create a real pool with size 1
	pool := md2pdf.NewServicePool(1)
	defer pool.Close()

	adapter := &poolAdapter{pool: pool}

	// Capture stderr to verify error message
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Release with wrong type - should log error, not panic
	wrongType := &wrongTypeConverter{}
	adapter.Release(wrongType) // Should not panic

	// Restore stderr and read captured output
	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Verify error was logged
	expectedMsg := "poolAdapter.Release: unexpected type"
	if output == "" || !bytes.Contains([]byte(output), []byte(expectedMsg)) {
		t.Errorf("expected error message containing %q, got %q", expectedMsg, output)
	}
}

func TestPoolAdapter_Size(t *testing.T) {
	pool := md2pdf.NewServicePool(3)
	defer pool.Close()

	adapter := &poolAdapter{pool: pool}

	if adapter.Size() != 3 {
		t.Errorf("Size() = %d, want 3", adapter.Size())
	}
}

func TestPoolAdapter_AcquireRelease(t *testing.T) {
	pool := md2pdf.NewServicePool(1)
	defer pool.Close()

	adapter := &poolAdapter{pool: pool}

	// Acquire should return a non-nil Converter
	svc := adapter.Acquire()
	if svc == nil {
		t.Fatal("Acquire() returned nil")
	}

	// Release should not panic
	adapter.Release(svc)
}

func TestVersion(t *testing.T) {
	// Version variable should be set (default is "dev")
	if Version == "" {
		t.Error("Version should not be empty")
	}

	// Capture output to verify version format
	expected := fmt.Sprintf("go-md2pdf %s\n", Version)
	_ = expected // Used in actual main() but we can't easily test that
}
