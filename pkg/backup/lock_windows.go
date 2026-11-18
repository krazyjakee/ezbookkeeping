//go:build windows

package backup

import (
	"os"
)

func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)

	if err != nil {
		return false
	}

	_ = process.Release()
	return true
}
