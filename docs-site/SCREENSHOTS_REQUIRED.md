# Screenshots Required for Documentation

The documentation build will FAIL until proper screenshots are generated. This is intentional.

## Required Screenshots

The following screenshots MUST be created before the documentation can build:

### Critical Screenshots (Build Blockers)
- `static/img/brummer-tui.png` - Main TUI interface
- `static/img/screenshots/tutorial-first-launch.png` - First launch experience
- `static/img/screenshots/react-scripts.png` - React development scripts
- `static/img/screenshots/nextjs-scripts.png` - Next.js scripts
- `static/img/screenshots/monorepo-overview.png` - Monorepo interface
- `static/img/screenshots/microservices-scripts.png` - Microservices scripts
- `static/img/screenshots/brummer-overview.gif` - Animated overview

## Generate Screenshots with VHS

### 1. Install VHS

```bash
# macOS
brew install vhs

# Linux/WSL
sudo snap install vhs

# Via Go
go install github.com/charmbracelet/vhs@latest
```

### 2. Install Brummer (if not already installed)

```bash
cd /home/beagle/work/brummer
make install-user
```

### 3. Generate Screenshots

```bash
cd docs-site/scripts
./generate-vhs-screenshots.sh
```

### 4. Verify Screenshots

```bash
# Check that all required images exist
ls -la ../static/img/screenshots/
```

## Alternative: Manual Screenshots

If VHS doesn't work for your setup:

1. Run Brummer in a terminal
2. Take screenshots using your OS tools
3. Save with exact filenames listed above
4. Place in `docs-site/static/img/screenshots/`

## DO NOT:
- Create placeholder images
- Use dummy screenshots
- Skip screenshot generation
- Modify docs to remove image references

The documentation is designed to fail without proper screenshots to ensure quality.