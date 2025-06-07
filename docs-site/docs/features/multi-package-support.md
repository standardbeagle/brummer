---
sidebar_position: 1
---

# Multi-Package Manager Support

Brummer seamlessly works with all major JavaScript package managers.

## Supported Package Managers

### npm
- **Version**: 5.0+
- **Detection**: `package-lock.json`
- **Commands**: Standard npm scripts

### Yarn
- **Version**: 1.0+ and 2.0+ (Berry)
- **Detection**: `yarn.lock`
- **Commands**: yarn run scripts

### pnpm
- **Version**: 3.0+
- **Detection**: `pnpm-lock.yaml`
- **Commands**: pnpm run scripts

### Bun
- **Version**: 1.0+
- **Detection**: `bun.lockb`
- **Commands**: bun run scripts

## Auto-Detection

Brummer automatically detects your package manager by checking:

1. **Lock Files** (highest priority)
   - `package-lock.json` → npm
   - `yarn.lock` → yarn
   - `pnpm-lock.yaml` → pnpm
   - `bun.lockb` → bun

2. **Workspace Files**
   - `pnpm-workspace.yaml` → pnpm
   - `.yarnrc.yml` → yarn 2+

3. **Global Availability**
   - Checks PATH for installed managers
   - Falls back to npm if available

## Manual Override

### Via Settings
1. Press `6` to open Settings
2. Select Package Manager option
3. Press Enter to cycle through options

### Via Environment
```bash
BRUMMER_PACKAGE_MANAGER=pnpm brum
```

### Via Command Line (Future)
```bash
brum --package-manager yarn
```

## Monorepo Support

### Workspaces
Brummer detects workspace configurations:
- npm workspaces (package.json)
- yarn workspaces
- pnpm workspaces
- Lerna projects

### Running Scripts in Workspaces
```json
{
  "scripts": {
    "dev": "npm run dev --workspaces",
    "test:client": "npm run test -w client",
    "build:all": "lerna run build"
  }
}
```

## Package Manager Features

### npm
- Lifecycle scripts
- Pre/post scripts
- Script arguments
- Workspace support

### Yarn
- Plug'n'Play support
- Zero-installs
- Workspace protocols
- Berry plugins

### pnpm
- Efficient disk usage
- Strict dependencies
- Workspace protocols
- Filtering

### Bun
- Fast execution
- Built-in TypeScript
- Native bundling
- Compatible with npm

## Best Practices

### Lock File Management
- Commit lock files to version control
- Don't mix package managers
- Use single manager per project
- Keep lock files up to date

### Script Naming
Consistent naming across managers:
```json
{
  "scripts": {
    "dev": "...",        // Development server
    "build": "...",      // Production build
    "test": "...",       // Run tests
    "lint": "...",       // Code linting
    "start": "..."       // Production start
  }
}
```

### Performance Tips
- **pnpm**: Best for monorepos
- **bun**: Fastest execution
- **yarn**: Good caching
- **npm**: Most compatible

## Common Issues

### Wrong Package Manager Detected
1. Check for multiple lock files
2. Remove unintended lock files
3. Use manual override

### Command Not Found
- Ensure package manager is installed
- Check PATH configuration
- Try global installation

### Workspace Scripts Not Found
- Verify workspace configuration
- Check script exists in workspace
- Use correct workspace syntax

## Migration Guide

### npm to yarn
```bash
rm package-lock.json
yarn install
```

### yarn to pnpm
```bash
rm yarn.lock
pnpm import
pnpm install
```

### Any to bun
```bash
bun install
```

## Compatibility Table

| Feature | npm | yarn | pnpm | bun |
|---------|-----|------|------|-----|
| Basic scripts | ✅ | ✅ | ✅ | ✅ |
| Workspaces | ✅ | ✅ | ✅ | ✅ |
| Pre/post scripts | ✅ | ❌* | ✅ | ✅ |
| PnP | ❌ | ✅ | ❌ | ❌ |
| Built-in TS | ❌ | ❌ | ❌ | ✅ |

*Yarn 2+ removed automatic pre/post scripts

## Tips

1. **Consistency**: Use one package manager per project
2. **Lock Files**: Always commit lock files
3. **CI/CD**: Match local and CI package managers
4. **Updates**: Keep package managers updated