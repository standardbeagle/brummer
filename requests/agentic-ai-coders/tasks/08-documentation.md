# Task: Documentation for AI Coders
**Generated from Master Planning**: 2025-01-28
**Context Package**: `/requests/agentic-ai-coders/context/`
**Next Phase**: [subtasks-execute.md](../subtasks-execute.md)

## Task Sizing Assessment
**File Count**: 5 files - Within target range (3-7 files)
**Estimated Time**: 22 minutes - Within target (15-30min)
**Token Estimate**: 110k tokens - Within target (<150k)
**Complexity Level**: 1 (Simple) - Documentation with established patterns
**Parallelization Benefit**: LOW - Requires implementation completion
**Atomicity Assessment**: ✅ ATOMIC - Complete documentation package
**Boundary Analysis**: ✅ CLEAR - Documentation files with clear purpose

## Persona Assignment
**Persona**: Technical Writer (Developer Documentation)
**Expertise Required**: Technical writing, API documentation, user guides
**Worktree**: `~/work/worktrees/agentic-ai-coders/08-documentation/`

## Context Summary
**Risk Level**: LOW (documentation task, well-established patterns)
**Integration Points**: All AI coder components, existing documentation structure
**Architecture Pattern**: Documentation Structure Pattern (from existing docs)
**Similar Reference**: Existing documentation files in `/docs/` directory

### Codebase Context (from master analysis)
**Files in Scope**:
```yaml
read_files:   [docs/mcp-tools.md, docs/tui-usage.md, README.md]
modify_files: [README.md]
create_files: [
  /docs/ai-coders.md,
  /docs/ai-coder-configuration.md,
  /docs/ai-coder-providers.md,
  /docs/ai-coder-troubleshooting.md
]
# Total: 5 files (1 modify, 4 create) - complete documentation suite
```

**Existing Patterns to Follow**:
- Documentation structure from existing `/docs/` files
- Markdown formatting standards and conventions
- Code example formatting and syntax highlighting

**Dependencies Context**:
- Documentation for all components from Tasks 01-07
- Integration with existing Brummer documentation structure
- User guide and API reference documentation

### Task Scope Boundaries
**MODIFY Zone** (Direct Changes):
```yaml
primary_files:
  - /README.md                                 # Add AI coder section
  - /docs/ai-coders.md                         # Main AI coder documentation
  - /docs/ai-coder-configuration.md            # Configuration guide
  - /docs/ai-coder-providers.md                # Provider setup and usage
  - /docs/ai-coder-troubleshooting.md          # Troubleshooting guide

direct_dependencies: []                        # Documentation files only
```

**REVIEW Zone** (Check for Impact):
```yaml
check_integration:
  - /docs/mcp-tools.md                         # MCP tool documentation updates
  - /docs/tui-usage.md                         # TUI usage documentation
  - /docs/configuration.md                     # Configuration documentation

check_documentation:
  - /docs/README.md                            # Documentation index updates
```

**IGNORE Zone** (Do Not Touch):
```yaml
ignore_completely:
  - /internal/                                 # Implementation code
  - /pkg/                                      # Package code
  - /cmd/                                      # Command code

ignore_search_patterns:
  - "**/testdata/**"                           # Test data
  - "**/vendor/**"                             # Third-party code
  - "**/node_modules/**"                       # JavaScript dependencies
```

**Boundary Analysis Results**:
- **Usage Count**: Limited to documentation files
- **Scope Assessment**: LIMITED scope - documentation only
- **Impact Radius**: 1 file to modify, 4 new documentation files

### External Context Sources (from master research)
**Primary Documentation**:
- [Documentation Best Practices](https://www.writethedocs.org/guide/) - Technical writing standards
- [Markdown Guide](https://www.markdownguide.org/) - Markdown formatting
- [API Documentation Standards](https://swagger.io/resources/articles/best-practices-in-api-documentation/) - API documentation

**Standards Applied**:
- Clear headings and section organization
- Code examples with syntax highlighting
- Step-by-step tutorials and guides
- Troubleshooting section with common issues

**Reference Implementation**:
- Existing documentation structure in `/docs/` directory
- README.md formatting and organization
- Code example formatting from existing docs

## Task Requirements
**Objective**: Create comprehensive documentation for AI coder functionality

**Success Criteria**:
- [ ] Main AI coder documentation with overview and getting started
- [ ] Configuration guide with all options and examples
- [ ] Provider setup guide for Claude, OpenAI, and local models
- [ ] Troubleshooting guide with common issues and solutions
- [ ] README.md updated with AI coder feature description
- [ ] Integration with existing documentation structure
- [ ] Code examples tested and verified
- [ ] Screenshots and diagrams where helpful

**Documentation Areas to Cover**:
1. **Overview and Getting Started** - What AI coders are and how to use them
2. **Configuration Guide** - Complete configuration reference
3. **Provider Setup** - Setting up different AI providers
4. **TUI Usage** - Using AI coders in the terminal interface
5. **MCP Integration** - Controlling AI coders via MCP tools
6. **Troubleshooting** - Common issues and solutions

**Validation Commands**:
```bash
# Documentation Verification
grep -q "AI Coder" README.md                           # README updated
ls docs/ai-coder*.md                                   # Documentation files exist
markdownlint docs/ai-coder*.md                        # Markdown formatting valid
grep -q "ai_coder_create" docs/ai-coders.md           # MCP tools documented
```

## Implementation Specifications

### Main AI Coder Documentation
```markdown
# docs/ai-coders.md

# AI Coders

Brummer's AI Coder feature enables you to launch and manage autonomous AI coding assistants directly within your development environment. AI coders can help with code generation, refactoring, documentation, and complex development tasks while maintaining full workspace isolation and security.

## Overview

AI Coders are autonomous coding assistants that:
- Run in isolated workspaces with security boundaries
- Support multiple AI providers (Claude, OpenAI, local models)
- Integrate seamlessly with Brummer's TUI and MCP tools
- Provide real-time progress tracking and monitoring
- Maintain full audit logs of all operations

## Quick Start

### 1. Configuration

First, configure AI coders in your `.brum.toml` file:

```toml
[ai_coders]
enabled = true
max_concurrent = 3
workspace_base_dir = "~/.brummer/ai-coders"
default_provider = "claude"

[ai_coders.providers.claude]
api_key_env = "ANTHROPIC_API_KEY"
model = "claude-3-5-sonnet-20241022"
```

### 2. Set Up API Keys

Set up your AI provider API keys:

```bash
# For Claude (Anthropic)
export ANTHROPIC_API_KEY="your-api-key-here"

# For OpenAI
export OPENAI_API_KEY="your-api-key-here"
```

### 3. Launch Brummer

Start Brummer with AI coder support:

```bash
brum
# Navigate to the "AI Coders" tab (Ctrl+6)
```

### 4. Create Your First AI Coder

Using the TUI:
1. Press `n` to create a new AI coder
2. Enter your coding task description
3. Select an AI provider
4. Press Enter to launch

Using MCP tools:
```bash
# Via Claude Desktop or other MCP client
ai_coder_create {
  "task": "Create a REST API with authentication",
  "provider": "claude"
}
```

## Core Concepts

### AI Coder Lifecycle

AI coders progress through several states:

1. **Creating** - Initial setup and workspace preparation
2. **Running** - Actively working on the assigned task
3. **Paused** - Temporarily suspended (can be resumed)
4. **Completed** - Task finished successfully
5. **Failed** - Task failed with error details
6. **Stopped** - Manually terminated

### Workspace Isolation

Each AI coder operates in an isolated workspace:
- Separate directory structure
- Path validation prevents directory traversal
- Configurable file type restrictions
- Automatic cleanup on completion

### Progress Tracking

AI coders provide real-time progress information:
- Percentage completion
- Current task stage
- Milestone achievements
- Estimated time remaining

## TUI Interface

### AI Coder View

The AI Coder view (`Ctrl+6`) provides:

**Left Panel - Coder List:**
- Status indicators with color coding
- Progress bars showing completion
- Provider and creation time info
- Filter and search capabilities

**Right Panel - Details:**
- Selected coder detailed information
- Workspace file listing
- Progress history and milestones
- Real-time output stream

**Command Input:**
- Send additional instructions to AI coder
- Request clarifications or modifications
- Pause/resume/stop operations

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `n` | Create new AI coder |
| `s` | Start selected AI coder |
| `p` | Pause selected AI coder |
| `r` | Resume paused AI coder |
| `d` | Delete AI coder |
| `Enter` | Send command to AI coder |
| `Tab` | Switch focus between panels |
| `/` | Search/filter AI coders |

## MCP Integration

AI coders are fully controllable via MCP tools:

### Core Tools

**`ai_coder_create`** - Create new AI coder
```json
{
  "task": "Implement user authentication system",
  "provider": "claude",
  "workspace_files": ["main.go", "auth.go"]
}
```

**`ai_coder_list`** - List all AI coders
```json
{
  "status_filter": "running",
  "limit": 20
}
```

**`ai_coder_control`** - Control AI coder operations
```json
{
  "coder_id": "ai-coder-123",
  "action": "pause"
}
```

**`ai_coder_status`** - Get detailed status
```json
{
  "coder_id": "ai-coder-123"
}
```

**`ai_coder_workspace`** - Access workspace files
```json
{
  "coder_id": "ai-coder-123",
  "operation": "list"
}
```

**`ai_coder_logs`** - Stream execution logs
```json
{
  "coder_id": "ai-coder-123",
  "follow": true
}
```

### Integration Examples

**With Claude Desktop:**
```javascript
// Create a new AI coder for bug fixing
const result = await claudeCode.callTool('ai_coder_create', {
  task: 'Fix the authentication bug in login.js',
  provider: 'claude',
  workspace_files: ['login.js', 'auth.js', 'test/auth.test.js']
});
```

**With VSCode Extension:**
```javascript
// Monitor AI coder progress
const status = await brummer.callTool('ai_coder_status', {
  coder_id: 'ai-coder-456'
});
console.log(`Progress: ${status.progress * 100}%`);
```

## Event System

AI coders emit events for all operations:

### Event Types

- **Lifecycle Events**: `ai_coder_created`, `ai_coder_started`, `ai_coder_completed`
- **Progress Events**: `ai_coder_progress`, `ai_coder_milestone`
- **Workspace Events**: `ai_coder_file_created`, `ai_coder_file_modified`
- **Provider Events**: `ai_coder_api_call`, `ai_coder_rate_limit`

### Event Subscription

```go
// Subscribe to AI coder events
eventBus.Subscribe("ai_coder_progress", func(data interface{}) {
    event := data.(AICoderProgressEvent)
    fmt.Printf("Coder %s: %.1f%% complete\n", 
        event.CoderID, event.Progress*100)
})
```

## Best Practices

### Task Description

Write clear, specific task descriptions:

**Good:**
```
Create a REST API for user management with the following endpoints:
- POST /users (create user)
- GET /users/:id (get user by ID)  
- PUT /users/:id (update user)
- DELETE /users/:id (delete user)
Include input validation and error handling.
```

**Avoid:**
```
Make a user API
```

### Provider Selection

Choose the right provider for your task:

- **Claude**: Best for complex reasoning and code architecture
- **OpenAI GPT-4**: Strong general-purpose coding abilities
- **Local Models**: Good for privacy-sensitive work, lower cost

### Resource Management

Monitor resource usage:
- Set appropriate `max_concurrent` limits
- Configure resource limits per coder
- Enable auto-cleanup for completed tasks
- Monitor disk space in workspace directory

### Security Considerations

- Never expose API keys in configuration files
- Use workspace isolation for untrusted tasks
- Review generated code before deployment
- Configure allowed file extensions appropriately

## Integration with Development Workflow

### Git Integration

AI coders work well with version control:

```bash
# Create feature branch for AI coder work
git checkout -b feature/ai-generated-auth

# Let AI coder work in workspace
ai_coder_create "Implement OAuth2 authentication"

# Review and copy generated code
cp ~/.brummer/ai-coders/ai-coder-123/* ./src/

# Commit with proper attribution
git add .
git commit -m "feat: implement OAuth2 authentication

Generated with AI assistance"
```

### Testing Integration

Integrate AI coders with your testing workflow:

```bash
# Create AI coder for test generation
ai_coder_create "Generate unit tests for user.go with >90% coverage"

# Run tests after generation
go test ./... -cover
```

### Code Review

Use AI coders for code review assistance:

```bash
# Create AI coder for code review
ai_coder_create "Review auth.go for security vulnerabilities and suggest improvements"
```

## Monitoring and Debugging

### Logs

AI coder logs are available through multiple channels:

**TUI View:**
- Real-time log stream in detail panel
- Filterable by log level and component

**MCP Tools:**
```bash
ai_coder_logs {
  "coder_id": "ai-coder-123",
  "follow": true,
  "level": "error"
}
```

**Log Files:**
```bash
# AI coder specific logs
tail -f ~/.brummer/logs/ai-coders.log

# System logs with AI coder events
tail -f ~/.brummer/logs/brummer.log | grep ai_coder
```

### Metrics

Monitor AI coder performance:

```bash
# Via MCP
ai_coder_list {"status_filter": "all"}

# Via TUI status bar
# Shows: Active: 2, Completed: 15, Failed: 1
```

### Health Checks

Verify AI coder system health:

```bash
# Check provider connectivity
ai_coder_create {
  "task": "Echo test",
  "provider": "claude"
}

# Monitor resource usage
ps aux | grep ai-coder
df -h ~/.brummer/ai-coders/
```

## Advanced Usage

### Custom Providers

Implement custom AI providers:

```go
type CustomProvider struct {
    apiEndpoint string
    apiKey      string
}

func (p *CustomProvider) GenerateCode(ctx context.Context, prompt string, options GenerateOptions) (*GenerateResult, error) {
    // Custom implementation
}

// Register provider
manager.RegisterProvider("custom", &CustomProvider{
    apiEndpoint: "https://api.custom.ai/v1",
    apiKey: os.Getenv("CUSTOM_API_KEY"),
})
```

### Workflow Automation

Automate AI coder workflows:

```bash
#!/bin/bash
# ai-workflow.sh

# Create AI coder for feature implementation
CODER_ID=$(ai_coder_create "Implement feature X" | grep "ID:" | cut -d: -f2)

# Wait for completion
while [ "$(ai_coder_status $CODER_ID | grep Status)" != "completed" ]; do
    sleep 30
done

# Copy results and commit
cp ~/.brummer/ai-coders/$CODER_ID/* ./src/
git add . && git commit -m "feat: implement feature X (AI-generated)"
```

### Multi-Coder Coordination

Coordinate multiple AI coders:

```bash
# Create frontend and backend coders simultaneously
FRONTEND_ID=$(ai_coder_create "Create React frontend" | grep ID | cut -d: -f2)
BACKEND_ID=$(ai_coder_create "Create Go backend API" | grep ID | cut -d: -f2)

# Monitor both
ai_coder_list | grep -E "($FRONTEND_ID|$BACKEND_ID)"
```

## Migration and Upgrade

### Upgrading from Previous Versions

When upgrading Brummer with AI coder support:

1. **Backup existing workspaces:**
   ```bash
   cp -r ~/.brummer/ai-coders ~/.brummer/ai-coders.backup
   ```

2. **Update configuration:**
   ```bash
   # Add new AI coder settings to .brum.toml
   ```

3. **Verify provider setup:**
   ```bash
   # Test API connectivity
   ai_coder_create "test connectivity"
   ```

### Migrating Workspaces

Move AI coder workspaces:

```bash
# Update configuration
[ai_coders]
workspace_base_dir = "/new/path/ai-coders"

# Move existing workspaces
mv ~/.brummer/ai-coders/* /new/path/ai-coders/
```

For more detailed configuration options, see [AI Coder Configuration](ai-coder-configuration.md).

For provider-specific setup instructions, see [AI Coder Providers](ai-coder-providers.md).

For troubleshooting common issues, see [AI Coder Troubleshooting](ai-coder-troubleshooting.md).
```

### Configuration Documentation
```markdown
# docs/ai-coder-configuration.md

# AI Coder Configuration

This guide covers all configuration options for Brummer's AI Coder feature.

## Configuration File Structure

AI coder settings are configured in the `[ai_coders]` section of your `.brum.toml` file:

```toml
[ai_coders]
enabled = true
max_concurrent = 3
workspace_base_dir = "~/.brummer/ai-coders"
default_provider = "claude"
timeout_minutes = 30
auto_cleanup = true
cleanup_after_hours = 24

[ai_coders.providers]
# Provider configurations...

[ai_coders.resource_limits]
# Resource limit configurations...

[ai_coders.workspace]
# Workspace security settings...

[ai_coders.logging]
# Logging configurations...
```

## Global Settings

### Basic Configuration

| Setting | Type | Default | Description |
|---------|------|---------|-------------|
| `enabled` | boolean | `true` | Enable/disable AI coder functionality |
| `max_concurrent` | integer | `3` | Maximum number of concurrent AI coders |
| `workspace_base_dir` | string | `"~/.brummer/ai-coders"` | Base directory for AI coder workspaces |
| `default_provider` | string | `"claude"` | Default AI provider to use |
| `timeout_minutes` | integer | `30` | Default timeout for AI coder operations |

### Cleanup Settings

| Setting | Type | Default | Description |
|---------|------|---------|-------------|
| `auto_cleanup` | boolean | `true` | Automatically clean up completed workspaces |
| `cleanup_after_hours` | integer | `24` | Hours to wait before cleaning up workspaces |

## Provider Configuration

Configure AI providers in the `[ai_coders.providers]` section:

### Claude (Anthropic)

```toml
[ai_coders.providers.claude]
api_key_env = "ANTHROPIC_API_KEY"
model = "claude-3-5-sonnet-20241022"
max_tokens = 4096
temperature = 0.7
request_timeout_seconds = 30

[ai_coders.providers.claude.rate_limit]
requests_per_minute = 50
tokens_per_minute = 150000
```

**Claude Configuration Options:**

| Setting | Type | Default | Description |
|---------|------|---------|-------------|
| `api_key_env` | string | `"ANTHROPIC_API_KEY"` | Environment variable containing API key |
| `model` | string | `"claude-3-5-sonnet-20241022"` | Claude model to use |
| `max_tokens` | integer | `4096` | Maximum tokens per request |
| `temperature` | float | `0.7` | Sampling temperature (0.0-2.0) |
| `request_timeout_seconds` | integer | `30` | Request timeout in seconds |

**Available Claude Models:**
- `claude-3-5-sonnet-20241022` - Best for complex reasoning and code
- `claude-3-haiku-20240307` - Faster, good for simple tasks
- `claude-3-opus-20240229` - Most capable but slower

### OpenAI

```toml
[ai_coders.providers.openai]
api_key_env = "OPENAI_API_KEY"
model = "gpt-4"
max_tokens = 4096
temperature = 0.7
request_timeout_seconds = 30

[ai_coders.providers.openai.rate_limit]
requests_per_minute = 60
tokens_per_minute = 200000
```

**OpenAI Configuration Options:**

| Setting | Type | Default | Description |
|---------|------|---------|-------------|
| `api_key_env` | string | `"OPENAI_API_KEY"` | Environment variable containing API key |
| `model` | string | `"gpt-4"` | OpenAI model to use |
| `max_tokens` | integer | `4096` | Maximum tokens per request |
| `temperature` | float | `0.7` | Sampling temperature (0.0-2.0) |
| `request_timeout_seconds` | integer | `30` | Request timeout in seconds |

**Available OpenAI Models:**
- `gpt-4` - Best overall performance
- `gpt-4-turbo` - Faster with large context window
- `gpt-3.5-turbo` - Faster and cheaper for simple tasks

### Local Models

```toml
[ai_coders.providers.local]
base_url = "http://localhost:11434"
model = "codellama"
max_tokens = 2048
temperature = 0.7
request_timeout_seconds = 60
```

**Local Provider Configuration:**

| Setting | Type | Default | Description |
|---------|------|---------|-------------|
| `base_url` | string | `"http://localhost:11434"` | Local model server URL |
| `model` | string | `"codellama"` | Local model name |
| `max_tokens` | integer | `2048` | Maximum tokens per request |
| `temperature` | float | `0.7` | Sampling temperature |
| `request_timeout_seconds` | integer | `60` | Request timeout (longer for local) |

**Supported Local Model Servers:**
- **Ollama** - `http://localhost:11434`
- **LocalAI** - `http://localhost:8080`
- **Text Generation WebUI** - `http://localhost:5000`

### Rate Limiting

Configure rate limits to avoid API quotas:

```toml
[ai_coders.providers.claude.rate_limit]
requests_per_minute = 50
tokens_per_minute = 150000

[ai_coders.providers.openai.rate_limit]
requests_per_minute = 60
tokens_per_minute = 200000
```

## Resource Limits

Control resource usage for AI coders:

```toml
[ai_coders.resource_limits]
max_memory_mb = 512
max_disk_space_mb = 1024
max_cpu_percent = 50
max_processes = 5
max_files_per_coder = 100
```

**Resource Limit Options:**

| Setting | Type | Default | Description |
|---------|------|---------|-------------|
| `max_memory_mb` | integer | `512` | Maximum memory usage per AI coder (MB) |
| `max_disk_space_mb` | integer | `1024` | Maximum disk space per workspace (MB) |
| `max_cpu_percent` | integer | `50` | Maximum CPU usage per AI coder (%) |
| `max_processes` | integer | `5` | Maximum child processes per AI coder |
| `max_files_per_coder` | integer | `100` | Maximum files per workspace |

## Workspace Security

Configure workspace isolation and security:

```toml
[ai_coders.workspace]
template = "basic"
gitignore_rules = ["node_modules/", ".env", "*.log"]
allowed_extensions = [".go", ".js", ".ts", ".py", ".md", ".json", ".yaml", ".toml"]
forbidden_paths = ["/etc", "/var", "/sys", "/proc"]
max_file_size_mb = 10
backup_enabled = true
```

**Workspace Security Options:**

| Setting | Type | Default | Description |
|---------|------|---------|-------------|
| `template` | string | `"basic"` | Workspace template to use |
| `gitignore_rules` | array | `["node_modules/", ".env", "*.log"]` | Files to ignore in workspace |
| `allowed_extensions` | array | See default | File extensions AI coders can create |
| `forbidden_paths` | array | `["/etc", "/var", "/sys", "/proc"]` | Paths AI coders cannot access |
| `max_file_size_mb` | integer | `10` | Maximum file size AI coders can create |
| `backup_enabled` | boolean | `true` | Enable workspace backups |

**Available Workspace Templates:**
- `basic` - Empty workspace
- `go` - Go project structure with go.mod
- `node` - Node.js project with package.json
- `python` - Python project with requirements.txt

## Logging Configuration

Configure AI coder specific logging:

```toml
[ai_coders.logging]
level = "info"
output_file = "ai-coders.log"
rotate_size_mb = 50
keep_rotations = 5
include_ai_output = false
```

**Logging Options:**

| Setting | Type | Default | Description |
|---------|------|---------|-------------|
| `level` | string | `"info"` | Log level (debug, info, warn, error) |
| `output_file` | string | `"ai-coders.log"` | Log file name |
| `rotate_size_mb` | integer | `50` | Log rotation size (MB) |
| `keep_rotations` | integer | `5` | Number of rotated logs to keep |
| `include_ai_output` | boolean | `false` | Include AI model output in logs |

## Environment Variables

AI coder configuration supports environment variable substitution:

### API Keys

Always store API keys in environment variables:

```bash
# Claude
export ANTHROPIC_API_KEY="your-claude-api-key"

# OpenAI  
export OPENAI_API_KEY="your-openai-api-key"

# Custom provider
export CUSTOM_AI_API_KEY="your-custom-api-key"
```

### Dynamic Configuration

Use environment variables for dynamic settings:

```toml
[ai_coders]
workspace_base_dir = "${AI_WORKSPACE_DIR:-~/.brummer/ai-coders}"
max_concurrent = "${AI_MAX_CONCURRENT:-3}"
```

```bash
# Override defaults with environment variables
export AI_WORKSPACE_DIR="/tmp/ai-coders"
export AI_MAX_CONCURRENT="5"
```

## Configuration Validation

Brummer validates your AI coder configuration on startup:

### Common Validation Errors

**Missing API Key:**
```
Error: AI coder provider 'claude': environment variable ANTHROPIC_API_KEY is not set
```
**Solution:** Set the required environment variable

**Invalid Workspace Directory:**
```
Error: AI coder workspace directory '/invalid/path' is not writable
```
**Solution:** Create directory or fix permissions

**Resource Limit Exceeded:**
```
Error: AI coder max_memory_mb exceeds reasonable limit (8GB)
```
**Solution:** Reduce resource limits to reasonable values

### Configuration Testing

Test your configuration:

```bash
# Validate configuration
brum --validate-config

# Test AI coder creation
ai_coder_create "test configuration"
```

## Performance Tuning

### Optimal Settings for Different Use Cases

**Development Machine (8GB RAM):**
```toml
[ai_coders]
max_concurrent = 2

[ai_coders.resource_limits]
max_memory_mb = 256
max_disk_space_mb = 512
```

**High-Performance Workstation (32GB RAM):**
```toml
[ai_coders]
max_concurrent = 8

[ai_coders.resource_limits]
max_memory_mb = 1024
max_disk_space_mb = 2048
```

**Server Environment:**
```toml
[ai_coders]
max_concurrent = 16

[ai_coders.resource_limits]
max_memory_mb = 2048
max_disk_space_mb = 4096
max_cpu_percent = 80
```

### Provider-Specific Optimizations

**Claude Optimization:**
```toml
[ai_coders.providers.claude]
# Use faster model for simple tasks
model = "claude-3-haiku-20240307"
# Reduce tokens for faster responses
max_tokens = 2048
# Lower temperature for more consistent code
temperature = 0.3
```

**OpenAI Optimization:**
```toml
[ai_coders.providers.openai]
# Use turbo model for speed
model = "gpt-4-turbo"
# Increase timeout for complex tasks
request_timeout_seconds = 60
```

## Troubleshooting Configuration

### Debug Configuration Issues

Enable debug logging:

```toml
[ai_coders.logging]
level = "debug"
include_ai_output = true
```

### Common Configuration Issues

1. **API Key Not Found**
   - Verify environment variable is set
   - Check variable name matches configuration

2. **Permission Denied**
   - Check workspace directory permissions
   - Verify Brummer can create directories

3. **Resource Limits Too Low**
   - Increase memory and disk limits
   - Monitor actual usage and adjust

4. **Rate Limiting**
   - Reduce rate limits if hitting API quotas
   - Consider using local models for high-volume tasks

For more troubleshooting help, see [AI Coder Troubleshooting](ai-coder-troubleshooting.md).
```

### Provider Setup Documentation
```markdown
# docs/ai-coder-providers.md

# AI Coder Providers

This guide covers setting up and configuring different AI providers for Brummer's AI Coder feature.

## Overview

AI Coder providers are the AI services that power autonomous coding assistants. Brummer supports multiple provider types:

- **Cloud Providers** - Claude (Anthropic), OpenAI GPT models
- **Local Models** - Ollama, LocalAI, and other local inference servers
- **Custom Providers** - Implement your own provider interface

## Claude (Anthropic)

Claude is Anthropic's AI assistant, excellent for complex reasoning and code generation.

### Setup

1. **Get API Key:**
   - Sign up at [console.anthropic.com](https://console.anthropic.com)
   - Create a new API key
   - Note your usage limits and pricing

2. **Set Environment Variable:**
   ```bash
   export ANTHROPIC_API_KEY="your-api-key-here"
   ```

3. **Configure Provider:**
   ```toml
   [ai_coders.providers.claude]
   api_key_env = "ANTHROPIC_API_KEY"
   model = "claude-3-5-sonnet-20241022"
   max_tokens = 4096
   temperature = 0.7
   request_timeout_seconds = 30
   
   [ai_coders.providers.claude.rate_limit]
   requests_per_minute = 50
   tokens_per_minute = 150000
   ```

### Model Selection

**Claude 3.5 Sonnet (Recommended)**
- Model: `claude-3-5-sonnet-20241022`
- Best for: Complex coding tasks, architecture decisions
- Strengths: Excellent reasoning, code quality, following instructions
- Cost: Moderate

**Claude 3 Haiku**
- Model: `claude-3-haiku-20240307`
- Best for: Simple coding tasks, quick fixes
- Strengths: Fast responses, cost-effective
- Cost: Low

**Claude 3 Opus**
- Model: `claude-3-opus-20240229` 
- Best for: Most complex tasks requiring deep reasoning
- Strengths: Highest capability, best code quality
- Cost: High

### Usage Examples

```bash
# Create AI coder with specific Claude model
ai_coder_create {
  "task": "Implement complex authentication system",
  "provider": "claude",
  "model": "claude-3-5-sonnet-20241022"
}
```

### Rate Limits and Costs

**API Limits (as of 2024):**
- Tier 1: 50 requests/minute, 40,000 tokens/minute
- Tier 2: 1,000 requests/minute, 80,000 tokens/minute
- Tier 3: 5,000 requests/minute, 400,000 tokens/minute

**Pricing (approximate):**
- Claude 3.5 Sonnet: $3/M input tokens, $15/M output tokens
- Claude 3 Haiku: $0.25/M input tokens, $1.25/M output tokens
- Claude 3 Opus: $15/M input tokens, $75/M output tokens

Configure rate limits accordingly:
```toml
[ai_coders.providers.claude.rate_limit]
requests_per_minute = 45  # Slightly under limit
tokens_per_minute = 35000 # Conservative limit
```

## OpenAI

OpenAI provides GPT models through their API, offering strong general-purpose coding capabilities.

### Setup

1. **Get API Key:**
   - Sign up at [platform.openai.com](https://platform.openai.com)
   - Create API key in API settings
   - Add billing method for usage

2. **Set Environment Variable:**
   ```bash
   export OPENAI_API_KEY="your-api-key-here"
   ```

3. **Configure Provider:**
   ```toml
   [ai_coders.providers.openai]
   api_key_env = "OPENAI_API_KEY"
   model = "gpt-4"
   max_tokens = 4096
   temperature = 0.7
   request_timeout_seconds = 30
   
   [ai_coders.providers.openai.rate_limit]
   requests_per_minute = 60
   tokens_per_minute = 200000
   ```

### Model Selection

**GPT-4 (Recommended)**
- Model: `gpt-4`
- Best for: General coding tasks, debugging, refactoring
- Strengths: Reliable code generation, good at following instructions
- Context: 8,192 tokens

**GPT-4 Turbo**
- Model: `gpt-4-turbo` or `gpt-4-1106-preview`
- Best for: Large codebases, complex analysis
- Strengths: Large context window, faster responses
- Context: 128,000 tokens

**GPT-3.5 Turbo**
- Model: `gpt-3.5-turbo`
- Best for: Simple tasks, cost-effective coding
- Strengths: Fast, inexpensive
- Context: 4,096 tokens

### Usage Examples

```bash
# Use GPT-4 for complex refactoring
ai_coder_create {
  "task": "Refactor this codebase to use dependency injection",
  "provider": "openai",
  "model": "gpt-4"
}

# Use GPT-3.5 for simple tasks
ai_coder_create {
  "task": "Add error handling to these functions",
  "provider": "openai", 
  "model": "gpt-3.5-turbo"
}
```

### Rate Limits and Costs

**API Limits (Tier 1):**
- GPT-4: 500 requests/minute, 10,000 tokens/minute
- GPT-3.5: 3,500 requests/minute, 90,000 tokens/minute

**Pricing (approximate):**
- GPT-4: $30/M input tokens, $60/M output tokens
- GPT-4 Turbo: $10/M input tokens, $30/M output tokens
- GPT-3.5 Turbo: $0.50/M input tokens, $1.50/M output tokens

## Local Models

Local models run on your own hardware, providing privacy and cost benefits.

### Ollama Setup

Ollama is the easiest way to run local models:

1. **Install Ollama:**
   ```bash
   # macOS
   brew install ollama
   
   # Linux
   curl -fsSL https://ollama.ai/install.sh | sh
   
   # Windows
   # Download from https://ollama.ai/download
   ```

2. **Start Ollama:**
   ```bash
   ollama serve
   ```

3. **Install Code Models:**
   ```bash
   # Code Llama (recommended for coding)
   ollama pull codellama:7b
   ollama pull codellama:13b
   ollama pull codellama:34b
   
   # DeepSeek Coder (excellent for coding)
   ollama pull deepseek-coder:6.7b
   ollama pull deepseek-coder:33b
   
   # General purpose models
   ollama pull llama2:7b
   ollama pull mistral:7b
   ```

4. **Configure Brummer:**
   ```toml
   [ai_coders.providers.ollama]
   base_url = "http://localhost:11434"
   model = "codellama:13b"
   max_tokens = 2048
   temperature = 0.3
   request_timeout_seconds = 120
   ```

### LocalAI Setup

LocalAI provides OpenAI-compatible API for local models:

1. **Install LocalAI:**
   ```bash
   # Docker
   docker run -p 8080:8080 --name local-ai -ti localai/localai:latest
   
   # Binary
   curl -Lo local-ai "https://github.com/go-skynet/LocalAI/releases/download/v2.1.0/local-ai-$(uname -s)-$(uname -m)" && chmod +x local-ai && ./local-ai
   ```

2. **Configure Models:**
   Create `models/codellama.yaml`:
   ```yaml
   name: codellama
   backend: llama
   parameters:
     model: codellama-7b-instruct.Q4_K_M.gguf
     temperature: 0.3
     top_k: 40
     top_p: 0.95
   ```

3. **Configure Brummer:**
   ```toml
   [ai_coders.providers.localai]
   base_url = "http://localhost:8080"
   model = "codellama"
   max_tokens = 2048
   temperature = 0.3
   request_timeout_seconds = 180
   ```

### Model Recommendations

**For Coding Tasks:**

**Small Models (7B-13B parameters):**
- `codellama:7b` - Fast, good for simple tasks
- `deepseek-coder:6.7b` - Excellent code quality
- `starcoder:7b` - Good for code completion

**Medium Models (13B-33B parameters):**
- `codellama:13b` - Best balance of speed and quality
- `deepseek-coder:33b` - Highest quality for local models
- `phind-codellama:34b` - Optimized for reasoning

**Large Models (70B+ parameters):**
- `codellama:70b` - Highest capability (requires significant hardware)
- `deepseek-coder:67b` - Excellent but resource intensive

### Hardware Requirements

**Minimum Requirements:**
- 8GB RAM for 7B models
- 16GB RAM for 13B models
- 32GB RAM for 33B models

**Recommended Requirements:**
- 16GB RAM for good performance with 7B models
- 32GB RAM for 13B models
- 64GB+ RAM for 33B+ models
- GPU with 8GB+ VRAM for acceleration

### Performance Optimization

**CPU Optimization:**
```bash
# Set CPU threads for Ollama
export OLLAMA_NUM_THREADS=8

# For LocalAI
./local-ai --threads 8
```

**GPU Acceleration:**
```bash
# Ollama with GPU
ollama pull codellama:13b
# GPU will be used automatically if available

# LocalAI with GPU
docker run --gpus all -p 8080:8080 localai/localai:latest-gpu
```

## Custom Providers

Implement custom providers for specialized AI services:

### Provider Interface

```go
type AIProvider interface {
    Name() string
    GenerateCode(ctx context.Context, prompt string, options GenerateOptions) (*GenerateResult, error)
    StreamGenerate(ctx context.Context, prompt string, options GenerateOptions) (<-chan GenerateUpdate, error)
    ValidateConfig(config ProviderConfig) error
}
```

### Example Implementation

```go
type CustomProvider struct {
    apiKey    string
    baseURL   string
    model     string
    timeout   time.Duration
}

func (p *CustomProvider) Name() string {
    return "custom"
}

func (p *CustomProvider) GenerateCode(ctx context.Context, prompt string, options GenerateOptions) (*GenerateResult, error) {
    // Implement API call to your custom service
    client := &http.Client{Timeout: p.timeout}
    
    requestBody := map[string]interface{}{
        "model":       p.model,
        "prompt":      prompt,
        "max_tokens":  options.MaxTokens,
        "temperature": options.Temperature,
    }
    
    // Make HTTP request and parse response
    // Return GenerateResult with generated code
    
    return &GenerateResult{
        Code:    generatedCode,
        Summary: "Code generated successfully",
    }, nil
}

func (p *CustomProvider) ValidateConfig(config ProviderConfig) error {
    if config.APIKeyEnv == "" {
        return errors.New("API key environment variable required")
    }
    if config.Model == "" {
        return errors.New("model name required")
    }
    return nil
}

// Register custom provider
func RegisterCustomProvider(manager *AICoderManager) {
    provider := &CustomProvider{
        apiKey:  os.Getenv("CUSTOM_API_KEY"),
        baseURL: "https://api.custom-ai.com/v1",
        model:   "custom-model-v1",
        timeout: 30 * time.Second,
    }
    
    manager.RegisterProvider("custom", provider)
}
```

### Configuration

```toml
[ai_coders.providers.custom]
api_key_env = "CUSTOM_API_KEY"
base_url = "https://api.custom-ai.com/v1"
model = "custom-model-v1"
max_tokens = 4096
temperature = 0.7
request_timeout_seconds = 30
```

## Provider Comparison

| Provider | Strengths | Weaknesses | Best For |
|----------|-----------|------------|----------|
| **Claude** | Excellent reasoning, code quality | Higher cost, rate limits | Complex architecture, refactoring |
| **OpenAI** | Reliable, good documentation | Cost, usage tracking | General coding, debugging |
| **Local Models** | Privacy, no API costs, unlimited usage | Setup complexity, hardware requirements | Privacy-sensitive, high-volume |
| **Custom** | Tailored to needs | Development effort | Specialized use cases |

## Cost Management

### Monitoring Usage

Track API usage across providers:

```bash
# Monitor AI coder costs
ai_coder_list | grep -E "(claude|openai)" | wc -l

# Check token usage in logs
grep "tokens_used" ~/.brummer/logs/ai-coders.log
```

### Cost Optimization Strategies

1. **Use Appropriate Models:**
   - Simple tasks: Claude Haiku, GPT-3.5 Turbo
   - Complex tasks: Claude Sonnet, GPT-4
   - Bulk operations: Local models

2. **Set Token Limits:**
   ```toml
   [ai_coders.providers.claude]
   max_tokens = 2048  # Reduce for cost control
   ```

3. **Configure Rate Limits:**
   ```toml
   [ai_coders.providers.claude.rate_limit]
   requests_per_minute = 10  # Reduce to control costs
   ```

4. **Use Local Models for Development:**
   - Development and testing: Local models
   - Production deployments: Cloud providers

## Security Considerations

### API Key Security

- Never commit API keys to version control
- Use environment variables for all keys
- Rotate keys regularly
- Monitor for unauthorized usage

### Network Security

For local models:
- Bind to localhost only by default
- Use authentication if exposing externally
- Consider VPN for remote access

### Workspace Isolation

All providers operate within workspace security boundaries:
- Path validation prevents directory traversal
- File type restrictions limit potential damage
- Resource limits prevent system overload

For troubleshooting provider issues, see [AI Coder Troubleshooting](ai-coder-troubleshooting.md).
```

### Troubleshooting Documentation
```markdown
# docs/ai-coder-troubleshooting.md

# AI Coder Troubleshooting

This guide helps resolve common issues with Brummer's AI Coder feature.

## Common Issues

### 1. AI Coder Creation Fails

**Symptoms:**
- "Failed to create AI coder" error
- AI coder stuck in "Creating" state
- Workspace directory not created

**Possible Causes and Solutions:**

**Provider Not Available:**
```bash
# Check provider configuration
brum --validate-config

# Test provider connectivity
ai_coder_create "test connectivity" --provider claude
```

**API Key Issues:**
```bash
# Verify API key is set
echo $ANTHROPIC_API_KEY
echo $OPENAI_API_KEY

# Check key format (should start with expected prefix)
# Anthropic keys start with "sk-ant-"
# OpenAI keys start with "sk-"
```

**Workspace Directory Issues:**
```bash
# Check directory exists and is writable
ls -la ~/.brummer/ai-coders/
mkdir -p ~/.brummer/ai-coders/
chmod 755 ~/.brummer/ai-coders/
```

**Resource Limits:**
```bash
# Check available disk space
df -h ~/.brummer/ai-coders/

# Check memory usage
free -h

# Reduce resource limits if needed
```

### 2. AI Coder Stuck in Running State

**Symptoms:**
- AI coder shows "Running" but no progress
- No file changes in workspace
- No error messages

**Troubleshooting Steps:**

**Check Logs:**
```bash
# AI coder specific logs
tail -f ~/.brummer/logs/ai-coders.log

# System logs
tail -f ~/.brummer/logs/brummer.log | grep ai_coder

# Provider API logs (if available)
grep "api_call\|api_error" ~/.brummer/logs/ai-coders.log
```

**Check Provider Status:**
```bash
# Test provider directly
curl -X POST https://api.anthropic.com/v1/messages \
  -H "x-api-key: $ANTHROPIC_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"claude-3-haiku-20240307","max_tokens":10,"messages":[{"role":"user","content":"Hello"}]}'
```

**Resource Monitoring:**
```bash
# Check if process is consuming resources
ps aux | grep ai-coder
top -p $(pgrep -f ai-coder)

# Check network activity
netstat -an | grep :443  # HTTPS connections
```

**Recovery Actions:**
```bash
# Pause and resume
ai_coder_control coder-id pause
ai_coder_control coder-id resume

# Check workspace for partial progress
ls -la ~/.brummer/ai-coders/coder-id/
```

### 3. API Rate Limiting

**Symptoms:**
- "Rate limit exceeded" errors
- 429 HTTP status codes in logs
- AI coders failing after working initially

**Solutions:**

**Adjust Rate Limits:**
```toml
[ai_coders.providers.claude.rate_limit]
requests_per_minute = 30    # Reduced from 50
tokens_per_minute = 100000  # Reduced from 150000
```

**Implement Backoff Strategy:**
```toml
[ai_coders.providers.claude]
request_timeout_seconds = 60  # Increased timeout
retry_attempts = 3            # Allow retries
```

**Monitor Usage:**
```bash
# Check recent API calls
grep "rate_limit\|429" ~/.brummer/logs/ai-coders.log | tail -20

# Monitor token usage
grep "tokens_used" ~/.brummer/logs/ai-coders.log | \
  awk '{sum += $NF} END {print "Total tokens:", sum}'
```

### 4. Workspace Permission Issues

**Symptoms:**
- "Permission denied" when creating files
- AI coder fails to write to workspace
- Workspace cleanup fails

**Solutions:**

**Fix Directory Permissions:**
```bash
# Set proper permissions
chmod -R 755 ~/.brummer/ai-coders/
chown -R $USER ~/.brummer/ai-coders/

# Check SELinux/AppArmor (Linux)
ls -Z ~/.brummer/ai-coders/
getenforce  # Check if SELinux is enforcing
```

**Verify Workspace Configuration:**
```bash
# Check workspace base directory
ls -la $(dirname ~/.brummer/ai-coders/)

# Test directory creation
mkdir ~/.brummer/ai-coders/test-workspace
echo "test" > ~/.brummer/ai-coders/test-workspace/test.txt
rm -rf ~/.brummer/ai-coders/test-workspace
```

**Alternative Workspace Location:**
```toml
[ai_coders]
workspace_base_dir = "/tmp/brummer-ai-coders"  # Use /tmp if home dir issues
```

### 5. Local Model Issues

**Symptoms:**
- Connection refused to local model server
- Slow response times from local models
- Out of memory errors

**Ollama Troubleshooting:**

**Check Ollama Status:**
```bash
# Verify Ollama is running
ps aux | grep ollama
curl http://localhost:11434/api/tags

# Check Ollama logs
journalctl -u ollama -f  # systemd
ollama logs              # if running manually
```

**Model Issues:**
```bash
# List installed models
ollama list

# Test model directly
ollama run codellama:7b "Write a hello world function"

# Update model if corrupted
ollama pull codellama:7b
```

**Performance Issues:**
```bash
# Check system resources
htop
nvidia-smi  # if using GPU

# Adjust Ollama settings
export OLLAMA_NUM_THREADS=4
export OLLAMA_GPU_MEMORY=2048  # MB
```

**LocalAI Troubleshooting:**

**Check LocalAI Status:**
```bash
# Docker container
docker ps | grep localai
docker logs localai

# Direct binary
ps aux | grep local-ai
```

**Model Configuration:**
```bash  
# Check model files
ls -la models/
cat models/codellama.yaml

# Test API endpoint
curl http://localhost:8080/v1/models
```

### 6. Memory and Resource Issues

**Symptoms:**
- System becomes unresponsive
- Out of memory errors
- AI coders killed unexpectedly

**Monitoring:**

```bash
# Monitor resource usage
htop
iotop  # disk I/O
nethogs  # network usage

# Check AI coder specific usage
ps aux | grep ai-coder
du -sh ~/.brummer/ai-coders/*
```

**Solutions:**

**Reduce Concurrent AI Coders:**
```toml
[ai_coders]
max_concurrent = 1  # Reduce from default 3
```

**Adjust Resource Limits:**
```toml
[ai_coders.resource_limits]
max_memory_mb = 256      # Reduced from 512
max_disk_space_mb = 512  # Reduced from 1024
```

**System-Level Limits:**
```bash
# Set memory limits using systemd (Linux)
systemd-run --user --slice=ai-coders.slice --property=MemoryMax=2G brum

# Use ulimit for process limits
ulimit -v 2000000  # 2GB virtual memory limit
```

### 7. Network and Connectivity Issues

**Symptoms:**
- Connection timeouts to AI providers
- DNS resolution failures
- SSL/TLS certificate errors

**Network Diagnostics:**

```bash
# Test connectivity
ping api.anthropic.com
ping api.openai.com

# Check DNS resolution
nslookup api.anthropic.com
dig api.anthropic.com

# Test HTTPS connectivity
curl -I https://api.anthropic.com
openssl s_client -connect api.anthropic.com:443
```

**Proxy/Firewall Issues:**

```bash
# Check proxy settings
echo $HTTP_PROXY
echo $HTTPS_PROXY

# Test through proxy
curl --proxy $HTTP_PROXY https://api.anthropic.com
```

**Configure Provider Timeout:**
```toml
[ai_coders.providers.claude]
request_timeout_seconds = 120  # Increased for slow connections
```

### 8. Configuration Issues

**Symptoms:**
- "Configuration validation failed" errors
- AI coders not respecting settings
- Provider not found errors

**Validation:**

```bash
# Validate configuration
brum --validate-config

# Check configuration loading
brum --show-config

# Verify TOML syntax
python3 -c "import toml; toml.load('.brum.toml')"
```

**Common Configuration Errors:**

**Invalid TOML Syntax:**
```toml
# Wrong - missing quotes
api_key_env = ANTHROPIC_API_KEY

# Correct
api_key_env = "ANTHROPIC_API_KEY"
```

**Incorrect Section Names:**
```toml
# Wrong
[ai_coder.providers.claude]

# Correct  
[ai_coders.providers.claude]
```

**Missing Required Fields:**
```toml
# Add all required provider fields
[ai_coders.providers.claude]
api_key_env = "ANTHROPIC_API_KEY"
model = "claude-3-5-sonnet-20241022"  # Required
```

## Debugging Strategies

### 1. Enable Debug Logging

```toml
[ai_coders.logging]
level = "debug"
include_ai_output = true
```

### 2. Isolate the Problem

**Test Minimal Configuration:**
```toml
[ai_coders]
enabled = true
max_concurrent = 1
default_provider = "claude"

[ai_coders.providers.claude]
api_key_env = "ANTHROPIC_API_KEY"
model = "claude-3-haiku-20240307"
```

**Test with Simple Task:**
```bash
ai_coder_create "print hello world in python"
```

### 3. Check Dependencies

**System Dependencies:**
```bash
# Required tools
which curl
which git
python3 --version
go version

# Check file system
df -h
mount | grep $(dirname ~/.brummer)
```

**Process Dependencies:**
```bash
# Check for conflicting processes
ps aux | grep -E "(ollama|localai|ai-coder)"
netstat -tlnp | grep -E "(7777|11434|8080)"
```

### 4. Compare Working vs Non-Working

**Export Working Configuration:**
```bash
# From working system
cp .brum.toml .brum.toml.working
env | grep -E "(ANTHROPIC|OPENAI)" > .env.working
```

**Compare Configurations:**
```bash
diff .brum.toml.working .brum.toml
diff .env.working <(env | grep -E "(ANTHROPIC|OPENAI)")
```

## Recovery Procedures

### 1. Reset AI Coder System

**Stop All AI Coders:**
```bash
# List and stop all active AI coders
ai_coder_list | grep -E "(running|creating)" | \
  awk '{print $1}' | xargs -I {} ai_coder_control {} stop
```

**Clean Workspaces:**
```bash
# Backup existing workspaces
cp -r ~/.brummer/ai-coders ~/.brummer/ai-coders.backup.$(date +%Y%m%d)

# Clean all workspaces
rm -rf ~/.brummer/ai-coders/*

# Or remove specific workspace
rm -rf ~/.brummer/ai-coders/ai-coder-123
```

### 2. Reconfigure from Scratch

**Reset Configuration:**
```bash
# Backup current config
cp .brum.toml .brum.toml.backup

# Create minimal config
cat > .brum.toml << 'EOF'
[ai_coders]
enabled = true
max_concurrent = 1
workspace_base_dir = "~/.brummer/ai-coders"
default_provider = "claude"

[ai_coders.providers.claude]
api_key_env = "ANTHROPIC_API_KEY"
model = "claude-3-haiku-20240307"
max_tokens = 2048
temperature = 0.7
EOF
```

**Test Basic Functionality:**
```bash
# Restart Brummer
brum --validate-config
ai_coder_create "simple hello world test"
```

### 3. Provider-Specific Recovery

**Reset Claude Provider:**
```bash
# Test API key
curl -X POST https://api.anthropic.com/v1/messages \
  -H "x-api-key: $ANTHROPIC_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"claude-3-haiku-20240307","max_tokens":10,"messages":[{"role":"user","content":"test"}]}'

# Use different model if current one has issues
# Change from claude-3-5-sonnet to claude-3-haiku
```

**Reset Local Models:**
```bash
# Restart Ollama
sudo systemctl restart ollama  # or
ollama stop && ollama start

# Re-pull models
ollama pull codellama:7b
```

## Performance Optimization

### 1. System-Level Optimizations

**Increase File Descriptors:**
```bash
# Check current limits
ulimit -n

# Increase limits
echo "* soft nofile 65536" >> /etc/security/limits.conf
echo "* hard nofile 65536" >> /etc/security/limits.conf
```

**Optimize Disk I/O:**
```bash
# Use faster disk for workspaces
ln -sf /tmp/ai-coders ~/.brummer/ai-coders

# Mount tmpfs for temporary workspaces
sudo mount -t tmpfs -o size=2G tmpfs /tmp/ai-coders
```

### 2. Provider Optimizations

**Reduce Latency:**
```toml
[ai_coders.providers.claude]
request_timeout_seconds = 15  # Fail fast
max_tokens = 2048             # Smaller responses
```

**Batch Operations:**
```bash
# Create multiple coders for parallel work
for task in task1 task2 task3; do
  ai_coder_create "$task" &
done
wait  # Wait for all to complete
```

## Getting Help

### 1. Collect Diagnostic Information

```bash
#!/bin/bash
# Create diagnostic report
echo "=== Brummer AI Coder Diagnostics ===" > ai-coder-diagnostics.txt
echo "Date: $(date)" >> ai-coder-diagnostics.txt
echo >> ai-coder-diagnostics.txt

echo "=== System Information ===" >> ai-coder-diagnostics.txt
uname -a >> ai-coder-diagnostics.txt
go version >> ai-coder-diagnostics.txt
echo >> ai-coder-diagnostics.txt

echo "=== Configuration ===" >> ai-coder-diagnostics.txt
cat .brum.toml >> ai-coder-diagnostics.txt
echo >> ai-coder-diagnostics.txt

echo "=== Environment ===" >> ai-coder-diagnostics.txt
env | grep -E "(ANTHROPIC|OPENAI|OLLAMA)" >> ai-coder-diagnostics.txt
echo >> ai-coder-diagnostics.txt

echo "=== Recent Logs ===" >> ai-coder-diagnostics.txt
tail -50 ~/.brummer/logs/ai-coders.log >> ai-coder-diagnostics.txt
echo >> ai-coder-diagnostics.txt

echo "=== Process Status ===" >> ai-coder-diagnostics.txt
ps aux | grep -E "(brum|ollama|ai-coder)" >> ai-coder-diagnostics.txt
echo >> ai-coder-diagnostics.txt

echo "=== Network Status ===" >> ai-coder-diagnostics.txt
netstat -tlnp | grep -E "(7777|11434|8080)" >> ai-coder-diagnostics.txt
```

### 2. Community Support

- **GitHub Issues**: [github.com/standardbeagle/brummer/issues](https://github.com/standardbeagle/brummer/issues)
- **Discussions**: Use GitHub Discussions for questions
- **Documentation**: Check latest docs for updates

### 3. Reporting Bugs

When reporting issues, include:
- Diagnostic information (see script above)
- Steps to reproduce the problem
- Expected vs actual behavior
- Brummer version (`brum --version`)
- Operating system and version

## Prevention

### 1. Regular Maintenance

```bash
# Weekly cleanup
find ~/.brummer/ai-coders -name "*.tmp" -delete
find ~/.brummer/ai-coders -mtime +7 -exec rm -rf {} \;

# Log rotation
logrotate ~/.brummer/logs/ai-coders.log

# Check for updates
brum --check-updates
```

### 2. Monitoring Setup

```bash
# Monitor disk usage
echo "$(du -sh ~/.brummer/ai-coders)" >> ~/.brummer/logs/disk-usage.log

# Monitor API usage
grep "tokens_used" ~/.brummer/logs/ai-coders.log | \
  tail -100 | awk '{sum += $NF} END {print strftime("%Y-%m-%d"), sum}' >> ~/.brummer/logs/api-usage.log
```

### 3. Configuration Validation

Set up automated validation:
```bash
# Add to cron for daily validation
0 9 * * * cd /path/to/project && brum --validate-config || echo "AI Coder config invalid" | mail -s "Brummer Config Error" admin@company.com
```

This troubleshooting guide covers the most common issues with AI Coders. For additional help, refer to the main [AI Coders documentation](ai-coders.md) or reach out to the community.
```

## Risk Mitigation (from master analysis)
**Low-Risk Mitigations**:
- Documentation accuracy - Follow established documentation patterns and validate examples - Testing: All code examples tested and verified
- Content organization - Use clear structure and navigation - Review: Technical review for accuracy and completeness
- User experience - Provide step-by-step guides with troubleshooting - Feedback: User testing and feedback incorporation

**Context Validation**:
- [ ] Documentation structure from existing `/docs/` files successfully applied
- [ ] Code examples tested and verified to work correctly
- [ ] Integration with existing documentation maintained

## Integration with Other Tasks
**Dependencies**: All other tasks (01-07) - Documentation requires complete implementation
**Integration Points**: 
- Documents all components and their interactions
- Provides user guides for all functionality
- Enables user adoption and success

**Shared Context**: Documentation becomes the primary resource for users and developers

## Execution Notes
- **Start Pattern**: Use existing documentation structure from `/docs/` directory
- **Key Context**: Focus on user experience and comprehensive coverage
- **Code Examples**: Test all examples to ensure they work correctly
- **Review Focus**: Technical accuracy and user experience

This task creates comprehensive documentation that enables users to successfully adopt and use all AI coder functionality while maintaining consistency with existing Brummer documentation standards.