package components

// LayoutRegions represents the calculated dimensions for different areas of a view
// This provides a centralized way to manage layout calculations across components
type LayoutRegions struct {
	// Total available dimensions
	TotalWidth  int
	TotalHeight int

	// Terminal/PTY dimensions (actual terminal size)
	TerminalWidth  int
	TerminalHeight int

	// Content area dimensions (viewport)
	ContentWidth  int
	ContentHeight int

	// Header and footer dimensions
	HeaderHeight int
	FooterHeight int

	// Padding and borders
	Padding int
}

// LayoutOptions contains parameters for layout calculation
type LayoutOptions struct {
	TotalWidth   int
	TotalHeight  int
	HeaderHeight int
	FooterHeight int
	Padding      int
	MinTermCols  int
	MinTermRows  int
}

// DefaultLayoutOptions returns default layout configuration
func DefaultLayoutOptions(width, height int) LayoutOptions {
	return LayoutOptions{
		TotalWidth:   width,
		TotalHeight:  height,
		HeaderHeight: 3, // Default header height
		FooterHeight: 2, // Default footer height
		Padding:      2, // Default padding
		MinTermCols:  80,
		MinTermRows:  24,
	}
}

// CalculateLayoutRegions computes all layout dimensions from the given options
func CalculateLayoutRegions(opts LayoutOptions) LayoutRegions {
	regions := LayoutRegions{
		TotalWidth:   opts.TotalWidth,
		TotalHeight:  opts.TotalHeight,
		HeaderHeight: opts.HeaderHeight,
		FooterHeight: opts.FooterHeight,
		Padding:      opts.Padding,
	}

	// Calculate content area (viewport)
	regions.ContentWidth = opts.TotalWidth - (opts.Padding * 2)
	regions.ContentHeight = opts.TotalHeight - opts.HeaderHeight - opts.FooterHeight - opts.Padding

	// Calculate terminal dimensions (content area minus borders)
	regions.TerminalWidth = regions.ContentWidth - 4 // 4 for padding and borders
	regions.TerminalHeight = regions.ContentHeight - 5 // 5 for header/footer

	// Enforce minimum dimensions
	if regions.TerminalWidth < opts.MinTermCols {
		regions.TerminalWidth = opts.MinTermCols
	}
	if regions.TerminalHeight < opts.MinTermRows {
		regions.TerminalHeight = opts.MinTermRows
	}

	return regions
}

// GetViewportArea returns the content area dimensions (width, height)
func (r LayoutRegions) GetViewportArea() (int, int) {
	return r.ContentWidth, r.ContentHeight
}

// GetTerminalSize returns the terminal dimensions (cols, rows)
func (r LayoutRegions) GetTerminalSize() (int, int) {
	return r.TerminalWidth, r.TerminalHeight
}
