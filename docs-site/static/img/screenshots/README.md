# Screenshot Guidelines

This directory contains screenshots for the Brummer documentation. Follow these guidelines when creating screenshots.

## Required Screenshots

### Priority 1 - Core UI Views
- [ ] `brummer-tui.png` - Main TUI interface showing scripts list
- [ ] `tutorial-first-launch.png` - First time running Brummer
- [ ] `tutorial-processes-view.png` - Processes tab with running processes
- [ ] `tutorial-logs-view.png` - Logs view with colored output
- [ ] `tutorial-errors-view.png` - Errors tab showing error detection

### Priority 2 - Feature Demonstrations  
- [ ] `react-dev-server.png` - React dev server running
- [ ] `react-typescript-error.png` - TypeScript error highlighting
- [ ] `monorepo-overview.png` - Monorepo scripts organization
- [ ] `microservices-health.png` - Multiple services health status

### Priority 3 - Workflow Examples
- [ ] `nextjs-services.png` - Multiple Next.js services
- [ ] `monorepo-turbo.png` - Turborepo execution
- [ ] `microservices-errors.png` - Distributed error tracking

## Screenshot Requirements

### Dimensions
- Width: 800-1200px
- Height: 600-800px
- Aspect ratio: 4:3 or 16:10

### Terminal Settings
- Font: Use monospace font (SF Mono, Consolas, etc.)
- Font size: 14-16px
- Theme: Dark theme preferred
- Colors: Ensure good contrast

### Content Guidelines
1. **Clear Focus**: Each screenshot should demonstrate one key feature
2. **Realistic Data**: Use believable project names and outputs
3. **Clean State**: Remove unnecessary clutter
4. **Highlight Key Areas**: Use arrows or boxes for important elements

## Creating Screenshots

### Method 1: Actual Screenshots
1. Set up a demo project with relevant scripts
2. Run Brummer and navigate to the desired view
3. Use screenshot tool (cmd+shift+4 on macOS)
4. Crop to focus on Brummer window

### Method 2: Using Placeholder Generator
```bash
cd docs-site
node scripts/generate-screenshot-placeholders.js
```

This creates placeholder images that can be replaced with real screenshots.

### Method 3: Terminal Recording
Use `asciinema` or `terminalizer` for animated demonstrations:

```bash
# Record terminal session
asciinema rec demo.cast

# Convert to GIF
agg demo.cast demo.gif
```

## Image Optimization

Before committing:
1. Optimize PNG files using `pngquant` or `tinypng.com`
2. Keep file sizes under 200KB where possible
3. Use meaningful filenames that match documentation references

## Naming Convention

- `feature-aspect.png` - Static screenshots
- `workflow-name.gif` - Animated demonstrations
- `comparison-before-after.png` - Before/after comparisons

Examples:
- `react-dev-server.png`
- `monorepo-build-process.gif`
- `error-detection-comparison.png`

## Tools Recommended

- **macOS**: CleanShot X, Kap (for GIFs)
- **Windows**: ShareX, ScreenToGif
- **Linux**: Flameshot, Peek
- **Cross-platform**: OBS Studio

## Placeholder Note

Current placeholders were generated with:
- Dark terminal theme
- Standard terminal chrome (red/yellow/green buttons)
- Descriptive text indicating what should be shown

Replace these with actual screenshots before documentation release.