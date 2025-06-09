---
sidebar_position: 3
---

# Installation

Multiple installation methods are available depending on your needs and system configuration.

## Quick Install (Recommended)

The fastest way to get started:

```bash
# Using curl
curl -sSL https://raw.githubusercontent.com/standardbeagle/brummer/main/quick-install.sh | bash

# Using wget
wget -qO- https://raw.githubusercontent.com/standardbeagle/brummer/main/quick-install.sh | bash
```

This script will:
- Check for Go installation
- Clone and build Brummer
- Install to `~/.local/bin` or `/usr/local/bin`
- Add to PATH if needed

## Install from Source

### Using Make

Clone the repository and use the Makefile:

```bash
git clone https://github.com/standardbeagle/brummer
cd brummer

# Install for current user (recommended)
make install-user

# OR install system-wide (requires sudo)
make install
```

### Using the Interactive Installer

For a guided installation experience:

```bash
git clone https://github.com/standardbeagle/brummer
cd brummer
./install.sh
```

The interactive installer will:
- Check prerequisites
- Build from source
- Optionally set up shell completion
- Verify the installation

### Manual Build

For full control over the installation:

```bash
git clone https://github.com/standardbeagle/brummer
cd brummer
go build -o brum ./cmd/brummer
mv brum ~/.local/bin/  # Or your preferred location
```

## Using Go Install

If you have Go installed and configured:

```bash
go install github.com/standardbeagle/brummer/cmd/brummer@latest
```

## Verifying Installation

After installation, verify Brummer is working:

```bash
brum --version
```

## Adding to PATH

If Brummer is not found after installation, add the install directory to your PATH:

### For User Installation

Add to your shell configuration file (`~/.bashrc`, `~/.zshrc`, etc.):

```bash
export PATH="$HOME/.local/bin:$PATH"
```

### For System Installation

The system directory `/usr/local/bin` should already be in PATH.

## Shell Completion

### Bash

Add to `~/.bashrc`:

```bash
source ~/.bash_completion.d/brum
```

### Zsh

Add to `~/.zshrc`:

```bash
# Coming soon
```

### Fish

Add to `~/.config/fish/config.fish`:

```bash
# Coming soon
```

## Updating Brummer

To update to the latest version:

```bash
# Using quick install
curl -sSL https://raw.githubusercontent.com/standardbeagle/brummer/main/quick-install.sh | bash

# Using make
cd brummer
git pull
make install-user
```

## Uninstalling

To remove Brummer:

```bash
# If installed with make
make uninstall

# Manual removal
rm ~/.local/bin/brum
# or
sudo rm /usr/local/bin/brum
```

## Troubleshooting

### Go Not Found

If you get "Go is not installed" error:

1. Install Go from [https://golang.org/dl/](https://golang.org/dl/)
2. Ensure Go is in your PATH:
   ```bash
   go version
   ```

### Permission Denied

If you get permission errors:
- Use `make install-user` instead of `make install`
- Or run with `sudo` for system-wide installation

### Command Not Found

If `brum` is not found after installation:
1. Check installation location:
   ```bash
   which brum
   ```
2. Add installation directory to PATH (see above)
3. Restart your terminal or run:
   ```bash
   source ~/.bashrc  # or your shell config
   ```