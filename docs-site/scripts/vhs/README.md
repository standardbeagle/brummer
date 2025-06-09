# VHS Screenshot Generation

This directory contains VHS (Video Hero System) scripts for automatically generating documentation screenshots.

## What is VHS?

VHS is a tool that allows you to script terminal interactions and generate GIFs or screenshots. It's perfect for creating consistent, reproducible documentation images.

## Installation

```bash
# macOS
brew install vhs

# Linux (via Go)
go install github.com/charmbracelet/vhs@latest

# Or download from releases
# https://github.com/charmbracelet/vhs/releases
```

## Usage

### Generate All Screenshots

```bash
cd docs-site/scripts
./generate-vhs-screenshots.sh
```

### Generate Individual Screenshot

```bash
cd docs-site/scripts/vhs
vhs brummer-tui.tape
```

## VHS Script Structure

Each `.tape` file contains commands to:
1. Set up the terminal environment
2. Create sample files/projects
3. Run Brummer
4. Navigate through the UI
5. Take screenshots at specific moments

Example:
```tape
Output ../static/img/screenshots/example.png
Set FontSize 14
Set Width 1200
Set Height 800
Set Theme "Dracula"

Type "brum"
Enter
Sleep 2s
Screenshot
```

## Available Scripts

- `brummer-tui.tape` - Main TUI interface
- `tutorial-first-launch.tape` - First time setup
- `tutorial-multiple-processes.tape` - Multiple processes running
- `tutorial-logs-view.tape` - Logs view demonstration
- `react-screenshots.tape` - React development workflow
- `error-detection.tape` - Error detection features
- `generate-all-screenshots.tape` - Comprehensive demo

## Customization

### Themes

VHS supports multiple terminal themes:
- `Dracula` (default)
- `Catppuccin Frappe`
- `Catppuccin Latte`
- `Catppuccin Macchiato`
- `Catppuccin Mocha`
- `TokyoNight`
- `Gruvbox`
- `OneDark`

### Output Formats

- `.png` - Static screenshots
- `.gif` - Animated demonstrations
- `.mp4` - Video files
- `.webm` - Web-optimized videos

### Terminal Settings

```tape
Set FontSize 16
Set FontFamily "JetBrains Mono"
Set LineHeight 1.2
Set LetterSpacing 0
Set Padding 20
Set Width 1200
Set Height 800
```

## Tips

1. **Consistency**: All screenshots use the same theme and dimensions
2. **Timing**: Use appropriate `Sleep` durations for realistic interaction
3. **Focus**: Each screenshot should demonstrate one clear feature
4. **Realism**: Create realistic project structures and data

## Troubleshooting

### "command not found: brum"
The VHS scripts assume `brum` is in PATH. Either:
- Install Brummer globally
- Modify scripts to use full path
- Create a mock `brum` command for screenshots

### Screenshots look different
Ensure:
- Same terminal theme is used
- Font settings match
- Window dimensions are consistent

### Scripts fail to run
Check:
- VHS is properly installed
- Output directories exist
- No syntax errors in `.tape` files