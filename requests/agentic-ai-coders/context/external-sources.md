# External Context Sources: Agentic AI Coders

## Primary Documentation

### BubbleTea Framework
- **Documentation**: [BubbleTea Guide](https://github.com/charmbracelet/bubbletea/tree/master/tutorials) - Component composition and view management
- **Key Insights**: Model-View-Update pattern, command handling, subscription patterns
- **Implementation Guidance**: New AI coder view follows existing view patterns, async operations via Cmd

### Model Context Protocol (MCP)
- **Specification**: [MCP 2025-06-18](https://modelcontextprotocol.io/specification/2025-06-18) - Protocol for AI tool integration
- **Key Sections**: Tool registration, resource management, client-server communication
- **Implementation Focus**: New AI coder tools, session management, streaming capabilities

### AI Integration Patterns
- **OpenAI Go SDK**: [OpenAI Go](https://github.com/sashabaranov/go-openai) - Go client for AI API integration
- **Claude API**: [Anthropic API](https://docs.anthropic.com/en/api/getting-started) - Claude integration patterns
- **Key Patterns**: Streaming responses, context management, error handling, rate limiting

### Go Concurrency Patterns
- **Go Blog**: [Concurrency Patterns](https://blog.golang.org/pipelines) - Pipeline and worker pool patterns
- **Relevance**: AI coder task management, async processing, resource coordination
- **Application**: Background AI tasks, status monitoring, event processing

## Industry Standards & Best Practices

### Security Standards
- **OWASP Go Security**: [Go Security Guide](https://owasp.org/www-project-go-secure-coding-practices-guide/) - Secure coding practices
- **Applicable Guidelines**: Input validation, sandbox isolation, credential management
- **Threat Model**: AI coder workspace isolation, API key protection, file system security

### Performance Standards
- **Go Performance**: [Go Performance Tips](https://github.com/golang/go/wiki/Performance) - Memory management and optimization
- **Target Metrics**: <100ms TUI response time, <500MB memory per AI coder instance
- **Optimization**: Connection pooling, context reuse, efficient event handling

### API Design Standards
- **REST Guidelines**: [Go API Standards](https://github.com/golang-standards/project-layout) - Project structure and API design
- **MCP Compliance**: Tool naming conventions, error handling, resource management
- **Implementation**: Consistent tool interfaces, structured error responses

## Reference Implementations

### Similar AI Integration Projects
- **Continue.dev**: [Continue VSCode Extension](https://github.com/continuedev/continue) - AI coding assistant integration
- **Pattern Demonstrated**: AI coder lifecycle, workspace management, streaming UI updates
- **Adaptation Needed**: Terminal UI instead of VSCode, MCP integration, Go implementation

### Process Management Examples
- **Docker CLI**: [Docker CLI Source](https://github.com/docker/cli) - Container lifecycle management
- **Integration Pattern**: Process spawning, status monitoring, resource isolation
- **Lessons Learned**: Graceful shutdown, resource cleanup, status aggregation

### Terminal UI References
- **Lazygit**: [Lazygit TUI](https://github.com/jesseduffield/lazygit) - Complex TUI with multiple views
- **Architecture Pattern**: View switching, keyboard navigation, async operations
- **Trade-offs**: Performance vs features, keyboard shortcuts, visual hierarchy

### Event-Driven Architecture
- **NATS Messaging**: [NATS Go Client](https://github.com/nats-io/nats.go) - Pub/sub messaging patterns
- **System Design**: Event sourcing, async processing, error handling
- **Scalability Patterns**: Worker pools, backpressure, graceful degradation

## Standards Applied

### Coding Standards
- **Go Style Guide**: [Effective Go](https://golang.org/doc/effective_go.html) - Naming, error handling, concurrency
- **Specific Rules**: 
  - Package naming: `ai_coder` or `aicoder` (avoid mixed case)
  - Interface naming: `AICoderManager`, `WorkspaceIsolator`
  - Error handling: Explicit error returns, context propagation

### API Design Standards
- **MCP Tool Naming**: [MCP Tool Guidelines](https://modelcontextprotocol.io/specification/2025-06-18#tools) - Consistent naming conventions
- **Naming Patterns**: `ai_coder_create`, `ai_coder_status`, `ai_coder_workspace`
- **Error Handling**: Structured error responses with codes and messages

### Testing Standards
- **Go Testing**: [Go Testing Guide](https://golang.org/doc/tutorial/add-a-test) - Unit and integration testing
- **Coverage Requirements**: >80% coverage for AI coder core logic
- **Test Patterns**: Table-driven tests, mock AI providers, integration test isolation

### Security Standards
- **Credential Management**: Environment variables, no hardcoded secrets
- **Workspace Isolation**: Chroot/container-like restrictions for AI coder file access
- **API Security**: Rate limiting, input validation, secure credential storage

## External Dependencies Assessment

### AI Provider APIs
- **OpenAI API**: Rate limits, pricing considerations, streaming support
- **Claude API**: Context window management, tool use capabilities
- **Local Models**: Ollama integration, resource requirements, performance trade-offs

### System Dependencies
- **File System**: Workspace sandboxing requirements (potentially containers/chroot)
- **Process Management**: Resource limits, cgroup integration
- **Network**: Outbound API access, proxy support, timeout handling

### Development Tools
- **Testing Frameworks**: Testify for assertions, httptest for API mocking
- **Build Tools**: Go modules, cross-platform compilation
- **Documentation**: godoc integration, README maintenance

## Implementation Decision Framework

### AI Provider Selection
- **Criteria**: API stability, Go SDK quality, feature completeness, cost
- **Default Choice**: Multiple provider support with plugin architecture
- **Configuration**: Provider selection via configuration files

### Workspace Isolation Strategy
- **Options**: Directory restrictions, containers, virtual filesystems
- **Recommendation**: Start with directory restrictions, evolve to containers
- **Trade-offs**: Security vs complexity, performance vs isolation

### UI Architecture Approach
- **Pattern**: Follow existing BubbleTea view pattern from TUI
- **Components**: Reuse existing list, viewport, textinput components
- **Extensions**: Custom AI coder status components, progress indicators

### Error Handling Strategy
- **Approach**: Structured errors with context, graceful degradation
- **User Experience**: Clear error messages, recovery suggestions
- **Logging**: Comprehensive logging for debugging and monitoring