package main

// Notes:
// - Tests use black-box approach: testing through runDoctorCmd() observable outputs
// - Container detection tests modify environment variables, cannot use t.Parallel()
// - Chrome detection depends on system state, tested via observable JSON output
// - Internal functions (isContainer, checkChrome, checkSystem) are not tested directly
//   as they are implementation details; behavior is verified through command output

import (
	"bytes"
	"encoding/json"
	"os"
	"runtime"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// TestRunDoctorCmd_JSONOutput - Verifies JSON output format and structure
// ---------------------------------------------------------------------------

func TestRunDoctorCmd_JSONOutput(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	env := &Environment{Stdout: &stdout, Stderr: &stderr}

	exitCode := runDoctorCmd([]string{"--json"}, env)

	// Should produce valid JSON
	var result doctorResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("Invalid JSON output: %v\nOutput was: %s", err, stdout.String())
	}

	// Verify required fields are present
	if result.Env.OS == "" {
		t.Error("JSON should contain OS")
	}
	if result.Env.Arch == "" {
		t.Error("JSON should contain Arch")
	}
	if result.Status == "" {
		t.Error("JSON should contain status")
	}

	// Status must be one of the valid values
	validStatuses := map[string]bool{"ready": true, "warnings": true, "errors": true}
	if !validStatuses[result.Status] {
		t.Errorf("Invalid status %q, expected ready/warnings/errors", result.Status)
	}

	// Exit code should be consistent with status
	if result.Status == "errors" && exitCode != ExitGeneral {
		t.Errorf("Expected exit code %d for errors status, got %d", ExitGeneral, exitCode)
	}
	if result.Status != "errors" && exitCode != ExitSuccess {
		t.Errorf("Expected exit code %d for non-error status, got %d", ExitSuccess, exitCode)
	}

	// Platform should match runtime
	if result.Env.OS != runtime.GOOS {
		t.Errorf("OS = %q, want %q", result.Env.OS, runtime.GOOS)
	}
	if result.Env.Arch != runtime.GOARCH {
		t.Errorf("Arch = %q, want %q", result.Env.Arch, runtime.GOARCH)
	}
}

// ---------------------------------------------------------------------------
// TestRunDoctorCmd_HumanOutput - Verifies human-readable output format
// ---------------------------------------------------------------------------

func TestRunDoctorCmd_HumanOutput(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	env := &Environment{Stdout: &stdout, Stderr: &stderr}

	runDoctorCmd([]string{}, env)

	output := stdout.String()

	// Should contain required section headers
	requiredSections := []string{
		"md2pdf doctor",
		"Chrome/Chromium",
		"Environment",
		"System",
		"Status:",
	}
	for _, section := range requiredSections {
		if !strings.Contains(output, section) {
			t.Errorf("Output should contain section %q", section)
		}
	}

	// Should contain platform info
	platformStr := runtime.GOOS + "/" + runtime.GOARCH
	if !strings.Contains(output, platformStr) {
		t.Errorf("Output should contain platform %q", platformStr)
	}
}

// ---------------------------------------------------------------------------
// TestRunDoctorCmd_ContainerDetection - Verifies container environment detection
// ---------------------------------------------------------------------------

func TestRunDoctorCmd_ContainerDetection(t *testing.T) {
	// NO t.Parallel() - modifies environment variables

	tests := []struct {
		name          string
		envVar        string
		envVal        string
		wantContainer bool
		wantHint      string
	}{
		{
			name:          "explicit MD2PDF_CONTAINER override",
			envVar:        "MD2PDF_CONTAINER",
			envVal:        "1",
			wantContainer: true,
			wantHint:      "MD2PDF_CONTAINER=1",
		},
		{
			name:          "kubernetes environment",
			envVar:        "KUBERNETES_SERVICE_HOST",
			envVal:        "10.0.0.1",
			wantContainer: true,
			wantHint:      "KUBERNETES_SERVICE_HOST",
		},
		{
			name:          "podman container",
			envVar:        "container",
			envVal:        "podman",
			wantContainer: true,
			wantHint:      "container=podman",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean all container signals first
			cleanContainerEnv()

			os.Setenv(tt.envVar, tt.envVal)
			defer os.Unsetenv(tt.envVar)

			var stdout bytes.Buffer
			env := &Environment{Stdout: &stdout, Stderr: &bytes.Buffer{}}

			runDoctorCmd([]string{"--json"}, env)

			var result doctorResult
			if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
				t.Fatalf("Invalid JSON: %v", err)
			}

			if result.Env.Container != tt.wantContainer {
				t.Errorf("Container = %v, want %v", result.Env.Container, tt.wantContainer)
			}
			if result.Env.ContainerHint != tt.wantHint {
				t.Errorf("ContainerHint = %q, want %q", result.Env.ContainerHint, tt.wantHint)
			}
		})
	}
}

func TestRunDoctorCmd_ContainerPriority(t *testing.T) {
	// NO t.Parallel() - modifies environment variables

	cleanContainerEnv()

	// Set multiple container signals
	os.Setenv("MD2PDF_CONTAINER", "1")
	os.Setenv("KUBERNETES_SERVICE_HOST", "10.0.0.1")
	defer func() {
		os.Unsetenv("MD2PDF_CONTAINER")
		os.Unsetenv("KUBERNETES_SERVICE_HOST")
	}()

	var stdout bytes.Buffer
	env := &Environment{Stdout: &stdout, Stderr: &bytes.Buffer{}}

	runDoctorCmd([]string{"--json"}, env)

	var result doctorResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	// MD2PDF_CONTAINER should have highest priority
	if result.Env.ContainerHint != "MD2PDF_CONTAINER=1" {
		t.Errorf("MD2PDF_CONTAINER should have priority, got hint %q", result.Env.ContainerHint)
	}
}

// ---------------------------------------------------------------------------
// TestRunDoctorCmd_CIDetection - Verifies CI environment detection
// ---------------------------------------------------------------------------

func TestRunDoctorCmd_CIDetection(t *testing.T) {
	// NO t.Parallel() - modifies environment variables

	tests := []struct {
		name   string
		envVar string
		envVal string
		wantCI bool
	}{
		{"CI generic", "CI", "true", true},
		{"GitHub Actions", "GITHUB_ACTIONS", "true", true},
		{"GitLab CI", "GITLAB_CI", "true", true},
		{"Jenkins", "JENKINS_URL", "http://jenkins.local", true},
		{"CircleCI", "CIRCLECI", "true", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanCIEnv()
			defer saveAndRestoreNoSandbox(t)()

			os.Setenv(tt.envVar, tt.envVal)
			// Also set sandbox to avoid warning noise in output
			os.Setenv("ROD_NO_SANDBOX", "1")
			defer os.Unsetenv(tt.envVar)

			var stdout bytes.Buffer
			env := &Environment{Stdout: &stdout, Stderr: &bytes.Buffer{}}

			runDoctorCmd([]string{"--json"}, env)

			var result doctorResult
			if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
				t.Fatalf("Invalid JSON: %v", err)
			}

			if result.Env.CI != tt.wantCI {
				t.Errorf("CI = %v, want %v", result.Env.CI, tt.wantCI)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestRunDoctorCmd_SandboxWarning - Verifies sandbox warning in container/CI
// ---------------------------------------------------------------------------

func TestRunDoctorCmd_SandboxWarning(t *testing.T) {
	// NO t.Parallel() - modifies environment variables

	cleanContainerEnv()
	cleanCIEnv()
	defer saveAndRestoreNoSandbox(t)()

	os.Unsetenv("ROD_NO_SANDBOX")

	// Simulate CI environment without sandbox disabled
	os.Setenv("CI", "true")
	defer os.Unsetenv("CI")

	var stdout bytes.Buffer
	env := &Environment{Stdout: &stdout, Stderr: &bytes.Buffer{}}

	runDoctorCmd([]string{"--json"}, env)

	var result doctorResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	// Should have warning about sandbox
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "ROD_NO_SANDBOX") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected warning about ROD_NO_SANDBOX when in CI without sandbox disabled")
	}

	// Status should be "warnings"
	if result.Status != "warnings" && result.Status != "errors" {
		// Could be errors if Chrome not found, but if ready, that's wrong
		if result.Status == "ready" && len(result.Warnings) > 0 {
			t.Error("Status should be 'warnings' when warnings present")
		}
	}
}

func TestRunDoctorCmd_NoSandboxWarningWhenDisabled(t *testing.T) {
	// NO t.Parallel() - modifies environment variables

	cleanContainerEnv()
	cleanCIEnv()
	defer saveAndRestoreNoSandbox(t)()

	// Simulate CI with sandbox properly disabled
	os.Setenv("CI", "true")
	os.Setenv("ROD_NO_SANDBOX", "1")
	defer os.Unsetenv("CI")

	var stdout bytes.Buffer
	env := &Environment{Stdout: &stdout, Stderr: &bytes.Buffer{}}

	runDoctorCmd([]string{"--json"}, env)

	var result doctorResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	// Should NOT have sandbox warning
	for _, w := range result.Warnings {
		if strings.Contains(w, "ROD_NO_SANDBOX") {
			t.Error("Should not warn about sandbox when ROD_NO_SANDBOX=1")
		}
	}
}

// ---------------------------------------------------------------------------
// TestRunDoctorCmd_ExitCodes - Verifies correct exit codes
// ---------------------------------------------------------------------------

func TestRunDoctorCmd_ExitCodeSuccess(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	env := &Environment{Stdout: &stdout, Stderr: &bytes.Buffer{}}

	exitCode := runDoctorCmd([]string{"--json"}, env)

	var result doctorResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	// If no errors, exit code should be 0
	if result.Status != "errors" && exitCode != ExitSuccess {
		t.Errorf("Exit code should be %d for status %q, got %d",
			ExitSuccess, result.Status, exitCode)
	}
}

// ---------------------------------------------------------------------------
// TestRunDoctorCmd_TempDirCheck - Verifies temp directory check
// ---------------------------------------------------------------------------

func TestRunDoctorCmd_TempDirWritable(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	env := &Environment{Stdout: &stdout, Stderr: &bytes.Buffer{}}

	runDoctorCmd([]string{"--json"}, env)

	var result doctorResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	// In normal conditions, temp dir should be writable
	if !result.System.TempWritable {
		t.Error("Temp directory should be writable in normal conditions")
	}
}

// ---------------------------------------------------------------------------
// TestRunDoctorCmd_EnvironmentVariables - Verifies env var reporting
// ---------------------------------------------------------------------------

func TestRunDoctorCmd_ReportsRODBrowserBin(t *testing.T) {
	// NO t.Parallel() - modifies environment variables

	testPath := "/custom/chrome/path"
	os.Setenv("ROD_BROWSER_BIN", testPath)
	defer os.Unsetenv("ROD_BROWSER_BIN")

	var stdout bytes.Buffer
	env := &Environment{Stdout: &stdout, Stderr: &bytes.Buffer{}}

	runDoctorCmd([]string{"--json"}, env)

	var result doctorResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	if result.Env.BrowserBin != testPath {
		t.Errorf("BrowserBin = %q, want %q", result.Env.BrowserBin, testPath)
	}
}

func TestRunDoctorCmd_ReportsRODNoSandbox(t *testing.T) {
	// NO t.Parallel() - modifies environment variables

	defer saveAndRestoreNoSandbox(t)()
	os.Setenv("ROD_NO_SANDBOX", "1")

	var stdout bytes.Buffer
	env := &Environment{Stdout: &stdout, Stderr: &bytes.Buffer{}}

	runDoctorCmd([]string{"--json"}, env)

	var result doctorResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	if result.Env.NoSandbox != "1" {
		t.Errorf("NoSandbox = %q, want %q", result.Env.NoSandbox, "1")
	}
}

// ---------------------------------------------------------------------------
// TestRunDoctorCmd_HumanOutput_Formatting - Verifies human output formatting
// ---------------------------------------------------------------------------

func TestRunDoctorCmd_HumanOutput_ShowsContainerInfo(t *testing.T) {
	// NO t.Parallel() - modifies environment variables

	cleanContainerEnv()
	defer saveAndRestoreNoSandbox(t)()

	os.Setenv("MD2PDF_CONTAINER", "1")
	os.Setenv("ROD_NO_SANDBOX", "1") // Avoid warning
	defer os.Unsetenv("MD2PDF_CONTAINER")

	var stdout bytes.Buffer
	env := &Environment{Stdout: &stdout, Stderr: &bytes.Buffer{}}

	runDoctorCmd([]string{}, env)

	output := stdout.String()

	if !strings.Contains(output, "Container: detected") {
		t.Error("Human output should show container detection")
	}
	if !strings.Contains(output, "MD2PDF_CONTAINER=1") {
		t.Error("Human output should show container hint")
	}
}

func TestRunDoctorCmd_HumanOutput_ShowsCIInfo(t *testing.T) {
	// NO t.Parallel() - modifies environment variables

	cleanCIEnv()
	defer saveAndRestoreNoSandbox(t)()

	os.Setenv("GITHUB_ACTIONS", "true")
	os.Setenv("ROD_NO_SANDBOX", "1") // Avoid warning
	defer os.Unsetenv("GITHUB_ACTIONS")

	var stdout bytes.Buffer
	env := &Environment{Stdout: &stdout, Stderr: &bytes.Buffer{}}

	runDoctorCmd([]string{}, env)

	output := stdout.String()

	if !strings.Contains(output, "CI: detected") {
		t.Error("Human output should show CI detection")
	}
}

func TestRunDoctorCmd_HumanOutput_ShowsWarnings(t *testing.T) {
	// NO t.Parallel() - modifies environment variables

	cleanContainerEnv()
	cleanCIEnv()
	defer saveAndRestoreNoSandbox(t)()

	os.Unsetenv("ROD_NO_SANDBOX")

	os.Setenv("CI", "true")
	defer os.Unsetenv("CI")

	var stdout bytes.Buffer
	env := &Environment{Stdout: &stdout, Stderr: &bytes.Buffer{}}

	runDoctorCmd([]string{}, env)

	output := stdout.String()

	if !strings.Contains(output, "[WARN]") {
		t.Error("Human output should show warnings with [WARN] prefix")
	}
	if !strings.Contains(output, "ROD_NO_SANDBOX") {
		t.Error("Warning about ROD_NO_SANDBOX should be visible")
	}
}

func TestRunDoctorCmd_HumanOutput_StatusLine(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	env := &Environment{Stdout: &stdout, Stderr: &bytes.Buffer{}}

	runDoctorCmd([]string{}, env)

	output := stdout.String()

	// Should end with one of the valid status lines
	validStatusLines := []string{
		"Status: Ready to convert",
		"Status: Ready with warnings",
		"Status: Not ready (see errors above)",
	}

	found := false
	for _, status := range validStatusLines {
		if strings.Contains(output, status) {
			found = true
			break
		}
	}
	if !found {
		t.Error("Human output should contain a valid status line")
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// cleanContainerEnv removes all container detection environment variables.
func cleanContainerEnv() {
	os.Unsetenv("MD2PDF_CONTAINER")
	os.Unsetenv("KUBERNETES_SERVICE_HOST")
	os.Unsetenv("container")
}

// cleanCIEnv removes all CI detection environment variables.
func cleanCIEnv() {
	os.Unsetenv("CI")
	os.Unsetenv("GITHUB_ACTIONS")
	os.Unsetenv("GITLAB_CI")
	os.Unsetenv("JENKINS_URL")
	os.Unsetenv("CIRCLECI")
}

// saveAndRestoreNoSandbox saves the current ROD_NO_SANDBOX value and returns
// a cleanup function that restores it. Use with defer.
func saveAndRestoreNoSandbox(t *testing.T) func() {
	t.Helper()
	orig := os.Getenv("ROD_NO_SANDBOX")
	return func() {
		if orig != "" {
			os.Setenv("ROD_NO_SANDBOX", orig)
		} else {
			os.Unsetenv("ROD_NO_SANDBOX")
		}
	}
}
