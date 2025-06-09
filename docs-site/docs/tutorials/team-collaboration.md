---
sidebar_position: 3
---

# Team Collaboration

Learn how to standardize Brummer usage across your team for consistent development environments.

## Overview

When working in teams, consistency is key. This guide shows how to:
- Share Brummer configurations
- Standardize script naming conventions
- Create team-specific workflows
- Document best practices
- Integrate with CI/CD pipelines

## Standardizing Scripts

### 1. Create Team Conventions

Establish naming patterns for scripts:

```json title="package.json"
{
  "scripts": {
    // Development
    "dev": "Main development server",
    "dev:*": "Additional development services",
    
    // Testing
    "test": "Run all tests",
    "test:unit": "Unit tests only",
    "test:integration": "Integration tests",
    "test:e2e": "End-to-end tests",
    
    // Building
    "build": "Production build",
    "build:*": "Specific build targets",
    
    // Infrastructure
    "infra:*": "Infrastructure services",
    
    // Utilities
    "lint": "Code linting",
    "format": "Code formatting",
    "typecheck": "TypeScript checking"
  }
}
```

### 2. Document Script Purpose

Add a `scripts.md` file to your project:

```markdown title="scripts.md"
# Project Scripts Guide

## Development Scripts

### `npm run dev`
Starts the main development server with hot reload enabled.
- Port: 3000
- Environment: development
- Features: Hot reload, source maps, debug logging

### `npm run dev:api`
Starts the mock API server for local development.
- Port: 3001
- Data: Uses `mock-data/` directory

## Testing Scripts

### `npm run test`
Runs all test suites in watch mode.
- Use `npm run test:ci` for single run
- Coverage reports in `coverage/`
```

## Shared Configuration

### 1. Project Configuration File

Create `.brummer/config.json`:

```json title=".brummer/config.json"
{
  "defaultScripts": ["dev", "test:watch"],
  "processGroups": {
    "frontend": ["dev:web", "dev:storybook"],
    "backend": ["dev:api", "dev:workers"],
    "full": ["infra:start", "dev:api", "dev:web"]
  },
  "aliases": {
    "start": "dev",
    "api": "dev:api",
    "web": "dev:web"
  },
  "environment": {
    "development": {
      "NODE_ENV": "development",
      "LOG_LEVEL": "debug"
    },
    "staging": {
      "NODE_ENV": "staging",
      "LOG_LEVEL": "info"
    }
  }
}
```

### 2. Team Scripts

Add team-specific helper scripts:

```json title="package.json"
{
  "scripts": {
    // Team workflows
    "team:setup": "npm install && npm run db:migrate && npm run db:seed",
    "team:reset": "npm run clean && npm run team:setup",
    "team:update": "git pull && npm install && npm run db:migrate",
    
    // Onboarding
    "onboard": "node scripts/onboard.js",
    "verify:setup": "node scripts/verify-setup.js"
  }
}
```

## MCP Server Configuration

### Shared MCP Settings

Create team MCP configuration:

```json title=".brummer/mcp-team.json"
{
  "server": {
    "port": 7777,
    "authentication": {
      "enabled": true,
      "tokens": {
        "vscode": "${BRUMMER_VSCODE_TOKEN}",
        "cursor": "${BRUMMER_CURSOR_TOKEN}",
        "claude": "${BRUMMER_CLAUDE_TOKEN}"
      }
    }
  },
  "clients": {
    "vscode": {
      "name": "VS Code",
      "features": ["logs", "execute", "monitor"]
    },
    "ci": {
      "name": "CI Pipeline",
      "features": ["execute", "monitor"],
      "readonly": true
    }
  }
}
```

### IDE Configuration

Share IDE settings for Brummer integration:

```json title=".vscode/settings.json"
{
  "brummer.mcp.enabled": true,
  "brummer.mcp.port": 7777,
  "brummer.mcp.autoConnect": true,
  "brummer.ui.defaultView": "processes"
}
```

## Documentation Templates

### 1. README Integration

Add Brummer section to README:

```markdown title="README.md"
## Development with Brummer

This project uses [Brummer](https://github.com/standardbeagle/brummer) for development process management.

### Quick Start

1. Install Brummer: `npm install -g brummer`
2. Start development: `brum` then select `dev`
3. Run tests: Select `test:watch` in Brummer

### Common Workflows

#### Full Stack Development
1. Start Brummer: `brum`
2. Run `infra:start` (databases, Redis, etc.)
3. Run `dev:api` (backend API)
4. Run `dev:web` (frontend)
5. Run `test:watch` (optional)

#### Running Specific Services
- Frontend only: `brum` → `dev:web`
- API only: `brum` → `dev:api`
- Tests only: `brum` → `test`

### Troubleshooting
See [Development Guide](./docs/development.md) for detailed information.
```

### 2. Development Guide

Create comprehensive guide:

```markdown title="docs/development-guide.md"
# Development Guide

## Environment Setup

### Prerequisites
- Node.js 18+
- Brummer (`npm install -g brummer`)
- Docker (for databases)

### Initial Setup
```bash
# Clone and install
git clone <repo>
cd <project>
npm install

# Setup databases
npm run infra:setup

# Verify setup
npm run verify:setup
```

## Daily Development

### Starting Your Environment

1. **Start Brummer**
   ```bash
   brum
   ```

2. **Start Infrastructure**
   - Select `infra:start` or press `1`
   - Wait for "Infrastructure ready" message

3. **Start Services**
   - Select `dev` for all services
   - Or start individually:
     - `dev:api` - Backend API
     - `dev:web` - Frontend
     - `dev:worker` - Background jobs

### Monitoring

- **Logs**: Tab to Logs view
- **Errors**: Tab to Errors view for issues
- **URLs**: Tab to URLs view for endpoints

### Common Tasks

#### Running Tests
- `test` - All tests in watch mode
- `test:unit` - Unit tests only
- `test:api` - API integration tests

#### Database Tasks
- `db:migrate` - Run migrations
- `db:seed` - Seed development data
- `db:reset` - Reset and reseed

#### Code Quality
- `lint` - Check code style
- `format` - Auto-format code
- `typecheck` - TypeScript validation
```

## Team Workflows

### 1. Onboarding New Developers

Create onboarding script:

```javascript title="scripts/onboard.js"
#!/usr/bin/env node

const { execSync } = require('child_process');
const fs = require('fs');
const path = require('path');

console.log('🚀 Welcome to the team! Setting up your development environment...\n');

// Check prerequisites
console.log('📋 Checking prerequisites...');
checkCommand('node', '--version', '18.0.0');
checkCommand('npm', '--version', '8.0.0');
checkCommand('brum', '--version', null, 'npm install -g brummer');

// Install dependencies
console.log('\n📦 Installing dependencies...');
execSync('npm install', { stdio: 'inherit' });

// Setup environment
console.log('\n🔧 Setting up environment...');
if (!fs.existsSync('.env')) {
  fs.copyFileSync('.env.example', '.env');
  console.log('✅ Created .env file from template');
}

// Setup databases
console.log('\n🗄️ Setting up databases...');
execSync('npm run infra:setup', { stdio: 'inherit' });

// Run initial migrations
console.log('\n📊 Running database migrations...');
execSync('npm run db:migrate', { stdio: 'inherit' });

// Seed data
console.log('\n🌱 Seeding development data...');
execSync('npm run db:seed', { stdio: 'inherit' });

// Verify setup
console.log('\n✅ Verifying setup...');
execSync('npm run verify:setup', { stdio: 'inherit' });

console.log('\n🎉 Setup complete! You can now run:');
console.log('   brum');
console.log('   Then select "dev" to start developing\n');

function checkCommand(command, flag, minVersion, installCmd) {
  try {
    const version = execSync(`${command} ${flag}`, { encoding: 'utf8' });
    console.log(`✅ ${command}: ${version.trim()}`);
  } catch (error) {
    console.error(`❌ ${command} not found`);
    if (installCmd) {
      console.log(`   Please install with: ${installCmd}`);
    }
    process.exit(1);
  }
}
```

### 2. Daily Standup Helper

Create standup script that uses Brummer:

```javascript title="scripts/standup.js"
#!/usr/bin/env node

const { execSync } = require('child_process');

console.log('📊 Generating standup report...\n');

// Get git commits from yesterday
const yesterday = new Date();
yesterday.setDate(yesterday.getDate() - 1);
const dateStr = yesterday.toISOString().split('T')[0];

console.log('📝 Yesterday\'s commits:');
try {
  execSync(`git log --since="${dateStr} 00:00" --until="${dateStr} 23:59" --oneline --author="$(git config user.email)"`, { stdio: 'inherit' });
} catch (e) {
  console.log('   No commits yesterday');
}

console.log('\n🚀 Currently running processes:');
// This would integrate with Brummer's MCP API
console.log('   Check Brummer for active processes\n');

console.log('📋 Today\'s plan:');
console.log('   1. Start development environment with Brummer');
console.log('   2. Continue working on current tasks');
console.log('   3. Run tests before pushing\n');
```

## CI/CD Integration

### 1. CI Pipeline with Brummer

```yaml title=".github/workflows/ci.yml"
name: CI

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    
    steps:
    - uses: actions/checkout@v3
    
    - name: Setup Node.js
      uses: actions/setup-node@v3
      with:
        node-version: '18'
    
    - name: Install Brummer
      run: |
        curl -sSL https://raw.githubusercontent.com/standardbeagle/brummer/main/quick-install.sh | bash
        echo "$HOME/.local/bin" >> $GITHUB_PATH
    
    - name: Install dependencies
      run: npm ci
    
    - name: Start services with Brummer
      run: |
        # Start Brummer in headless mode
        brum --no-tui &
        BRUMMER_PID=$!
        
        # Wait for MCP server
        sleep 5
        
        # Execute test script via MCP
        curl -X POST http://localhost:7777/mcp/execute \
          -H "Content-Type: application/json" \
          -d '{"script": "test:ci"}'
    
    - name: Check test results
      run: |
        # Monitor test execution via MCP
        curl http://localhost:7777/mcp/logs?processId=test:ci
```

### 2. Local CI Testing

Add script for local CI testing:

```json title="package.json"
{
  "scripts": {
    "ci:local": "brum --no-tui & npm run test:ci && npm run build",
    "ci:validate": "npm run lint && npm run typecheck && npm run test:ci"
  }
}
```

## Best Practices

### 1. Script Naming Conventions

```javascript
// ✅ Good script names
"dev:api"        // Clear purpose
"test:unit"      // Specific scope
"build:prod"     // Environment specified

// ❌ Poor script names
"start"          // Ambiguous
"test1"          // Unclear purpose
"run-thing"      // Non-standard format
```

### 2. Process Groups

Organize related processes:

```json
{
  "scripts": {
    // Frontend group
    "frontend": "concurrently \"npm:dev:web\" \"npm:dev:storybook\"",
    
    // Backend group
    "backend": "concurrently \"npm:dev:api\" \"npm:dev:workers\"",
    
    // Full stack
    "dev:all": "concurrently \"npm:frontend\" \"npm:backend\""
  }
}
```

### 3. Environment Management

```bash
# Development
NODE_ENV=development brum

# Staging testing
NODE_ENV=staging brum

# Production debugging
NODE_ENV=production LOG_LEVEL=debug brum
```

## Troubleshooting Team Issues

### Common Problems

1. **Different Node Versions**
   - Use `.nvmrc` file
   - Document in README
   - Check in onboarding script

2. **Port Conflicts**
   - Document all ports used
   - Use environment variables
   - Provide port override options

3. **Missing Dependencies**
   - Keep `.env.example` updated
   - Document external services
   - Provide mock alternatives

### Team Communication

Create team channels:
- `#brummer-help` - Get help with Brummer
- `#dev-environment` - Environment issues
- `#script-updates` - Announce script changes

## Summary

Effective team collaboration with Brummer requires:
- ✅ Standardized script naming
- ✅ Shared configuration files
- ✅ Comprehensive documentation
- ✅ Onboarding automation
- ✅ CI/CD integration
- ✅ Clear communication channels

Your team will benefit from:
- Consistent development environments
- Faster onboarding
- Reduced setup issues
- Better debugging capabilities
- Improved productivity