# Brummer Distribution Methods

This document outlines the simplified distribution strategy for Brummer.

## Available Installation Methods

### 1. 📦 NPM Package (Recommended)
**Package**: `@standardbeagle/brum`

```bash
# Global installation
npm install -g @standardbeagle/brum
yarn global add @standardbeagle/brum
pnpm add -g @standardbeagle/brum

# Run without installing
npx @standardbeagle/brum

# After installation, run with:
brum
```

**Features**:
- ✅ Cross-platform binary distribution
- ✅ Automatic platform/architecture detection
- ✅ Works with npm, yarn, pnpm
- ✅ Available via npx for one-time use

### 2. 🐹 Go Install
```bash
go install github.com/standardbeagle/brummer/cmd/brum@latest
```

**Benefits**:
- ✅ Direct from source
- ✅ Always latest version
- ✅ Familiar to Go developers
- ✅ No configuration needed

### 3. 🚀 Quick Install Scripts
**Linux/macOS**:
```bash
curl -sSL https://raw.githubusercontent.com/standardbeagle/brummer/main/quick-install.sh | bash
```

**Windows**:
```powershell
irm https://raw.githubusercontent.com/standardbeagle/brummer/main/quick-install.ps1 | iex
```

**Features**:
- ✅ One-liner installation
- ✅ Automatic platform detection
- ✅ Installs to user directory
- ✅ No package manager required

### 4. 📥 Direct Download
Download binaries directly from GitHub Releases:
- `brum-linux-amd64`
- `brum-linux-arm64`
- `brum-darwin-amd64` (Intel Mac)
- `brum-darwin-arm64` (Apple Silicon)
- `brum-windows-amd64.exe`
- `brum-windows-arm64.exe`

## Removed Methods

The following distribution methods have been removed to simplify maintenance:

- ❌ **Homebrew**: Requires separate tap maintenance
- ❌ **Chocolatey**: Requires Windows code signing
- ❌ **Winget**: Requires Windows code signing and Microsoft validation

## Publishing Workflow

1. **Build binaries**: `make build-all`
2. **Test NPM package**: `node install.js && ./bin/brum --version`
3. **Create GitHub release** with binaries
4. **Publish NPM package**: `npm publish`
5. **Verify installations**:
   - `npm install -g @standardbeagle/brum && brum --version`
   - `go install github.com/standardbeagle/brummer/cmd/brum@latest && brum --version`

## Support

- **NPM Issues**: Check package installation with `npm list -g @standardbeagle/brum`
- **Go Install Issues**: Ensure Go 1.21+ is installed
- **Quick Install Issues**: Check script permissions and internet connectivity
- **General Issues**: [GitHub Issues](https://github.com/standardbeagle/brummer/issues)

## Benefits of Simplified Distribution

✅ **Reduced Maintenance**: No package manager-specific files to maintain  
✅ **Faster Releases**: No external approval processes  
✅ **Better Security**: No code signing requirements  
✅ **Easier Testing**: Fewer distribution methods to validate  
✅ **Cross-Platform**: NPM works on all platforms  
✅ **Developer Friendly**: Go install familiar to target audience