package parser

// CommonScriptPatterns defines common script names and their typical implementations
// across different project types
var CommonScriptPatterns = map[string][]ExecutableCommand{
	"server": {
		{
			Name:        "npm start",
			Command:     "npm",
			Args:        []string{"start"},
			Description: "Start npm server (if start script exists)",
			Category:    "npm",
			ProjectType: ProjectTypeNode,
			Priority:    100,
		},
		{
			Name:        "yarn start", 
			Command:     "yarn",
			Args:        []string{"start"},
			Description: "Start yarn server (if start script exists)",
			Category:    "yarn",
			ProjectType: ProjectTypeNode,
			Priority:    99,
		},
		{
			Name:        "python -m http.server",
			Command:     "python",
			Args:        []string{"-m", "http.server"},
			Description: "Start Python HTTP server",
			Category:    "Python",
			ProjectType: ProjectTypePython,
			Priority:    80,
		},
		{
			Name:        "php -S localhost:8000",
			Command:     "php",
			Args:        []string{"-S", "localhost:8000"},
			Description: "Start PHP built-in server",
			Category:    "PHP",
			ProjectType: ProjectTypePhp,
			Priority:    85,
		},
	},
	"dev": {
		{
			Name:        "npm run dev",
			Command:     "npm",
			Args:        []string{"run", "dev"},
			Description: "Run npm dev script",
			Category:    "npm",
			ProjectType: ProjectTypeNode,
			Priority:    100,
		},
		{
			Name:        "yarn dev",
			Command:     "yarn",
			Args:        []string{"dev"},
			Description: "Run yarn dev script",
			Category:    "yarn",
			ProjectType: ProjectTypeNode,
			Priority:    99,
		},
		{
			Name:        "pnpm dev",
			Command:     "pnpm",
			Args:        []string{"dev"},
			Description: "Run pnpm dev script",
			Category:    "pnpm",
			ProjectType: ProjectTypeNode,
			Priority:    98,
		},
		{
			Name:        "bun dev",
			Command:     "bun",
			Args:        []string{"dev"},
			Description: "Run bun dev script",
			Category:    "bun",
			ProjectType: ProjectTypeNode,
			Priority:    97,
		},
	},
	"test": {
		{
			Name:        "npm test",
			Command:     "npm",
			Args:        []string{"test"},
			Description: "Run npm tests",
			Category:    "npm",
			ProjectType: ProjectTypeNode,
			Priority:    100,
		},
		{
			Name:        "yarn test",
			Command:     "yarn",
			Args:        []string{"test"},
			Description: "Run yarn tests",
			Category:    "yarn",
			ProjectType: ProjectTypeNode,
			Priority:    99,
		},
		{
			Name:        "go test ./...",
			Command:     "go",
			Args:        []string{"test", "./..."},
			Description: "Run Go tests",
			Category:    "Go",
			ProjectType: ProjectTypeGo,
			Priority:    95,
		},
		{
			Name:        "cargo test",
			Command:     "cargo",
			Args:        []string{"test"},
			Description: "Run Rust tests",
			Category:    "Rust",
			ProjectType: ProjectTypeRust,
			Priority:    95,
		},
		{
			Name:        "pytest",
			Command:     "pytest",
			Args:        []string{},
			Description: "Run Python tests",
			Category:    "Python",
			ProjectType: ProjectTypePython,
			Priority:    90,
		},
	},
	"build": {
		{
			Name:        "npm run build",
			Command:     "npm",
			Args:        []string{"run", "build"},
			Description: "Build with npm",
			Category:    "npm",
			ProjectType: ProjectTypeNode,
			Priority:    100,
		},
		{
			Name:        "yarn build",
			Command:     "yarn",
			Args:        []string{"build"},
			Description: "Build with yarn",
			Category:    "yarn",
			ProjectType: ProjectTypeNode,
			Priority:    99,
		},
		{
			Name:        "go build",
			Command:     "go",
			Args:        []string{"build"},
			Description: "Build Go application",
			Category:    "Go",
			ProjectType: ProjectTypeGo,
			Priority:    95,
		},
		{
			Name:        "cargo build",
			Command:     "cargo",
			Args:        []string{"build"},
			Description: "Build Rust application",
			Category:    "Rust",
			ProjectType: ProjectTypeRust,
			Priority:    95,
		},
		{
			Name:        "make",
			Command:     "make",
			Args:        []string{},
			Description: "Build with make",
			Category:    "Make",
			ProjectType: ProjectTypeNode, // Generic
			Priority:    80,
		},
	},
	"start": {
		{
			Name:        "npm start",
			Command:     "npm",
			Args:        []string{"start"},
			Description: "Start with npm",
			Category:    "npm",
			ProjectType: ProjectTypeNode,
			Priority:    100,
		},
		{
			Name:        "yarn start",
			Command:     "yarn",
			Args:        []string{"start"},
			Description: "Start with yarn",
			Category:    "yarn",
			ProjectType: ProjectTypeNode,
			Priority:    99,
		},
		{
			Name:        "docker-compose up",
			Command:     "docker-compose",
			Args:        []string{"up"},
			Description: "Start Docker services",
			Category:    "Docker",
			ProjectType: ProjectTypeDocker,
			Priority:    90,
		},
	},
}

// GetCommonCommand returns a project-appropriate command for a common script name
func GetCommonCommand(scriptName string, projectPath string) *ExecutableCommand {
	commands, exists := CommonScriptPatterns[scriptName]
	if !exists {
		return nil
	}

	// Try to detect project type and return the most appropriate command
	detectedCommands := DetectProjectCommands(projectPath)
	
	// Check if we have a specific project type
	for _, cmd := range detectedCommands {
		for _, commonCmd := range commands {
			if cmd.ProjectType == commonCmd.ProjectType {
				// Found a match for this project type
				return &commonCmd
			}
		}
	}

	// Default to first option if no specific match
	if len(commands) > 0 {
		return &commands[0]
	}

	return nil
}