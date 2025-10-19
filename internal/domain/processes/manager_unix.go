//go:build !windows

package processes

// killWindowsDevProcessesImpl is a no-op on Unix
func (m *Manager) killWindowsDevProcessesImpl() {
	// No-op on Unix systems
}
