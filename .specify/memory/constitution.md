<!--
Sync Impact Report:
- Version change: None → 1.0.0 (new constitution)
- Modified principles: None (all newly created)
- Added sections: All sections (Core Principles, Performance & Concurrency Standards, Development Quality Gates, Governance)
- Removed sections: None
- Templates requiring updates: ✅ plan-template.md (Constitution Check alignment), ✅ spec-template.md (scope/requirements), ✅ tasks-template.md (task categorization), ⚠ pending: .claude/commands/constitution.md (self-reference cleanup)
- Follow-up TODOs: None
-->

# Brummer Constitution

## Core Principles

### I. Lock-Free Architecture
Prefer atomic operations and lock-free data structures over mutex-based synchronization. Use sync.Map for concurrent map operations and immutable state structures with atomic updates. All shared state modifications MUST use thread-safe patterns that avoid blocking operations.

### II. Event-Driven Design
All components MUST communicate through the EventBus using message-passing patterns. TUI components ONLY modify model state within BubbleTea Update() methods. Use bounded worker pools for event processing to prevent unbounded goroutine growth.

### III. Test-First Development
TDD is MANDATORY for all features: write failing tests, then implement. Red-Green-Refactor cycle is strictly enforced. Integration tests with TTY allocation are REQUIRED for TUI functionality. Race condition testing with make test-race is MANDATORY before any commits involving concurrency.

### IV. Process Management Excellence
Every process MUST be tracked with complete lifecycle management (pending → running → stopped/success/failed). All running processes MUST be cleaned up gracefully on exit. URL detection and proxy registration MUST be automatic for all discovered endpoints.

### V. MCP Integration Protocol
All MCP tools MUST follow the official Model Context Protocol specification (current version verified at modelcontextprotocol.io). Hub mode MUST support multi-instance coordination with session routing. JSON-RPC 2.0 compliance is REQUIRED for all MCP communications.

## Performance & Concurrency Standards

### Performance Requirements
- Search operations MUST complete in <5ms after indexing
- Index building MUST complete in <5s for typical projects (<10k files)
- Memory usage MUST stay <100MB for typical web projects
- API response times MUST be <500ms (p95)
- Page load times MUST be <2s

### Concurrency Safety
- Race condition detection is integrated in CI/CD pipeline
- Use atomic operations for shared state counters
- Event processing MUST use bounded channels
- TUI state modifications MUST be single-threaded via BubbleTea
- Process isolation MUST prevent shared state corruption

## Development Quality Gates

### Code Quality Standards
- All code MUST pass golangci-lint checks
- Code coverage MUST be maintained above 80%
- Documentation updates are REQUIRED for all public APIs
- Error messages MUST include context for troubleshooting
- Performance benchmarks are REQUIRED for new features

### Testing Requirements
- Unit tests for all business logic
- Integration tests for all user workflows
- TTY allocation tests for TUI components
- MCP protocol compliance tests
- Performance regression tests for critical paths

## Governance

This constitution supersedes all other development practices and guidelines. Amendments require:
1. Documentation of proposed changes with rationale
2. Team review and approval process
3. Migration plan for existing code
4. Version increment according to semantic versioning rules
5. Update of all dependent templates and documentation

All pull requests and code reviews MUST verify compliance with constitutional principles. Any complexity that deviates from these principles MUST be explicitly justified with documented alternatives that were considered and rejected.

**Version**: 1.0.0 | **Ratified**: 2025-10-05 | **Last Amended**: 2025-10-05