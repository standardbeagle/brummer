//go:build !windows

package process

// killWindowsDevProcessesImpl is a no-op on Unix
func (m *Manager) killWindowsDevProcessesImpl() {
	// No-op on Unix systems
}
