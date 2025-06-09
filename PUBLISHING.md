# Publishing Guide

This guide explains how to publish Brummer to various package managers.

## Prerequisites

1. Build binaries for all platforms:
   ```bash
   make build-all
   ```

2. Create a new GitHub release with the binaries
3. Note the SHA256 checksums for each binary

## NPM

1. Update version in package.json
2. Ensure binaries are available on GitHub releases
3. Publish to NPM:
   ```bash
   npm publish
   ```

## Homebrew

1. Update the SHA256 checksums in `homebrew/brummer.rb`
2. Submit PR to homebrew-core or maintain your own tap:
   ```bash
   # Create tap repository: homebrew-brummer
   # Add formula to: Formula/brummer.rb
   ```

## Chocolatey

1. Update SHA256 in `chocolatey/tools/chocolateyInstall.ps1`
2. Pack the package:
   ```powershell
   choco pack chocolatey/brummer.nuspec
   ```
3. Push to Chocolatey:
   ```powershell
   choco push brummer.0.1.0.nupkg --source https://push.chocolatey.org/
   ```

## Winget

1. Update SHA256 in installer manifest
2. Submit PR to [microsoft/winget-pkgs](https://github.com/microsoft/winget-pkgs)
3. Follow their contribution guidelines

## Version Update Script

Use the helper script to update versions across all files:
```bash
./scripts/update-version.sh 0.2.0
```

## Release Checklist

- [ ] Update CHANGELOG.md
- [ ] Run `./scripts/update-version.sh X.Y.Z`
- [ ] Build all platform binaries: `make build-all`
- [ ] Create GitHub release with binaries
- [ ] Update SHA256 checksums in all package files
- [ ] Publish to NPM
- [ ] Submit PRs to Homebrew, Winget
- [ ] Push to Chocolatey