package parser

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

type PackageJSON struct {
	Name            string            `json:"name"`
	Version         string            `json:"version"`
	Description     string            `json:"description"`
	Scripts         map[string]string `json:"scripts"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devdependencies"`
	Engines         map[string]string `json:"engines"`
	PackageManager  string            `json:"packageManager"`
	Private         bool              `json:"private"`
	Workspaces      interface{}       `json:"workspaces"` // Can be array or object
}

type PackageManager string

const (
	NPM  PackageManager = "npm"
	Yarn PackageManager = "yarn"
	PNPM PackageManager = "pnpm"
	Bun  PackageManager = "bun"
)

func ParsePackageJSON(path string) (*PackageJSON, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read package.json: %w", err)
	}

	var pkg PackageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("failed to parse package.json: %w", err)
	}

	return &pkg, nil
}

func DetectPackageManager(projectPath string) PackageManager {
	if _, err := os.Stat(filepath.Join(projectPath, "bun.lockb")); err == nil {
		return Bun
	}
	if _, err := os.Stat(filepath.Join(projectPath, "pnpm-lock.yaml")); err == nil {
		return PNPM
	}
	if _, err := os.Stat(filepath.Join(projectPath, "yarn.lock")); err == nil {
		return Yarn
	}
	if _, err := os.Stat(filepath.Join(projectPath, "package-lock.json")); err == nil {
		return NPM
	}
	return NPM
}

func (pm PackageManager) RunCommand() string {
	switch pm {
	case Yarn:
		return "yarn"
	case PNPM:
		return "pnpm"
	case Bun:
		return "bun"
	default:
		return "npm"
	}
}

func (pm PackageManager) RunScriptCommand(script string) []string {
	switch pm {
	case Yarn:
		return []string{"yarn", "run", script}
	case PNPM:
		return []string{"pnpm", "run", script}
	case Bun:
		return []string{"bun", "run", script}
	default:
		return []string{"npm", "run", script}
	}
}

type InstalledPackageManager struct {
	Manager PackageManager
	Version string
	Path    string
}

var (
	cachedInstalledManagers []InstalledPackageManager
	cacheOnce               sync.Once
)

// DetectInstalledPackageManagers checks which package managers are installed on the system
func DetectInstalledPackageManagers() []InstalledPackageManager {
	cacheOnce.Do(func() {
		cachedInstalledManagers = detectInstalledPackageManagersUncached()
	})
	return cachedInstalledManagers
}

// detectInstalledPackageManagersUncached performs the actual detection
func detectInstalledPackageManagersUncached() []InstalledPackageManager {
	var installed []InstalledPackageManager

	managers := []struct {
		name        PackageManager
		command     string
		versionArgs []string
	}{
		{NPM, "npm", []string{"--version"}},
		{Yarn, "yarn", []string{"--version"}},
		{PNPM, "pnpm", []string{"--version"}},
		{Bun, "bun", []string{"--version"}},
	}

	for _, mgr := range managers {
		path, err := findExecutable(mgr.command)
		if err != nil {
			continue
		}

		// Get version with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, mgr.command, mgr.versionArgs...)
		output, err := cmd.Output()
		if err != nil {
			continue
		}

		version := strings.TrimSpace(string(output))
		installed = append(installed, InstalledPackageManager{
			Manager: mgr.name,
			Version: version,
			Path:    path,
		})
	}

	return installed
}

// findExecutable finds the full path to an executable, cross-platform
func findExecutable(name string) (string, error) {
	// Use a channel to implement timeout for LookPath
	type result struct {
		path string
		err  error
	}

	resultChan := make(chan result, 1)

	go func() {
		if runtime.GOOS == "windows" {
			// On Windows, try with common extensions
			extensions := []string{"", ".exe", ".cmd", ".bat"}
			for _, ext := range extensions {
				path, err := exec.LookPath(name + ext)
				if err == nil {
					resultChan <- result{path: path, err: nil}
					return
				}
			}
			resultChan <- result{path: "", err: fmt.Errorf("executable not found: %s", name)}
		} else {
			path, err := exec.LookPath(name)
			resultChan <- result{path: path, err: err}
		}
	}()

	// Wait for result with timeout
	select {
	case res := <-resultChan:
		return res.path, res.err
	case <-time.After(1 * time.Second):
		return "", fmt.Errorf("timeout finding executable: %s", name)
	}
}

// GetPreferredPackageManager determines the preferred package manager based on:
// 1. User preference (if set)
// 2. packageManager field in package.json
// 3. engines field in package.json
// 4. Lock file detection
// 5. First installed manager
func GetPreferredPackageManager(pkg *PackageJSON, projectPath string, userPreference *PackageManager) PackageManager {
	// 1. User preference takes precedence
	if userPreference != nil {
		return *userPreference
	}

	// Handle nil package.json (fallback mode)
	if pkg == nil {
		return DetectPackageManager(projectPath)
	}

	// 2. Check packageManager field (e.g., "yarn@3.2.0")
	if pkg.PackageManager != "" {
		parts := strings.Split(pkg.PackageManager, "@")
		if len(parts) > 0 {
			switch strings.ToLower(parts[0]) {
			case "npm":
				return NPM
			case "yarn":
				return Yarn
			case "pnpm":
				return PNPM
			case "bun":
				return Bun
			}
		}
	}

	// 3. Check engines field
	if pkg.Engines != nil {
		// Check in order of preference
		if _, hasYarn := pkg.Engines["yarn"]; hasYarn {
			return Yarn
		}
		if _, hasPnpm := pkg.Engines["pnpm"]; hasPnpm {
			return PNPM
		}
		if _, hasBun := pkg.Engines["bun"]; hasBun {
			return Bun
		}
		if _, hasNpm := pkg.Engines["npm"]; hasNpm {
			return NPM
		}
	}

	// 4. Lock file detection (existing logic)
	return DetectPackageManager(projectPath)
}
