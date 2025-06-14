# Publishing Brummer to Package Managers

This guide covers how to publish Brummer to various package managers.

## Prerequisites

Before publishing, ensure:
1. All tests pass: `make test`
2. Build works: `make build-all`
3. Version is updated in `package.json`
4. Changes are committed and tagged

## NPM Package ✅ Ready

### Setup (One-time)
```bash
# Login to npm (if not already logged in)
npm login

# Verify your login
npm whoami
```

### Publishing Process
1. **Update version** in `package.json`
2. **Build binaries** for all platforms:
   ```bash
   make build-all
   ```
3. **Test the package locally**:
   ```bash
   node install.js
   ./bin/brum --help
   ```
4. **Publish to npm**:
   ```bash
   npm publish
   ```

### NPM Package Features
- ✅ Cross-platform binary installation
- ✅ Automatic platform detection (Linux, macOS, Windows)
- ✅ Architecture support (x64, arm64)
- ✅ Fallback to local binaries during development
- ✅ Post-install script downloads appropriate binary
- ✅ GitHub Releases integration

### Installation Commands
Users can install via:
```bash
# Global installation
npm install -g @standardbeagle/brum
yarn global add @standardbeagle/brum
pnpm add -g @standardbeagle/brum

# After installation, run with:
brum

# Run without installing
npx @standardbeagle/brum
```

## Go Install

Go install is automatically available since the code is public on GitHub:

```bash
go install github.com/standardbeagle/brummer/cmd/brum@latest
```

No additional setup required.

## Release Checklist

- [ ] Update CHANGELOG.md
- [ ] Build all platform binaries: `make build-all`
- [ ] Create GitHub release with binaries
- [ ] Publish to NPM: `npm publish`
- [ ] Test installations:
  - [ ] `npm install -g @standardbeagle/brum`
  - [ ] `npx @standardbeagle/brum`
  - [ ] `go install github.com/standardbeagle/brummer/cmd/brum@latest`