package tui

import (
	"testing"
	"time"

	"github.com/standardbeagle/brummer/internal/tui/components"
	"github.com/standardbeagle/brummer/internal/tui/messages"
)

// createTestRegistry creates a test command registry with sample commands
func createTestRegistry() *components.CommandRegistry {
	registry := components.NewCommandRegistry()

	// Add test commands
	testCommands := []messages.Command{
		{
			Name:        "clear",
			Description: "Clear logs, errors, or web requests",
			Arguments: []messages.CommandArgument{
				{
					Name:         "target",
					Description:  "What to clear",
					Required:     false,
					Suggestions:  []string{"all", "logs", "errors", "web"},
					DefaultValue: "all",
				},
			},
		},
		{
			Name:        "show",
			Description: "Show processes matching pattern",
			Arguments: []messages.CommandArgument{
				{
					Name:        "pattern",
					Description: "Pattern to match",
					Required:    false,
					Suggestions: []string{"running", "stopped", "all"},
				},
			},
		},
		{
			Name:        "hide",
			Description: "Hide processes matching pattern",
			Arguments: []messages.CommandArgument{
				{
					Name:        "pattern",
					Description: "Pattern to match",
					Required:    false,
					Suggestions: []string{"running", "stopped", "all"},
				},
			},
		},
		{
			Name:        "proxy",
			Description: "Set proxy URL for web access",
			Arguments: []messages.CommandArgument{
				{
					Name:        "url",
					Description: "URL to proxy",
					Required:    true,
					Suggestions: []string{"http://localhost:3000", "http://localhost:8080"},
				},
			},
		},
		{
			Name:        "ai",
			Description: "Start AI coding session",
			Arguments: []messages.CommandArgument{
				{
					Name:        "provider",
					Description: "AI provider to use",
					Required:    false,
					Suggestions: []string{"claude", "sonnet", "opus", "aider", "opencode"},
				},
			},
		},
	}

	for _, cmd := range testCommands {
		registry.Register(cmd)
	}

	return registry
}

// BenchmarkOriginalCommandInput tests the original command input performance
func BenchmarkOriginalCommandInput(b *testing.B) {
	registry := createTestRegistry()
	component := components.NewCommandInputComponent(registry)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Simulate typing a command with multiple keystrokes
		component.Open()

		// Type "clear " (6 keystrokes)
		testInputs := []string{"c", "l", "e", "a", "r", " "}
		for _, input := range testInputs {
			component.Update(testKeyMsg(input))
		}

		// Type argument (4 keystrokes)
		argInputs := []string{"a", "l", "l"}
		for _, input := range argInputs {
			component.Update(testKeyMsg(input))
		}

		component.Close()
	}
}

// BenchmarkOptimizedCommandInput tests the optimized command input performance
func BenchmarkOptimizedCommandInput(b *testing.B) {
	registry := createTestRegistry()
	component := components.NewOptimizedCommandInputComponent(registry)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Simulate typing a command with multiple keystrokes
		component.Open()

		// Type "clear " (6 keystrokes)
		testInputs := []string{"c", "l", "e", "a", "r", " "}
		for _, input := range testInputs {
			component.Update(testKeyMsg(input))
		}

		// Type argument (4 keystrokes)
		argInputs := []string{"a", "l", "l"}
		for _, input := range argInputs {
			component.Update(testKeyMsg(input))
		}

		// Wait for debounce to complete
		time.Sleep(10 * time.Millisecond)

		component.Close()
	}
}

// BenchmarkCommandInputLatency tests the latency of individual keystrokes
func BenchmarkCommandInputLatency(b *testing.B) {
	registry := createTestRegistry()

	b.Run("Original", func(b *testing.B) {
		component := components.NewCommandInputComponent(registry)
		component.Open()

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			component.Update(testKeyMsg("c"))
			component.Update(testKeyMsg("l"))
			component.Update(testKeyMsg("e"))
		}

		component.Close()
	})

	b.Run("Optimized", func(b *testing.B) {
		component := components.NewOptimizedCommandInputComponent(registry)
		component.Open()

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			component.Update(testKeyMsg("c"))
			component.Update(testKeyMsg("l"))
			component.Update(testKeyMsg("e"))
		}

		component.Close()
	})
}

// BenchmarkSuggestionCaching tests the effectiveness of suggestion caching
func BenchmarkSuggestionCaching(b *testing.B) {
	registry := createTestRegistry()
	component := components.NewOptimizedCommandInputComponent(registry)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Repeatedly type the same prefix to test cache hit performance
		component.Open()
		component.Update(testKeyMsg("c"))
		component.Update(testKeyMsg("l"))
		component.Close()
	}
}

// testKeyMsg creates a test key message for the given character
func testKeyMsg(char string) interface{} {
	// This would normally be a tea.KeyMsg, but we'll use a simple string for testing
	return struct{ string }{char}
}

// BenchmarkMemoryUsage tests memory usage patterns
func BenchmarkMemoryUsage(b *testing.B) {
	registry := createTestRegistry()

	b.Run("Original", func(b *testing.B) {
		component := components.NewCommandInputComponent(registry)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			component.Open()
			component.Update(testKeyMsg("c"))
			component.Update(testKeyMsg("l"))
			component.Update(testKeyMsg("e"))
			component.Update(testKeyMsg("a"))
			component.Update(testKeyMsg("r"))
			component.Close()
		}
	})

	b.Run("Optimized", func(b *testing.B) {
		component := components.NewOptimizedCommandInputComponent(registry)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			component.Open()
			component.Update(testKeyMsg("c"))
			component.Update(testKeyMsg("l"))
			component.Update(testKeyMsg("e"))
			component.Update(testKeyMsg("a"))
			component.Update(testKeyMsg("r"))
			component.Close()
		}
	})
}