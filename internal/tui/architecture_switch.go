package tui

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbletea"
	"github.com/standardbeagle/brummer/internal/tui_redux"
)

// ArchitectureManager handles switching between TUI architectures
type ArchitectureManager struct {
	useRedux bool
}

// NewArchitectureManager creates a new architecture manager
func NewArchitectureManager() *ArchitectureManager {
	return &ArchitectureManager{
		useRedux: IsReduxArchitectureEnabled(),
	}
}

// RunTUI runs the appropriate TUI architecture based on feature flags
func (am *ArchitectureManager) RunTUI() error {
	architecture := GetArchitectureName()
	fmt.Printf("Starting Brummer with %s architecture...\n", architecture)

	if am.useRedux {
		return am.runReduxTUI()
	}
	return am.runLegacyTUI()
}

// runReduxTUI runs the Redux/Elm TUI
func (am *ArchitectureManager) runReduxTUI() error {
	fmt.Println("Initializing Redux/Elm architecture...")

	// Create and run the Redux application
	app := redux.NewReduxApp()
	if err := app.Run(); err != nil {
		return fmt.Errorf("Redux TUI failed: %w", err)
	}

	return nil
}

// runLegacyTUI runs the legacy TUI
func (am *ArchitectureManager) runLegacyTUI() error {
	fmt.Println("Initializing legacy architecture...")

	// This would run the original TUI
	// For now, provide a placeholder that shows the legacy option
	p := tea.NewProgram(legacyModel{}, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("Legacy TUI failed: %w", err)
	}

	return nil
}

// SwitchArchitecture switches the default architecture
func (am *ArchitectureManager) SwitchArchitecture(toRedux bool) error {
	am.useRedux = toRedux

	if toRedux {
		EnableReduxArchitecture()
		fmt.Println("Switched default architecture to Redux/Elm")
	} else {
		EnableLegacyArchitecture()
		fmt.Println("Switched default architecture to Legacy")
	}

	// Save the setting to a configuration file
	return am.saveArchitectureSetting(toRedux)
}

// saveArchitectureSetting saves the architecture preference to a config file
func (am *ArchitectureManager) saveArchitectureSetting(useRedux bool) error {
	configPath := ".brum.toml"

	// This would normally marshal to TOML format
	// For now, just create a simple setting file
	content := ""
	if useRedux {
		content += "# Brummer Configuration\n"
		content += "architecture = \"redux\"\n"
	} else {
		content += "# Brummer Configuration\n"
		content += "architecture = \"legacy\"\n"
	}

	return os.WriteFile(configPath, []byte(content), 0644)
}

// RunArchitectureComparison runs a comparison between architectures
func (am *ArchitectureManager) RunArchitectureComparison() error {
	fmt.Println("Running architecture comparison...")

	// TODO: Implement A/B testing
	// result, err := RunABTest()
	// if err != nil {
	// 	return fmt.Errorf("A/B test failed: %w", err)
	// }

	// // Auto-switch if Redux is significantly better
	// if result.ReduxFaster && result.SpeedImprovement > 30 {
	// 	fmt.Println("\nAuto-switching to Redux architecture due to significant performance improvement...")
	// 	return am.SwitchArchitecture(true)
	// }

	fmt.Println("\nA/B testing not yet implemented.")
	fmt.Println("\nKeeping current architecture setting.")
	return nil
}

// ValidateCurrentArchitecture validates the current architecture meets requirements
func (am *ArchitectureManager) ValidateCurrentArchitecture() error {
	architecture := "legacy"
	if am.useRedux {
		architecture = "redux"
	}

	fmt.Printf("Validating %s architecture...\n", architecture)
	// TODO: Implement performance validation
	// return ValidateArchitecturePerformance(architecture)
	fmt.Println("Architecture validation not yet implemented.")
	return nil
}

// GetArchitectureInfo returns information about the current architecture
func (am *ArchitectureManager) GetArchitectureInfo() map[string]interface{} {
	return map[string]interface{}{
		"current":     GetArchitectureName(),
		"useRedux":    am.useRedux,
		"envRedux":    os.Getenv("REDUX_ARCHITECTURE") != "",
		"envDefault":  os.Getenv("DEFAULT_ARCHITECTURE"),
		"featureFlag":  IsReduxArchitectureEnabled(),
	}
}

// legacyModel represents a simple legacy TUI model for comparison
type legacyModel struct {
	legacy string
	count  int
}

func (m legacyModel) Init() tea.Cmd {
	return nil
}

func (m legacyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case " ":
			m.count++
		}
	}
	return m, nil
}

func (m legacyModel) View() string {
	return fmt.Sprintf(
		"Legacy TUI Architecture\n\n"+
			"Count: %d\n\n"+
			"Press Space to increment, q to quit",
		m.count,
	)
}