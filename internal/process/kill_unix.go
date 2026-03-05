//go:build !windows

// Package process contains process-control helpers used for robust shutdown.
package process

import "syscall"

// KillProcessGroup kills a process and its children so cancelled conversions
// do not leave orphan browser processes alive.
func KillProcessGroup(pid int) {
	// Best-effort cleanup; error ignored as launcher.Kill() provides fallback
	_ = syscall.Kill(-pid, syscall.SIGKILL)
}
