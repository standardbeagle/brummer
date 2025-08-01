//go:build windows

package discovery

import "os"

// isFileLockedUnix is a stub for Windows (not used, but needed for compilation)
func isFileLockedUnix(file *os.File) bool {
	// This function is not called on Windows, but needed for compilation
	return false
}
