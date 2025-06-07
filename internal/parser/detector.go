package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ProjectType represents the type of project detected
type ProjectType string

const (
	ProjectTypeNode      ProjectType = "node"
	ProjectTypeGo        ProjectType = "go"
	ProjectTypeRust      ProjectType = "rust"
	ProjectTypeJava      ProjectType = "java"
	ProjectTypeDotNet    ProjectType = "dotnet"
	ProjectTypePython    ProjectType = "python"
	ProjectTypeRuby      ProjectType = "ruby"
	ProjectTypePhp       ProjectType = "php"
	ProjectTypeWails     ProjectType = "wails"
	ProjectTypeElectron  ProjectType = "electron"
	ProjectTypeFlutter   ProjectType = "flutter"
	ProjectTypeMonorepo  ProjectType = "monorepo"
)

// ExecutableCommand represents a detected runnable command
type ExecutableCommand struct {
	Name        string
	Command     string
	Args        []string
	Description string
	Category    string
	ProjectType ProjectType
	Priority    int // Higher priority commands appear first
}

// MonorepoInfo contains information about a monorepo structure
type MonorepoInfo struct {
	Type      string   // pnpm, npm-workspaces, yarn-workspaces, lerna, nx, rush
	Root      string
	Workspaces []string
	Packages   []PackageInfo
}

// PackageInfo represents a package in a monorepo
type PackageInfo struct {
	Path        string
	Name        string
	Scripts     map[string]string
	HasLockFile bool
}

// DetectProjectCommands detects available commands based on project files
func DetectProjectCommands(projectPath string) []ExecutableCommand {
	var commands []ExecutableCommand

	// Check for package.json (Node.js)
	if pkg, err := ParsePackageJSON(filepath.Join(projectPath, "package.json")); err == nil {
		// Add npm scripts
		for name, script := range pkg.Scripts {
			commands = append(commands, ExecutableCommand{
				Name:        name,
				Command:     "npm",
				Args:        []string{"run", name},
				Description: script,
				Category:    "npm scripts",
				ProjectType: ProjectTypeNode,
				Priority:    100,
			})
		}

		// Check for Wails
		if hasWailsConfig(projectPath) {
			commands = append(commands, wailsCommands()...)
		}

		// Check for Electron
		if isElectronProject(pkg) {
			commands = append(commands, electronCommands()...)
		}
	}

	// Check for Go
	if _, err := os.Stat(filepath.Join(projectPath, "go.mod")); err == nil {
		commands = append(commands, goCommands()...)
	}

	// Check for Rust
	if _, err := os.Stat(filepath.Join(projectPath, "Cargo.toml")); err == nil {
		commands = append(commands, rustCommands()...)
	}

	// Check for Java/Gradle
	if _, err := os.Stat(filepath.Join(projectPath, "build.gradle")); err == nil {
		commands = append(commands, gradleCommands()...)
	} else if _, err := os.Stat(filepath.Join(projectPath, "build.gradle.kts")); err == nil {
		commands = append(commands, gradleCommands()...)
	}

	// Check for Maven
	if _, err := os.Stat(filepath.Join(projectPath, "pom.xml")); err == nil {
		commands = append(commands, mavenCommands()...)
	}

	// Check for .NET
	if hasDotNetProject(projectPath) {
		commands = append(commands, dotnetCommands()...)
	}

	// Check for Python
	if _, err := os.Stat(filepath.Join(projectPath, "setup.py")); err == nil {
		commands = append(commands, pythonCommands()...)
	} else if _, err := os.Stat(filepath.Join(projectPath, "pyproject.toml")); err == nil {
		commands = append(commands, pythonCommands()...)
	} else if _, err := os.Stat(filepath.Join(projectPath, "requirements.txt")); err == nil {
		commands = append(commands, pythonCommands()...)
	}

	// Check for Ruby
	if _, err := os.Stat(filepath.Join(projectPath, "Gemfile")); err == nil {
		commands = append(commands, rubyCommands()...)
	}

	// Check for PHP
	if _, err := os.Stat(filepath.Join(projectPath, "composer.json")); err == nil {
		commands = append(commands, phpCommands()...)
	}

	// Check for Flutter
	if _, err := os.Stat(filepath.Join(projectPath, "pubspec.yaml")); err == nil {
		commands = append(commands, flutterCommands()...)
	}

	// Add VS Code tasks if available
	vscodeCommands := detectVSCodeTasks(projectPath)
	commands = append(commands, vscodeCommands...)

	return commands
}

// DetectMonorepo detects if the project is a monorepo and returns info
func DetectMonorepo(projectPath string) (*MonorepoInfo, error) {
	// Check for pnpm workspace
	if _, err := os.Stat(filepath.Join(projectPath, "pnpm-workspace.yaml")); err == nil {
		return detectPnpmWorkspace(projectPath)
	}

	// Check for npm/yarn workspaces in package.json
	if pkg, err := ParsePackageJSON(filepath.Join(projectPath, "package.json")); err == nil {
		if workspaces := extractWorkspaces(pkg); len(workspaces) > 0 {
			mgr := DetectPackageManager(projectPath)
			repoType := "npm-workspaces"
			if mgr == Yarn {
				repoType = "yarn-workspaces"
			}
			return &MonorepoInfo{
				Type:       repoType,
				Root:       projectPath,
				Workspaces: workspaces,
				Packages:   discoverPackages(projectPath, workspaces),
			}, nil
		}
	}

	// Check for Lerna
	if _, err := os.Stat(filepath.Join(projectPath, "lerna.json")); err == nil {
		return detectLernaWorkspace(projectPath)
	}

	// Check for Nx
	if _, err := os.Stat(filepath.Join(projectPath, "nx.json")); err == nil {
		return detectNxWorkspace(projectPath)
	}

	// Check for Rush
	if _, err := os.Stat(filepath.Join(projectPath, "rush.json")); err == nil {
		return detectRushWorkspace(projectPath)
	}

	return nil, fmt.Errorf("no monorepo configuration found")
}

// Helper functions

func hasWailsConfig(projectPath string) bool {
	_, err1 := os.Stat(filepath.Join(projectPath, "wails.json"))
	_, err2 := os.Stat(filepath.Join(projectPath, "build/appicon.png"))
	return err1 == nil || err2 == nil
}

func isElectronProject(pkg *PackageJSON) bool {
	// Check dependencies
	deps := make(map[string]bool)
	for dep := range pkg.Dependencies {
		deps[dep] = true
	}
	for dep := range pkg.DevDependencies {
		deps[dep] = true
	}
	
	return deps["electron"] || deps["electron-builder"] || deps["electron-packager"]
}

func hasDotNetProject(projectPath string) bool {
	// Check for .csproj, .fsproj, .vbproj files
	patterns := []string{"*.csproj", "*.fsproj", "*.vbproj", "*.sln"}
	for _, pattern := range patterns {
		matches, _ := filepath.Glob(filepath.Join(projectPath, pattern))
		if len(matches) > 0 {
			return true
		}
	}
	return false
}

func extractWorkspaces(pkg *PackageJSON) []string {
	// Check for workspaces field (can be array or object with packages array)
	if pkg.Workspaces != nil {
		switch v := pkg.Workspaces.(type) {
		case []interface{}:
			var workspaces []string
			for _, ws := range v {
				if s, ok := ws.(string); ok {
					workspaces = append(workspaces, s)
				}
			}
			return workspaces
		case map[string]interface{}:
			if packages, ok := v["packages"].([]interface{}); ok {
				var workspaces []string
				for _, ws := range packages {
					if s, ok := ws.(string); ok {
						workspaces = append(workspaces, s)
					}
				}
				return workspaces
			}
		}
	}
	return nil
}

func discoverPackages(root string, workspaces []string) []PackageInfo {
	var packages []PackageInfo
	
	for _, ws := range workspaces {
		// Handle glob patterns
		if strings.Contains(ws, "*") {
			matches, _ := filepath.Glob(filepath.Join(root, ws))
			for _, match := range matches {
				if info, err := os.Stat(match); err == nil && info.IsDir() {
					if pkg := loadPackageInfo(match); pkg != nil {
						packages = append(packages, *pkg)
					}
				}
			}
		} else {
			// Direct path
			pkgPath := filepath.Join(root, ws)
			if pkg := loadPackageInfo(pkgPath); pkg != nil {
				packages = append(packages, *pkg)
			}
		}
	}
	
	return packages
}

func loadPackageInfo(pkgPath string) *PackageInfo {
	pkg, err := ParsePackageJSON(filepath.Join(pkgPath, "package.json"))
	if err != nil {
		return nil
	}
	
	// Check for lock files
	hasLock := false
	lockFiles := []string{"package-lock.json", "yarn.lock", "pnpm-lock.yaml", "bun.lockb"}
	for _, lockFile := range lockFiles {
		if _, err := os.Stat(filepath.Join(pkgPath, lockFile)); err == nil {
			hasLock = true
			break
		}
	}
	
	return &PackageInfo{
		Path:        pkgPath,
		Name:        pkg.Name,
		Scripts:     pkg.Scripts,
		HasLockFile: hasLock,
	}
}

// Command generators for different project types

func wailsCommands() []ExecutableCommand {
	return []ExecutableCommand{
		{
			Name:        "wails dev",
			Command:     "wails",
			Args:        []string{"dev"},
			Description: "Run Wails in development mode",
			Category:    "Wails",
			ProjectType: ProjectTypeWails,
			Priority:    90,
		},
		{
			Name:        "wails build",
			Command:     "wails",
			Args:        []string{"build"},
			Description: "Build Wails application",
			Category:    "Wails",
			ProjectType: ProjectTypeWails,
			Priority:    85,
		},
	}
}

func electronCommands() []ExecutableCommand {
	return []ExecutableCommand{
		{
			Name:        "electron .",
			Command:     "electron",
			Args:        []string{"."},
			Description: "Run Electron app",
			Category:    "Electron",
			ProjectType: ProjectTypeElectron,
			Priority:    90,
		},
	}
}

func goCommands() []ExecutableCommand {
	return []ExecutableCommand{
		{
			Name:        "go run .",
			Command:     "go",
			Args:        []string{"run", "."},
			Description: "Run Go application",
			Category:    "Go",
			ProjectType: ProjectTypeGo,
			Priority:    95,
		},
		{
			Name:        "go build",
			Command:     "go",
			Args:        []string{"build"},
			Description: "Build Go application",
			Category:    "Go",
			ProjectType: ProjectTypeGo,
			Priority:    90,
		},
		{
			Name:        "go test",
			Command:     "go",
			Args:        []string{"test", "./..."},
			Description: "Run Go tests",
			Category:    "Go",
			ProjectType: ProjectTypeGo,
			Priority:    85,
		},
		{
			Name:        "go mod tidy",
			Command:     "go",
			Args:        []string{"mod", "tidy"},
			Description: "Tidy Go modules",
			Category:    "Go",
			ProjectType: ProjectTypeGo,
			Priority:    80,
		},
	}
}

func rustCommands() []ExecutableCommand {
	return []ExecutableCommand{
		{
			Name:        "cargo run",
			Command:     "cargo",
			Args:        []string{"run"},
			Description: "Run Rust application",
			Category:    "Rust",
			ProjectType: ProjectTypeRust,
			Priority:    95,
		},
		{
			Name:        "cargo build",
			Command:     "cargo",
			Args:        []string{"build"},
			Description: "Build Rust application",
			Category:    "Rust",
			ProjectType: ProjectTypeRust,
			Priority:    90,
		},
		{
			Name:        "cargo test",
			Command:     "cargo",
			Args:        []string{"test"},
			Description: "Run Rust tests",
			Category:    "Rust",
			ProjectType: ProjectTypeRust,
			Priority:    85,
		},
		{
			Name:        "cargo check",
			Command:     "cargo",
			Args:        []string{"check"},
			Description: "Check Rust code",
			Category:    "Rust",
			ProjectType: ProjectTypeRust,
			Priority:    80,
		},
	}
}

func gradleCommands() []ExecutableCommand {
	return []ExecutableCommand{
		{
			Name:        "gradle run",
			Command:     "./gradlew",
			Args:        []string{"run"},
			Description: "Run Gradle application",
			Category:    "Gradle",
			ProjectType: ProjectTypeJava,
			Priority:    95,
		},
		{
			Name:        "gradle build",
			Command:     "./gradlew",
			Args:        []string{"build"},
			Description: "Build Gradle project",
			Category:    "Gradle",
			ProjectType: ProjectTypeJava,
			Priority:    90,
		},
		{
			Name:        "gradle test",
			Command:     "./gradlew",
			Args:        []string{"test"},
			Description: "Run Gradle tests",
			Category:    "Gradle",
			ProjectType: ProjectTypeJava,
			Priority:    85,
		},
		{
			Name:        "gradle bootRun",
			Command:     "./gradlew",
			Args:        []string{"bootRun"},
			Description: "Run Spring Boot application",
			Category:    "Gradle",
			ProjectType: ProjectTypeJava,
			Priority:    93,
		},
	}
}

func mavenCommands() []ExecutableCommand {
	return []ExecutableCommand{
		{
			Name:        "mvn spring-boot:run",
			Command:     "mvn",
			Args:        []string{"spring-boot:run"},
			Description: "Run Spring Boot application",
			Category:    "Maven",
			ProjectType: ProjectTypeJava,
			Priority:    95,
		},
		{
			Name:        "mvn compile",
			Command:     "mvn",
			Args:        []string{"compile"},
			Description: "Compile Maven project",
			Category:    "Maven",
			ProjectType: ProjectTypeJava,
			Priority:    90,
		},
		{
			Name:        "mvn test",
			Command:     "mvn",
			Args:        []string{"test"},
			Description: "Run Maven tests",
			Category:    "Maven",
			ProjectType: ProjectTypeJava,
			Priority:    85,
		},
		{
			Name:        "mvn package",
			Command:     "mvn",
			Args:        []string{"package"},
			Description: "Package Maven project",
			Category:    "Maven",
			ProjectType: ProjectTypeJava,
			Priority:    88,
		},
	}
}

func dotnetCommands() []ExecutableCommand {
	return []ExecutableCommand{
		{
			Name:        "dotnet run",
			Command:     "dotnet",
			Args:        []string{"run"},
			Description: "Run .NET application",
			Category:    ".NET",
			ProjectType: ProjectTypeDotNet,
			Priority:    95,
		},
		{
			Name:        "dotnet build",
			Command:     "dotnet",
			Args:        []string{"build"},
			Description: "Build .NET project",
			Category:    ".NET",
			ProjectType: ProjectTypeDotNet,
			Priority:    90,
		},
		{
			Name:        "dotnet test",
			Command:     "dotnet",
			Args:        []string{"test"},
			Description: "Run .NET tests",
			Category:    ".NET",
			ProjectType: ProjectTypeDotNet,
			Priority:    85,
		},
		{
			Name:        "dotnet watch",
			Command:     "dotnet",
			Args:        []string{"watch", "run"},
			Description: "Run .NET app with hot reload",
			Category:    ".NET",
			ProjectType: ProjectTypeDotNet,
			Priority:    93,
		},
	}
}

func pythonCommands() []ExecutableCommand {
	return []ExecutableCommand{
		{
			Name:        "python main.py",
			Command:     "python",
			Args:        []string{"main.py"},
			Description: "Run Python main script",
			Category:    "Python",
			ProjectType: ProjectTypePython,
			Priority:    95,
		},
		{
			Name:        "python -m pytest",
			Command:     "python",
			Args:        []string{"-m", "pytest"},
			Description: "Run Python tests",
			Category:    "Python",
			ProjectType: ProjectTypePython,
			Priority:    85,
		},
		{
			Name:        "pip install -r requirements.txt",
			Command:     "pip",
			Args:        []string{"install", "-r", "requirements.txt"},
			Description: "Install Python dependencies",
			Category:    "Python",
			ProjectType: ProjectTypePython,
			Priority:    80,
		},
	}
}

func rubyCommands() []ExecutableCommand {
	return []ExecutableCommand{
		{
			Name:        "rails server",
			Command:     "rails",
			Args:        []string{"server"},
			Description: "Start Rails server",
			Category:    "Ruby",
			ProjectType: ProjectTypeRuby,
			Priority:    95,
		},
		{
			Name:        "bundle exec rspec",
			Command:     "bundle",
			Args:        []string{"exec", "rspec"},
			Description: "Run Ruby tests",
			Category:    "Ruby",
			ProjectType: ProjectTypeRuby,
			Priority:    85,
		},
		{
			Name:        "bundle install",
			Command:     "bundle",
			Args:        []string{"install"},
			Description: "Install Ruby dependencies",
			Category:    "Ruby",
			ProjectType: ProjectTypeRuby,
			Priority:    80,
		},
	}
}

func phpCommands() []ExecutableCommand {
	return []ExecutableCommand{
		{
			Name:        "php artisan serve",
			Command:     "php",
			Args:        []string{"artisan", "serve"},
			Description: "Start Laravel server",
			Category:    "PHP",
			ProjectType: ProjectTypePhp,
			Priority:    95,
		},
		{
			Name:        "composer install",
			Command:     "composer",
			Args:        []string{"install"},
			Description: "Install PHP dependencies",
			Category:    "PHP",
			ProjectType: ProjectTypePhp,
			Priority:    80,
		},
		{
			Name:        "phpunit",
			Command:     "./vendor/bin/phpunit",
			Args:        []string{},
			Description: "Run PHP tests",
			Category:    "PHP",
			ProjectType: ProjectTypePhp,
			Priority:    85,
		},
	}
}

func flutterCommands() []ExecutableCommand {
	return []ExecutableCommand{
		{
			Name:        "flutter run",
			Command:     "flutter",
			Args:        []string{"run"},
			Description: "Run Flutter app",
			Category:    "Flutter",
			ProjectType: ProjectTypeFlutter,
			Priority:    95,
		},
		{
			Name:        "flutter build",
			Command:     "flutter",
			Args:        []string{"build"},
			Description: "Build Flutter app",
			Category:    "Flutter",
			ProjectType: ProjectTypeFlutter,
			Priority:    90,
		},
		{
			Name:        "flutter test",
			Command:     "flutter",
			Args:        []string{"test"},
			Description: "Run Flutter tests",
			Category:    "Flutter",
			ProjectType: ProjectTypeFlutter,
			Priority:    85,
		},
	}
}

// VS Code task detection
func detectVSCodeTasks(projectPath string) []ExecutableCommand {
	var commands []ExecutableCommand
	
	tasksFile := filepath.Join(projectPath, ".vscode", "tasks.json")
	data, err := os.ReadFile(tasksFile)
	if err != nil {
		return commands
	}
	
	var tasks struct {
		Version string `json:"version"`
		Tasks   []struct {
			Label   string   `json:"label"`
			Type    string   `json:"type"`
			Command string   `json:"command"`
			Args    []string `json:"args"`
			Group   struct {
				Kind      string `json:"kind"`
				IsDefault bool   `json:"isDefault"`
			} `json:"group"`
		} `json:"tasks"`
	}
	
	if err := json.Unmarshal(data, &tasks); err != nil {
		return commands
	}
	
	for _, task := range tasks.Tasks {
		if task.Type == "shell" && task.Command != "" {
			priority := 70
			if task.Group.IsDefault {
				priority = 75
			}
			
			commands = append(commands, ExecutableCommand{
				Name:        task.Label,
				Command:     task.Command,
				Args:        task.Args,
				Description: fmt.Sprintf("VS Code Task: %s", task.Label),
				Category:    "VS Code Tasks",
				ProjectType: ProjectTypeNode, // Default, could be improved
				Priority:    priority,
			})
		}
	}
	
	return commands
}

// Monorepo detection functions

func detectPnpmWorkspace(projectPath string) (*MonorepoInfo, error) {
	data, err := os.ReadFile(filepath.Join(projectPath, "pnpm-workspace.yaml"))
	if err != nil {
		return nil, err
	}
	
	// Simple YAML parsing for packages array
	lines := strings.Split(string(data), "\n")
	var workspaces []string
	inPackages := false
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "packages:" {
			inPackages = true
			continue
		}
		if inPackages && strings.HasPrefix(trimmed, "- ") {
			ws := strings.TrimPrefix(trimmed, "- ")
			ws = strings.Trim(ws, "'\"")
			workspaces = append(workspaces, ws)
		} else if inPackages && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			break
		}
	}
	
	return &MonorepoInfo{
		Type:       "pnpm",
		Root:       projectPath,
		Workspaces: workspaces,
		Packages:   discoverPackages(projectPath, workspaces),
	}, nil
}

func detectLernaWorkspace(projectPath string) (*MonorepoInfo, error) {
	data, err := os.ReadFile(filepath.Join(projectPath, "lerna.json"))
	if err != nil {
		return nil, err
	}
	
	var lernaConfig struct {
		Packages []string `json:"packages"`
		Version  string   `json:"version"`
	}
	
	if err := json.Unmarshal(data, &lernaConfig); err != nil {
		return nil, err
	}
	
	return &MonorepoInfo{
		Type:       "lerna",
		Root:       projectPath,
		Workspaces: lernaConfig.Packages,
		Packages:   discoverPackages(projectPath, lernaConfig.Packages),
	}, nil
}

func detectNxWorkspace(projectPath string) (*MonorepoInfo, error) {
	// For Nx, we need to check workspace.json or project.json files
	workspaces := []string{"apps/*", "libs/*", "packages/*"}
	
	// Check if custom paths exist
	if data, err := os.ReadFile(filepath.Join(projectPath, "workspace.json")); err == nil {
		var workspace struct {
			Projects map[string]string `json:"projects"`
		}
		if json.Unmarshal(data, &workspace) == nil {
			workspaces = []string{}
			for _, path := range workspace.Projects {
				workspaces = append(workspaces, path)
			}
		}
	}
	
	return &MonorepoInfo{
		Type:       "nx",
		Root:       projectPath,
		Workspaces: workspaces,
		Packages:   discoverPackages(projectPath, workspaces),
	}, nil
}

func detectRushWorkspace(projectPath string) (*MonorepoInfo, error) {
	data, err := os.ReadFile(filepath.Join(projectPath, "rush.json"))
	if err != nil {
		return nil, err
	}
	
	var rushConfig struct {
		Projects []struct {
			PackageName string `json:"packageName"`
			ProjectFolder string `json:"projectFolder"`
		} `json:"projects"`
	}
	
	if err := json.Unmarshal(data, &rushConfig); err != nil {
		return nil, err
	}
	
	var workspaces []string
	for _, proj := range rushConfig.Projects {
		workspaces = append(workspaces, proj.ProjectFolder)
	}
	
	return &MonorepoInfo{
		Type:       "rush",
		Root:       projectPath,
		Workspaces: workspaces,
		Packages:   discoverPackages(projectPath, workspaces),
	}, nil
}