package tui

import (
	"os"
	"strconv"
)

// Feature flags for TUI architecture
const (
	// ReduxArchitecture enables the new Redux/Elm architecture
	ReduxArchitecture = "REDUX_ARCHITECTURE"

	// DefaultArchitecture specifies which architecture to use by default
	DefaultArchitecture = "DEFAULT_ARCHITECTURE"
)

// IsReduxArchitectureEnabled checks if Redux architecture is enabled
func IsReduxArchitectureEnabled() bool {
	// Check environment variable first
	if os.Getenv(ReduxArchitecture) != "" {
		enabled, err := strconv.ParseBool(os.Getenv(ReduxArchitecture))
		if err == nil {
			return enabled
		}
	}

	// Check if specified as default
	if os.Getenv(DefaultArchitecture) == "redux" {
		return true
	}

	// Default to old architecture for safety
	return false
}

// IsLegacyArchitectureEnabled checks if legacy architecture is enabled
func IsLegacyArchitectureEnabled() bool {
	return !IsReduxArchitectureEnabled()
}

// EnableReduxArchitecture enables Redux architecture via environment
func EnableReduxArchitecture() {
	os.Setenv(ReduxArchitecture, "true")
}

// EnableLegacyArchitecture enables legacy architecture via environment
func EnableLegacyArchitecture() {
	os.Setenv(ReduxArchitecture, "false")
}

// GetArchitectureName returns the name of the current architecture
func GetArchitectureName() string {
	if IsReduxArchitectureEnabled() {
		return "Redux/Elm"
	}
	return "Legacy"
}