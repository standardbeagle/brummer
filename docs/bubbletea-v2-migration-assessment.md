# Bubble Tea v2 Migration Assessment

## Current Status
- **Current Version**: Bubble Tea v1 (v0.25.0)
- **Target Version**: Bubble Tea v2 (beta.3)
- **Reason**: Enable use of `bubbleterm` for better terminal emulation

## Scope of Changes
- **Total TUI files**: 152 actively compiled files
- **Files with Bubble Tea usage**: 96 active files (63% of TUI codebase)
- **Total tea.* references**: 702 occurrences
- **Complexity**: Large codebase with extensive Bubble Tea integration across:
  - Main package: 46 files
  - Components: 30 files
  - Commands: 27 files
  - State: 19 files
  - Services: 17 files
  - Messages: 11 files
  - Handlers: 2 files

## Key Breaking Changes in v2

### 1. Mouse Event Handling (Major)
**v1 Pattern:**
```go
case tea.MouseMsg:
    // Handle all mouse events
```

**v2 Pattern:**
```go
case tea.MouseClickMsg:
    // Handle clicks
case tea.MouseReleaseMsg:
    // Handle releases
case tea.MouseWheelMsg:
    // Handle wheel
case tea.MouseMotionMsg:
    // Handle motion
```

**Impact**: Low - We don't use mouse events extensively

### 2. Paste Events (Medium)
**v1 Pattern:**
```go
case tea.KeyMsg:
    if msg.Paste {
        // Handle paste
    }
```

**v2 Pattern:**
```go
case tea.PasteMsg:
    // Handle paste
```

**Impact**: Medium - Need to check paste handling in input components

### 3. Key Event String Representation (Low)
**v1**: `key.String()` returns keystroke representation
**v2**: `key.String()` returns textual value, use `key.Keystroke()` for keystroke

**Impact**: Low - Mostly internal changes

### 4. Import Path Changes (Trivial)
**v1**: `github.com/charmbracelet/bubbletea`
**v2**: `github.com/charmbracelet/bubbletea/v2`

**Impact**: Trivial - Automated with find/replace

## Migration Effort Estimate

### Phase 1: Preparation (1-2 hours)
- [ ] Review all mouse event usage
- [ ] Identify paste event handling
- [ ] Document key string usage patterns

### Phase 2: Update Imports (30 minutes)
- [ ] Update go.mod to use v2
- [ ] Find/replace import paths
- [ ] Update vendor directory if used

### Phase 3: Fix Breaking Changes (3-4 hours)
- [ ] Update mouse event handling (if any)
- [ ] Update paste event handling
- [ ] Fix key string representation issues
- [ ] Address any other breaking changes

### Phase 4: Testing (2-3 hours)
- [ ] Run all unit tests
- [ ] Run integration tests
- [ ] Manual testing of all views
- [ ] Test keyboard navigation
- [ ] Test input handling

### Phase 5: bubbleterm Integration (2-3 hours)
- [ ] Integrate bubbleterm for AI Coder
- [ ] Test with Claude Code
- [ ] Verify no regressions

**Total Estimate**: 8-13 hours

## Risks

### High Risk
- **Breaking changes** in undocumented areas
- **Test coverage gaps** revealing issues late
- **Dependency conflicts** with other packages

### Medium Risk
- **Performance regressions** in v2 beta
- **Bugs in v2 beta** that block progress
- **Community support** for beta version issues

### Low Risk
- **Import path changes** are straightforward
- **Most breaking changes** are well-documented

## Recommendation

### Option A: Migrate to v2 (Recommended for long-term)
**Pros:**
- Enable bubbleterm usage
- Better terminal emulation
- Future-proof codebase
- Better mouse/keyboard handling

**Cons:**
- 8-13 hours of work
- Testing overhead
- Potential beta bugs
- Risk of regressions

### Option B: Stay on v1 with Raw PTY (Current Solution)
**Pros:**
- Already implemented and working
- No migration risk
- Proven stability
- Simple and effective

**Cons:**
- Can't use bubbleterm
- Miss out on v2 improvements
- Eventually need to migrate anyway

## Decision Factors

### Migrate if:
1. bubbleterm provides significant value over raw PTY
2. We have 1-2 weeks for thorough testing
3. Team comfortable with beta software
4. Want better mouse/keyboard APIs

### Stay on v1 if:
1. Raw PTY solution works well enough
2. Tight timeline / need stability
3. Want to wait for v2 stable release
4. Risk-averse approach preferred

## Current Recommendation

**Stay on v1 with raw PTY for now**, because:
1. The raw PTY solution is working and solves the duplicate issue
2. v2 is still in beta (not stable)
3. Migration effort (8-13 hours) is significant
4. Can revisit when v2 is stable

**Future Migration**: Plan to migrate to v2 when:
- v2 reaches stable release
- We have a dedicated sprint for the work
- bubbleterm shows clear advantages
- Community adoption is strong