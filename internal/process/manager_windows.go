//go:build windows

package process

// killWindowsDevProcessesImpl is the Windows implementation
func (m *Manager) killWindowsDevProcessesImpl() {
	// Common Node.js and development server process names on Windows
	processes := []string{
		"node.exe",
		"npm.cmd",
		"yarn.cmd",
		"pnpm.cmd",
		"bun.exe",
		"webpack",
		"vite",
		"next",
		"react-scripts",
		"vue-cli-service",
		"ng",
		"nuxt",
	}

	for _, proc := range processes {
		killProcessesByName(proc)
	}
}
