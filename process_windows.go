//go:build windows

package md2pdf

import (
	"os/exec"
	"strconv"
)

// killProcessGroup kills a process and all its children using taskkill.
// /F = force kill, /T = terminate child processes (tree kill).
func killProcessGroup(pid int) {
	// Best-effort cleanup; error ignored as launcher.Kill() provides fallback
	_ = exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(pid)).Run()
}
