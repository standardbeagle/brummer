//go:build unix

package discovery

import (
	"os"
	"syscall"
)

// isFileLockedUnix checks if a file is locked on Unix systems using flock
func isFileLockedUnix(file *os.File) bool {
	// Try to get an exclusive lock (non-blocking)
	err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err == nil {
		// We got the lock, so it wasn't locked
		syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		return false
	}

	// Lock acquisition failed, file is locked
	return true
}
