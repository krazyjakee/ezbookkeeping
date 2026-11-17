//go:build !windows

package backup

import (
	"errors"
	"syscall"
)

func isProcessRunning(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}
