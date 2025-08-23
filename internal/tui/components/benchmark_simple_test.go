package components

import (
	"testing"
	"time"
	
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func init() {
	// Force color output for benchmarks
	lipgloss.SetColorProfile(termenv.ANSI256)
}

// BenchmarkMinimalShellRender benchmarks the minimal shell rendering
func BenchmarkMinimalShellRender(b *testing.B) {
	shell := NewMinimalShell()
	shell.Init()
	
	// Set window size
	sizeMsg := tea.WindowSizeMsg{Width: 80, Height: 24}
	model, _ := shell.Update(sizeMsg)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = model.View()
	}
}

// BenchmarkMinimalShellViewSwitch benchmarks view switching
func BenchmarkMinimalShellViewSwitch(b *testing.B) {
	shell := NewMinimalShell()
	shell.Init()
	
	// Set window size
	sizeMsg := tea.WindowSizeMsg{Width: 80, Height: 24}
	model, _ := shell.Update(sizeMsg)
	
	// Prepare key messages for switching views
	keys := []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune{'1'}},
		{Type: tea.KeyRunes, Runes: []rune{'2'}},
		{Type: tea.KeyRunes, Runes: []rune{'3'}},
		{Type: tea.KeyRunes, Runes: []rune{'4'}},
		{Type: tea.KeyRunes, Runes: []rune{'5'}},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := keys[i%len(keys)]
		model, _ = model.Update(msg)
	}
}

// BenchmarkMinimalShellTabNavigation benchmarks tab key navigation
func BenchmarkMinimalShellTabNavigation(b *testing.B) {
	shell := NewMinimalShell()
	shell.Init()
	
	// Set window size
	sizeMsg := tea.WindowSizeMsg{Width: 80, Height: 24}
	model, _ := shell.Update(sizeMsg)
	
	tabMsg := tea.KeyMsg{Type: tea.KeyTab}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		model, _ = model.Update(tabMsg)
	}
}

// BenchmarkBoxLayoutRender benchmarks the box layout rendering
func BenchmarkBoxLayoutRender(b *testing.B) {
	layout := NewBoxLayout(Vertical)
	layout.SetSize(80, 24)
	
	// Add children
	for i := 0; i < 5; i++ {
		child := &BaseComponent{width: 80, height: 4}
		layout.AddChild(child, 0.2)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = layout.View()
	}
}

// BenchmarkBorderWindowRender benchmarks border window rendering
func BenchmarkBorderWindowRender(b *testing.B) {
	content := &BaseComponent{width: 78, height: 22}
	window := NewBorderWindow(content)
	window.SetSize(80, 24)
	window.SetBorderStyle(lipgloss.NormalBorder())
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = window.View()
	}
}

// BenchmarkStyleApplication benchmarks lipgloss style application
func BenchmarkStyleApplication(b *testing.B) {
	text := "Sample text for styling benchmark"
	
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		Background(lipgloss.Color("236")).
		Padding(1, 2).
		Border(lipgloss.NormalBorder()).
		Width(40).
		Height(5)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = style.Render(text)
	}
}

// BenchmarkTabHighlighting benchmarks tab highlighting performance
func BenchmarkTabHighlighting(b *testing.B) {
	tabs := []string{"[1] Processes", "[2] Logs", "[3] Errors", "[4] URLs", "[5] Web"}
	activeIndex := 0
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		activeIndex = (activeIndex + 1) % len(tabs)
		
		var tabLine []string
		for j, tab := range tabs {
			style := lipgloss.NewStyle().Padding(0, 1)
			if j == activeIndex {
				style = style.Bold(true).
					Foreground(lipgloss.Color("205")).
					Background(lipgloss.Color("236"))
			} else {
				style = style.Foreground(lipgloss.Color("241"))
			}
			tabLine = append(tabLine, style.Render(tab))
		}
		
		_ = lipgloss.JoinHorizontal(lipgloss.Top, tabLine...)
	}
}

// TestPerformanceBaseline establishes performance baselines
func TestPerformanceBaseline(t *testing.T) {
	tests := []struct {
		name      string
		benchmark func() time.Duration
		maxTime   time.Duration
	}{
		{
			name: "MinimalShellRender",
			benchmark: func() time.Duration {
				shell := NewMinimalShell()
				sizeMsg := tea.WindowSizeMsg{Width: 80, Height: 24}
				model, _ := shell.Update(sizeMsg)
				
				start := time.Now()
				for i := 0; i < 1000; i++ {
					_ = model.View()
				}
				return time.Since(start)
			},
			maxTime: 100 * time.Millisecond, // 0.1ms per render
		},
		{
			name: "ViewSwitching",
			benchmark: func() time.Duration {
				shell := NewMinimalShell()
				sizeMsg := tea.WindowSizeMsg{Width: 80, Height: 24}
				model, _ := shell.Update(sizeMsg)
				
				start := time.Now()
				m := model
				for i := 0; i < 1000; i++ {
					msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{rune('1' + (i % 5))}}
					m, _ = m.Update(msg)
				}
				return time.Since(start)
			},
			maxTime: 50 * time.Millisecond, // 0.05ms per switch
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration := tt.benchmark()
			if duration > tt.maxTime {
				t.Errorf("%s took %v, expected less than %v", tt.name, duration, tt.maxTime)
			}
			t.Logf("%s: %v (%.3f ms/op)", tt.name, duration, float64(duration.Milliseconds())/1000.0)
		})
	}
}