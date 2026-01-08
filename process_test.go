package md2pdf

import "testing"

func TestKillProcessGroup_InvalidPID(t *testing.T) {
	// Verify function handles non-existent PID without panicking.
	// Actual kill behavior is tested via browser cleanup integration tests.
	killProcessGroup(999999999)
}