//go:build !windows

package process

import "syscall"

// killProcessGroup kills a process and all its children by sending SIGKILL
// to the process group (negative PID).
func KillProcessGroup(pid int) {
	// Best-effort cleanup; error ignored as launcher.Kill() provides fallback
	_ = syscall.Kill(-pid, syscall.SIGKILL)
}
