---
sidebar_position: 1
---

# Introduction

Welcome to **Brummer** - your Terminal UI Development Buddy that transforms chaotic terminal management into an organized, efficient workflow.

## See Brummer in Action

![Brummer Overview](./img/screenshots/brummer-overview.gif)

## What is Brummer?

Brummer is a powerful terminal UI that revolutionizes how developers manage npm/yarn/pnpm/bun scripts. Instead of juggling multiple terminal windows, Brummer provides a single, intelligent interface for all your development processes.

### Before Brummer 😫

```bash
# Terminal 1: Frontend
npm run dev

# Terminal 2: Backend
npm run api

# Terminal 3: Tests
npm run test:watch

# Terminal 4: Database
docker-compose up

# Terminal 5: Where's that error?
# Terminal 6: Which process crashed?
# Terminal 7: What port was that on?
```

### With Brummer 🚀

```bash
brum
# Everything in one beautiful interface!
```

## Real-World Benefits

### 🎯 For Frontend Developers
- **Hot Reload Monitoring**: See exactly when rebuilds happen
- **Error Highlighting**: TypeScript errors stand out immediately
- **Bundle Size Tracking**: Monitor build performance
- **Test Integration**: Run tests alongside development

### 🔧 For Full-Stack Developers
- **Service Orchestration**: Start frontend, backend, and database together
- **API Monitoring**: Track requests and responses
- **Database Management**: Run migrations and seeders easily
- **Unified Logging**: All services in one view

### 📦 For Monorepo Teams
- **Workspace Management**: Handle multiple packages effortlessly
- **Dependency Tracking**: See build order and caching
- **Parallel Execution**: Monitor concurrent tasks
- **Cross-Package HMR**: Track changes across packages

## Why Developers Love Brummer

### 1. **Zero Configuration**
```bash
# Just run in any project
cd my-project
brum
```

### 2. **Intelligent Error Detection**
![Error Detection](./img/screenshots/error-detection-preview.png)

Brummer automatically:
- ✅ Detects errors across all processes
- ✅ Groups related errors together
- ✅ Preserves full stack traces
- ✅ Highlights the important parts

### 3. **Process Management Made Easy**
![Process Management](./img/screenshots/process-management-preview.png)

- **Visual Status**: Know at a glance what's running
- **Quick Controls**: Stop, restart, or start new processes
- **Resource Monitoring**: Track CPU and memory usage
- **Smart Grouping**: Related processes stay together

### 4. **Powerful Log Management**
```bash
# Filter logs instantly
/show error
/hide webpack
/show api

# Search across all processes
/search "user authentication"
```

### 5. **IDE Integration**
Connect your favorite tools:
- **VS Code** ✅
- **Cursor** ✅
- **Claude Code** ✅
- **Windsurf** ✅
- **And more...**

## Perfect For

### 🚀 Startups
- Onboard developers faster
- Standardize development environments
- Reduce context switching
- Improve debugging efficiency

### 🏢 Enterprise Teams
- Manage complex microservices
- Monitor multiple environments
- Integrate with CI/CD pipelines
- Maintain consistency across teams

### 👩‍💻 Individual Developers
- Simplify daily workflows
- Learn from better error messages
- Focus on coding, not terminal management
- Boost productivity

## Core Features at a Glance

| Feature | Description |
|---------|-------------|
| **Multi-Package Support** | npm, yarn, pnpm, bun - all supported |
| **Monorepo Ready** | Turborepo, Nx, Lerna, Rush integration |
| **Smart Detection** | Auto-discovers scripts and commands |
| **Error Intelligence** | Contextual error detection and grouping |
| **Process Control** | Start, stop, restart with one key |
| **Log Filtering** | Powerful search and filter capabilities |
| **URL Collection** | Auto-detects and tracks URLs |
| **MCP Server** | API for external tool integration |
| **Cross-Platform** | Works on macOS, Linux, and Windows |

## Quick Wins

### Day 1: Immediate Benefits
- 🎯 All processes in one view
- 🎯 No more lost terminals
- 🎯 Errors are instantly visible
- 🎯 One-key process control

### Week 1: Workflow Transformation
- 📈 Faster debugging with grouped errors
- 📈 Better understanding of process interactions
- 📈 Reduced context switching
- 📈 More productive development

### Month 1: Team Impact
- 🚀 Standardized workflows across team
- 🚀 Faster onboarding for new developers
- 🚀 Fewer environment-related issues
- 🚀 Improved collaboration

## Get Started in 30 Seconds

```bash
# Install
curl -sSL https://raw.githubusercontent.com/beagle/brummer/main/quick-install.sh | bash

# Or with npm
npm install -g brummer

# Run
cd your-project
brum
```

That's it! No configuration needed.

## What's Next?

Ready to transform your development workflow?

1. **[Quick Start](./quick-start)** - Get running in minutes
2. **[First Project Tutorial](./tutorials/first-project)** - Step-by-step walkthrough
3. **[Examples](./examples/react-development)** - Real-world use cases
4. **[MCP Integration](./mcp-integration/overview)** - Connect your IDE

---

<div style="text-align: center; margin-top: 50px;">
  <h3>Join thousands of developers who've simplified their workflow with Brummer</h3>
  <p>
    <a href="https://github.com/beagle/brummer" target="_blank">⭐ Star on GitHub</a> | 
    <a href="./installation">📦 Install Now</a> | 
    <a href="./tutorials/first-project">📚 Tutorial</a>
  </p>
</div>