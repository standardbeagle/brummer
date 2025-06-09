---
sidebar_position: 3
---

# Monorepo Workflows

Master monorepo development with Brummer's intelligent workspace management and cross-package script coordination.

## Overview

Monorepos present unique challenges:
- Running multiple applications simultaneously
- Managing shared dependencies
- Coordinating builds across packages
- Handling inter-package dependencies
- Monitoring multiple test suites

Brummer excels at managing these complex scenarios.

## Supported Monorepo Tools

Brummer automatically detects and supports:
- 📦 **pnpm workspaces**
- 🧶 **Yarn workspaces**
- 📦 **npm workspaces**
- 🏗️ **Turborepo**
- 🛠️ **Nx**
- 🚀 **Rush**
- 📚 **Lerna**

## Monorepo Structure Example

```
my-monorepo/
├── apps/
│   ├── web/          # Next.js frontend
│   ├── mobile/       # React Native app
│   └── api/          # Express backend
├── packages/
│   ├── ui/           # Shared UI components
│   ├── database/     # Prisma schema & client
│   ├── config/       # Shared configurations
│   └── utils/        # Shared utilities
├── package.json
├── pnpm-workspace.yaml
└── turbo.json
```

## Starting Brummer in a Monorepo

```bash
cd my-monorepo
brum
```

![Monorepo Scripts Overview](../img/screenshots/monorepo-overview.png)

Brummer shows:
- Root-level scripts
- Per-package scripts
- Workspace commands
- Cross-package tasks

## Turborepo Integration

### Configuration

```json title="turbo.json"
{
  "pipeline": {
    "build": {
      "dependsOn": ["^build"],
      "outputs": ["dist/**", ".next/**"]
    },
    "dev": {
      "cache": false,
      "persistent": true
    },
    "test": {
      "dependsOn": ["build"],
      "inputs": ["src/**", "test/**"]
    }
  }
}
```

### Running Turbo Commands

![Turborepo Pipeline](../img/screenshots/monorepo-turbo.png)

**Brummer Features:**
- ⚡ Shows Turbo cache hits/misses
- 🔄 Displays task dependencies
- 📊 Tracks parallel execution
- ⏱️ Reports task timing

## Common Monorepo Workflows

### 1. Starting All Development Servers

```json title="package.json (root)"
{
  "scripts": {
    "dev": "turbo dev",
    "dev:web": "turbo dev --filter=web",
    "dev:api": "turbo dev --filter=api",
    "dev:all": "turbo dev --parallel"
  }
}
```

In Brummer:
1. Select `dev:all` to start all apps
2. Monitor each app in the Processes tab
3. View consolidated logs in Logs tab

![All Apps Running](../img/screenshots/monorepo-all-apps.png)

### 2. Building Dependent Packages

Watch the build cascade:

```bash
# Building app with dependencies
turbo build --filter=web...
```

Brummer shows:
```
🔨 Building @repo/utils
✅ @repo/utils built (2.1s)
🔨 Building @repo/ui
✅ @repo/ui built (5.3s)
🔨 Building @repo/web
✅ @repo/web built (12.4s)
```

### 3. Running Tests Across Packages

```json
{
  "scripts": {
    "test": "turbo test",
    "test:watch": "turbo test --watch",
    "test:affected": "turbo test --filter=[origin/main]"
  }
}
```

![Monorepo Tests](../img/screenshots/monorepo-tests.png)

## Package-Specific Development

### Working on Shared UI Library

```bash
# Focus on UI package development
cd packages/ui
brum
```

Scripts available:
- `dev`: Run Storybook
- `test`: Run component tests
- `build`: Build library
- `lint`: Check code quality

### Cross-Package Hot Reload

Brummer tracks hot reload across packages:

1. Edit component in `packages/ui`
2. See rebuild in UI package logs
3. Watch web app hot reload
4. Monitor for type errors

![Cross-Package HMR](../img/screenshots/monorepo-hmr.png)

## Advanced Patterns

### Filtered Commands

Run commands for specific packages:

```json
{
  "scripts": {
    // Run only changed packages
    "dev:changed": "turbo dev --filter='[HEAD^1]'",
    
    // Run specific app and deps
    "dev:web-stack": "turbo dev --filter=web...",
    
    // Exclude packages
    "build:apps": "turbo build --filter='./apps/*'"
  }
}
```

### Dependency Graph Visualization

```bash
# Generate and open dependency graph
turbo graph
```

Brummer captures the graph generation URL:

![Dependency Graph](../img/screenshots/monorepo-graph.png)

## Error Handling in Monorepos

### TypeScript Project References

Monitor TypeScript build errors across packages:

```typescript
// packages/ui/src/Button.tsx
export interface ButtonProps {
  variant: 'primary' | 'secondary'
  // Error: missing required prop
}

// apps/web/src/HomePage.tsx
import { Button } from '@repo/ui'
// Error shows here too!
```

Brummer displays:
- Source package error
- Consuming package errors
- Build order issues

### Circular Dependencies

Brummer helps identify circular dependencies:

```
❌ Circular dependency detected:
   @repo/ui → @repo/utils → @repo/ui
```

## Performance Optimization

### Parallel Execution

Monitor parallel task execution:

![Parallel Tasks](../img/screenshots/monorepo-parallel.png)

Tips:
- Use `--parallel` for independent tasks
- Monitor CPU usage
- Adjust concurrency with `--concurrency`

### Cache Performance

Track Turborepo cache efficiency:

```
Cache Summary:
  - Total tasks: 12
  - Cache hits: 9 (75%)
  - Cache misses: 3
  - Time saved: 45.2s
```

## Workspace Management

### pnpm Workspaces

```yaml title="pnpm-workspace.yaml"
packages:
  - 'apps/*'
  - 'packages/*'
  - 'tools/*'
```

Brummer commands:
```bash
# Install dependency to specific package
pnpm add lodash --filter web

# Run script in all packages
pnpm -r dev
```

### Yarn Workspaces

```json title="package.json"
{
  "workspaces": [
    "apps/*",
    "packages/*"
  ]
}
```

### npm Workspaces

```json title="package.json"
{
  "workspaces": [
    "apps/web",
    "apps/api",
    "packages/*"
  ]
}
```

## Debugging Monorepo Issues

### 1. Dependency Resolution

Monitor package resolution:

```bash
# Check why package version is used
pnpm why react
```

### 2. Build Order Problems

Identify build order issues:
- Check Turbo pipeline configuration
- Verify package.json dependencies
- Look for missing peerDependencies

### 3. Hot Reload Not Working

Common causes:
- Symlinks not watched
- Incorrect TypeScript paths
- Missing workspace dependencies

## CI/CD Integration

### Local CI Testing

```json
{
  "scripts": {
    "ci": "turbo lint test build --cache-dir=.turbo",
    "ci:affected": "turbo lint test build --filter=[origin/main]"
  }
}
```

Monitor CI-like execution locally:

![Local CI](../img/screenshots/monorepo-ci.png)

## Best Practices

### 1. Script Organization

```json
{
  "scripts": {
    // Development
    "dev": "turbo dev",
    "dev:web": "turbo dev --filter=web",
    
    // Building
    "build": "turbo build",
    "build:packages": "turbo build --filter='./packages/*'",
    
    // Testing
    "test": "turbo test",
    "test:watch": "turbo test -- --watch",
    
    // Utilities
    "clean": "turbo clean && rm -rf node_modules",
    "format": "turbo format"
  }
}
```

### 2. Process Management

1. **Start in Dependency Order**
   - Database first
   - Shared packages
   - Applications

2. **Group Related Processes**
   - Frontend stack
   - Backend services
   - Development tools

3. **Monitor Resource Usage**
   - Watch memory per package
   - CPU usage during builds
   - Disk I/O for large builds

### 3. Log Management

Filter logs by package:

```bash
# Show only web app logs
/show @repo/web

# Hide verbose build output
/hide turbo:build

# Show only errors
/show error
```

## Troubleshooting

### Common Issues

1. **"Cannot find module" Errors**
   - Rebuild dependent packages
   - Check workspace configuration
   - Verify symlinks

2. **Type Errors Across Packages**
   - Ensure TypeScript project references
   - Build packages in correct order
   - Check tsconfig paths

3. **Slow Build Times**
   - Enable Turborepo caching
   - Use remote caching
   - Parallelize independent tasks

## Next Steps

- Explore [Microservices Development](./microservices)
- Learn about [Performance Monitoring](./performance-monitoring)
- Set up [Team Collaboration](../tutorials/team-collaboration)