# Environment Variable Management Feature Specification

## Overview
Brummer will provide comprehensive environment variable management capabilities, allowing developers to manage, view, and distribute environment variables across scripts, processes, and AI agents in a unified interface.

## Core Features

### 1. Environment File Management
- **Multi-format Support**: Handle various .env file formats
  - `.env` - Base environment file
  - `.env.local` - Local overrides (gitignored)
  - `.env.development`, `.env.production`, `.env.test` - Environment-specific
  - `.env.example` - Template files
  - Custom named files (e.g., `.env.staging`)

- **File Precedence System**:
  ```
  Priority (highest to lowest):
  1. Runtime/CLI overrides
  2. .env.local
  3. .env.[environment]
  4. .env
  5. System environment
  ```

### 2. TUI Environment View
New TUI view accessible via Tab navigation showing:

```
┌─────────────────────────────────────────────────────────────┐
│ Environment Variables (dev) - 42 variables                  │
├─────────────────────────────────────────────────────────────┤
│ Name                 Value             Source               │
│ ─────────────────────────────────────────────────────────── │
│ NODE_ENV            development       .env.development      │
│ API_URL             http://local...   .env.local           │
│ DATABASE_URL        postgres://...    .env (masked)        │
│ > STRIPE_KEY        sk_test_***       .env.local (secret)  │
│ PORT                3000              .env                 │
│ DEBUG               true              CLI override         │
├─────────────────────────────────────────────────────────────┤
│ [e]dit [a]dd [d]elete [r]eload [f]ilter [m]ask/unmask     │
│ [Tab] Switch View [Enter] Edit [/] Search [q] Quit         │
└─────────────────────────────────────────────────────────────┘
```

Features:
- Color coding by source
- Secret masking (automatic for keys containing SECRET, KEY, TOKEN, PASSWORD)
- Quick edit capability
- Search/filter functionality
- Export to file

### 3. Script Integration
Automatic environment injection for npm/yarn/pnpm scripts:

```toml
# .brum.toml
[environment]
auto_load = true
files = [".env", ".env.local"]
script_injection = true

[environment.overrides]
# Override specific vars for specific scripts
test = { NODE_ENV = "test", DATABASE_URL = "sqlite::memory:" }
build = { NODE_ENV = "production" }
```

### 4. AI Agent Integration
Environment variables automatically available to AI coders:

```go
// AI agents receive sanitized environment
type AIEnvironment struct {
    Variables map[string]string
    Secrets   []string // List of secret var names (values masked)
}
```

Features:
- Automatic secret detection and masking
- AI can request specific env vars through prompts
- Environment context included in AI coder workspace

### 5. MCP Tools

#### env_list
List all environment variables with metadata:
```json
{
  "variables": [
    {
      "name": "NODE_ENV",
      "value": "development",
      "source": ".env.development",
      "is_secret": false
    }
  ]
}
```

#### env_get
Get specific environment variable:
```json
{
  "name": "API_URL",
  "value": "https://api.example.com",
  "source": ".env.local",
  "is_secret": false,
  "overridden_by": null
}
```

#### env_set
Set environment variable (updates appropriate file):
```json
{
  "name": "NEW_VAR",
  "value": "new_value",
  "file": ".env.local",
  "temporary": false
}
```

#### env_export
Export current environment state:
```json
{
  "format": "dotenv|json|shell",
  "include_secrets": false,
  "output_file": "environment.env"
}
```

### 6. Security Features

#### Secret Management
- Automatic detection of sensitive variables
- Masked display in TUI (show/hide toggle with 'm' key)
- Encryption support for `.env.encrypted` files
- Integration with system keychain/credential managers

#### Access Control
- Read-only mode for production environments
- Audit logging for environment changes
- Git integration to prevent committing secrets

### 7. Advanced Features

#### Variable Validation
```toml
# .brum.toml
[environment.validation]
required = ["NODE_ENV", "API_URL", "DATABASE_URL"]

[environment.validation.rules]
PORT = { type = "number", min = 1000, max = 9999 }
NODE_ENV = { enum = ["development", "test", "production"] }
API_URL = { pattern = "^https?://" }
```

#### Variable Templates
Support for dynamic values:
```env
# .env.example
API_URL=https://api.${STAGE}.example.com
BUILD_TIME=${timestamp}
GIT_COMMIT=${git:commit:short}
```

#### Multi-Project Support
In hub mode, manage environment across instances:
```
hub_env_sync - Synchronize environment variables across instances
hub_env_diff - Compare environments between instances
```

## Implementation Phases

### Phase 1: Core Functionality
- [ ] Basic .env file parsing and loading
- [ ] Environment variable merging with precedence
- [ ] TUI view for viewing variables
- [ ] Basic script integration

### Phase 2: Advanced Management
- [ ] Multi-file support with precedence
- [ ] Edit capabilities in TUI
- [ ] MCP tools implementation
- [ ] Secret detection and masking

### Phase 3: Security & Integration
- [ ] Encryption support
- [ ] AI coder integration
- [ ] Validation system
- [ ] Template support
- [ ] Hub mode features

## Technical Implementation

### Package Structure
```
/internal/env/
├── parser.go      # .env file parsing
├── manager.go     # Environment management core
├── merger.go      # Multi-source merging logic
├── secrets.go     # Secret detection and handling
└── validator.go   # Validation rules engine

/internal/tui/
└── env_view.go    # TUI environment view

/internal/mcp/
└── env_tools.go   # MCP environment tools
```

### Key Interfaces
```go
type EnvManager interface {
    Load(files ...string) error
    Get(key string) (value string, source string)
    Set(key, value, file string) error
    List() []EnvVariable
    Export(format string) ([]byte, error)
    Validate() []ValidationError
}

type EnvVariable struct {
    Name      string
    Value     string
    Source    string
    IsSecret  bool
    Overrides []Override
}
```

## User Benefits

1. **Unified Management**: Single interface for all environment configuration
2. **Safety**: Automatic secret detection prevents accidental exposure
3. **Productivity**: Quick access and editing without leaving Brummer
4. **AI Integration**: AI agents can work with proper environment context
5. **Team Collaboration**: Share configurations safely with .env.example
6. **Debugging**: Clear visibility into which values are active and their sources

## Success Metrics

- Reduce environment-related debugging time by 50%
- Zero accidental secret commits with git integration
- 90% of users adopt environment view for daily development
- AI agents successfully use environment context in 95% of tasks