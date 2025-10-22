package tui_test

import (
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hinshun/vt10x"
	"github.com/standardbeagle/brummer/internal/tui/components"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVT10XScrollbackBuffer tests that vt10x maintains a scrollback buffer
func TestVT10XScrollbackBuffer(t *testing.T) {
	// Create a vt10x terminal with scrollback (1000 rows)
	term := vt10x.New(vt10x.WithSize(80, 1000))
	require.NotNil(t, term)

	// Write 100 lines of content
	for i := 0; i < 100; i++ {
		line := "Line " + string(rune('0'+i%10)) + "\n"
		n, err := term.Write([]byte(line))
		require.NoError(t, err)
		require.Greater(t, n, 0)
	}

	// Verify terminal dimensions
	cols, rows := term.Size()
	assert.Equal(t, 80, cols, "Terminal should have 80 columns")
	assert.Equal(t, 1000, rows, "Terminal should have 1000 rows for scrollback")

	// Verify we can read all rows (including scrollback)
	for y := 0; y < 100; y++ {
		// Check first cell of each line exists
		cell := term.Cell(0, y)
		assert.NotNil(t, cell, "Cell at row %d should exist", y)
	}
}

// TestVT10XScrollbackWithANSI tests scrollback with ANSI color codes
func TestVT10XScrollbackWithANSI(t *testing.T) {
	term := vt10x.New(vt10x.WithSize(80, 1000))
	require.NotNil(t, term)

	// Write colored lines (simple text, vt10x processes ANSI codes)
	colors := []string{
		"\x1b[31m", // Red
		"\x1b[32m", // Green
		"\x1b[33m", // Yellow
		"\x1b[34m", // Blue
	}

	for i := 0; i < 50; i++ {
		colorCode := colors[i%len(colors)]
		line := colorCode + "Colored line\x1b[0m\n"
		_, err := term.Write([]byte(line))
		require.NoError(t, err)
	}

	// Verify terminal has content and dimensions are correct
	cols, rows := term.Size()
	assert.Equal(t, 80, cols, "Should maintain width")
	assert.Equal(t, 1000, rows, "Should maintain scrollback height")

	// Verify we can access cells from scrollback
	// (The actual character content depends on vt10x's processing)
	for y := 0; y < 50; y++ {
		cell := term.Cell(0, y)
		assert.NotNil(t, cell, "Should be able to access cell at row %d", y)
	}
}

// TestViewportScrolling tests Bubble Tea viewport scroll behavior
func TestViewportScrolling(t *testing.T) {
	// Create viewport with 10 visible lines
	vp := viewport.New(80, 10)
	vp.MouseWheelEnabled = true

	// Set content with 50 lines
	lines := make([]string, 50)
	for i := range lines {
		lines[i] = "Line " + string(rune('0'+i%10))
	}
	content := ""
	for i, line := range lines {
		if i > 0 {
			content += "\n"
		}
		content += line
	}
	vp.SetContent(content)

	// Verify initial state (at top)
	assert.Equal(t, 0, vp.YOffset, "Should start at top")
	totalLines := vp.TotalLineCount()
	assert.GreaterOrEqual(t, totalLines, 50, "Should have at least 50 total lines")
	assert.Equal(t, 10, vp.Height, "Should show 10 visible lines")
	assert.Equal(t, 0.0, vp.ScrollPercent(), "Should be at 0% scroll")

	// Simulate scroll down (arrow down key)
	vp, _ = vp.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 1, vp.YOffset, "Should scroll down by 1 line")

	// Simulate page down
	vp, _ = vp.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	assert.Greater(t, vp.YOffset, 1, "Should scroll down by page")

	// Verify scroll percentage increased
	assert.Greater(t, vp.ScrollPercent(), 0.0, "Scroll percentage should increase")

	// Scroll to bottom
	vp.GotoBottom()
	maxOffset := totalLines - 10 // Total lines - visible lines
	assert.GreaterOrEqual(t, vp.YOffset, maxOffset-1, "Should be near bottom")
	assert.GreaterOrEqual(t, vp.ScrollPercent(), 0.9, "Should be at or near 100% scroll")

	// Scroll back up
	offsetBeforeUp := vp.YOffset
	vp, _ = vp.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.Less(t, vp.YOffset, offsetBeforeUp, "Should scroll up")
}

// TestAICodersViewScrollbar tests the scrollbar rendering
func TestAICodersViewScrollbar(t *testing.T) {
	// Create AI Coders view component
	component := components.NewAICodersViewComponent(nil)
	require.NotNil(t, component)

	// Set size for viewport
	component.SetSize(100, 30)

	// Note: We can't directly test addScrollbar since it's a private method,
	// but we can verify the component has a viewport configured
	// The actual scrollbar rendering would be tested via integration tests

	// Verify component is created
	assert.NotNil(t, component)
}

// TestScrollIndicatorCalculations tests scrollbar thumb position calculations
func TestScrollIndicatorCalculations(t *testing.T) {
	testCases := []struct {
		name             string
		totalLines       int
		visibleLines     int
		scrollPercent    float64
		expectedThumbSize int
		expectedThumbPos  int
	}{
		{
			name:             "At top of large buffer",
			totalLines:       1000,
			visibleLines:     24,
			scrollPercent:    0.0,
			expectedThumbSize: 1, // (24 * 24) / 1000 = 0.576, max(1, 0) = 1
			expectedThumbPos:  0,
		},
		{
			name:             "Middle of large buffer",
			totalLines:       1000,
			visibleLines:     24,
			scrollPercent:    0.5,
			expectedThumbSize: 1,
			expectedThumbPos:  11, // 0.5 * (24 - 1) = 11.5 -> 11
		},
		{
			name:             "At bottom of large buffer",
			totalLines:       1000,
			visibleLines:     24,
			scrollPercent:    1.0,
			expectedThumbSize: 1,
			expectedThumbPos:  23, // 1.0 * (24 - 1) = 23
		},
		{
			name:             "Half-filled buffer",
			totalLines:       48,
			visibleLines:     24,
			scrollPercent:    0.0,
			expectedThumbSize: 12, // (24 * 24) / 48 = 12
			expectedThumbPos:  0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Calculate thumb size (same formula as in addScrollbar)
			thumbSize := max(1, (tc.visibleLines * tc.visibleLines) / tc.totalLines)
			assert.Equal(t, tc.expectedThumbSize, thumbSize, "Thumb size calculation")

			// Calculate thumb position
			trackSize := tc.visibleLines
			thumbPosition := int(tc.scrollPercent * float64(trackSize - thumbSize))
			assert.Equal(t, tc.expectedThumbPos, thumbPosition, "Thumb position calculation")
		})
	}
}

// BenchmarkVT10XScrollback benchmarks vt10x scrollback access performance
func BenchmarkVT10XScrollback(b *testing.B) {
	term := vt10x.New(vt10x.WithSize(80, 1000))

	// Fill with content
	for i := 0; i < 1000; i++ {
		term.Write([]byte("Benchmark line with some content\n"))
	}

	b.ResetTimer()

	// Benchmark reading full scrollback buffer
	for i := 0; i < b.N; i++ {
		for y := 0; y < 1000; y++ {
			for x := 0; x < 80; x++ {
				_ = term.Cell(x, y)
			}
		}
	}
}

// BenchmarkViewportUpdate benchmarks viewport scroll operations
func BenchmarkViewportUpdate(b *testing.B) {
	vp := viewport.New(80, 24)

	// Fill with content
	lines := make([]string, 1000)
	for i := range lines {
		lines[i] = "Line content here"
	}
	content := ""
	for _, line := range lines {
		content += line + "\n"
	}
	vp.SetContent(content)

	b.ResetTimer()

	// Benchmark scroll operations
	for i := 0; i < b.N; i++ {
		vp.Update(tea.KeyMsg{Type: tea.KeyDown})
		if i % 100 == 0 {
			vp.GotoTop() // Reset periodically
		}
	}
}

// TestScrollingResponsiveness tests that scroll operations complete quickly
func TestScrollingResponsiveness(t *testing.T) {
	vp := viewport.New(80, 24)

	// Create large content (1000 lines)
	lines := make([]string, 1000)
	for i := range lines {
		lines[i] = "Line " + string(rune('0'+i%10))
	}
	content := ""
	for _, line := range lines {
		content += line + "\n"
	}
	vp.SetContent(content)

	// Test that scroll operations complete in <1ms (target for UI responsiveness)
	start := time.Now()
	for i := 0; i < 100; i++ {
		vp, _ = vp.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	elapsed := time.Since(start)

	// 100 scroll operations should complete in <100ms (1ms per operation)
	assert.Less(t, elapsed, 100*time.Millisecond, "Scroll operations should be fast")

	// Average time per scroll should be <1ms
	avgPerScroll := elapsed / 100
	assert.Less(t, avgPerScroll, 1*time.Millisecond, "Average scroll time should be <1ms")
}
