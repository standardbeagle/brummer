# Phase 0: Research - Code Cleanup and Refactoring

## Static Analysis Tools for Go
**Decision**: Use existing Go analysis tools and build custom utilities where needed
**Rationale**: Go provides excellent built-in static analysis capabilities through `go build`, `go vet`, and third-party tools
**Alternatives considered**:
- External commercial tools (overkill for this scope)
- Custom parser from scratch (unnecessary complexity)

## Key Event Handling in BubbleTea
**Decision**: Use BubbleTea's key event system with proper event ordering and priority handling
**Rationale**: BubbleTea provides built-in key event handling that can be extended for global handlers
**Alternatives considered**:
- Direct terminal escape sequence handling (too low-level)
- External keyboard event libraries (adds dependencies)

## AI Terminal Integration
**Decision**: Implement terminal emulator within TUI using BubbleTea and connect to browser debug tools via reverse proxy
**Rationale**: Leverages existing architecture and maintains TUI consistency
**Alternatives considered**:
- External terminal emulator integration (complex IPC)
- Web-based AI interface (breaks TUI paradigm)

## Performance Profiling Approach
**Decision**: Use Go's pprof for CPU/memory profiling and custom benchmarks for key event handling
**Rationale**: Go's profiling tools are comprehensive and well-integrated
**Alternatives considered**:
- External APM tools (overkill for desktop app)
- Custom timing instrumentation (redundant with pprof)

## Code Cleanup Strategy
**Decision**: Implement multi-stage cleanup: unused imports, unreachable functions, and dead code elimination
**Rationale**: Gradual approach minimizes risk and allows validation at each stage
**Alternatives considered**:
- Aggressive single-pass cleanup (too risky)
- Manual identification only (time-consuming)