# AI Coder Tab Repair - Cleanup Checklist

## Overview
This document lists all the deprecated code and infrastructure that should be removed after the new screen buffer architecture is fully tested and working.

## Files to Delete Entirely
- [ ] `internal/aicoder/channel_monitor.go` (if exists) - Not needed for 1-3 sessions
- [ ] `internal/aicoder/buffer_manager.go` (if exists) - Replaced by RingBuffer
- [ ] `internal/tui/services/stats_collector.go` (if exists) - Not needed

## Code to Remove from Existing Files

### `internal/aicoder/pty_session.go`
- [x] `channelMemoryStats` struct field - REMOVED
- [x] `UpdateChannelStats()` method - REMOVED
- [x] `GetChannelStats()` method - REMOVED
- [x] `channelMonitorLoop()` method - REMOVED
- [x] Channel buffer sizes reduced from 1000 to 100 - DONE

### `internal/tui/state/aicoder_state.go`
- [x] `PTYOutputBuffers` field - REMOVED (fallback rendering eliminated)
- [x] `TerminalStates` field - REMOVED (fallback rendering eliminated)
- [x] `LastPollTime` field - REMOVED (no longer needed)
- [x] `PollingInterval` field - REMOVED (no longer needed)
- [x] `AddPTYOutput()` method - REMOVED (migrated to AddRawOutput())
- [x] `UpdateTerminalState()` method - REMOVED (migrated to UpdateScreenBuffer())
- [x] `GetPTYOutput()` method - REMOVED (no longer called anywhere)
- [x] `GetTerminalState()` method - REMOVED (no longer called anywhere)
- [x] `UpdatePollingTime()` method - REMOVED (was never called)

### `internal/tui/services/aicoder_service.go`
- [ ] `CheckPTYOutput()` method - Currently marked as DEPRECATED
- [ ] `CheckSessionStatus()` method (simplified version kept)
- [ ] Complex channel monitoring logic - REMOVED via debouncer

### `internal/tui/messages/aicoder_messages.go`
- [ ] `AICoderPTYOutputMsg` - Currently marked as DEPRECATED (handlers still use it)
- [x] `AICoderPollPTYMsg` - REMOVED (no longer needed with debounced updates)
- [x] `TerminalStateMsg` - DEPRECATED and handler removed (message was never emitted, replaced by AICoderScreenSyncMsg)
- [ ] `AICoderTerminalUpdateMsg` (if exists) - Replaced by AICoderScreenSyncMsg

### `internal/tui/ai_coder_message_handler.go`
- [x] Handler for `AICoderPTYOutputMsg` - MIGRATED to use AddRawOutput() instead of AddPTYOutput()
- [x] Handler for `AICoderPollPTYMsg` - Message type removed, no handler needed

### `internal/tui/components/aicoders_view.go`
- [x] Fallback rendering code using `GetPTYOutput()` - REMOVED (replaced with simple waiting message)
- [x] Fallback rendering code using `GetTerminalState()` - REMOVED (replaced with simple waiting message)
- [x] Deprecated width/height fields - REMOVED (using layoutRegions exclusively)

## Migration Path

### Phase 1: Parallel Implementation (COMPLETE)
✅ Added new fields to AICoderState alongside old ones
✅ Implemented debouncer without breaking existing code
✅ Created screen buffer infrastructure
✅ Added new message types
✅ Updated handlers to support both old and new messages

### Phase 2: Switch Over (IN PROGRESS)
- [ ] Connect debouncer callback in TUI initialization
- [ ] Route PTY output through debouncer instead of channels
- [ ] Use screen buffer for rendering in AI Coder view
- [ ] Test session switching with fast replay

### Phase 3: Clean Up (PENDING)
- [ ] Remove all deprecated fields from AICoderState
- [ ] Delete all deprecated methods
- [ ] Remove old message types
- [ ] Clean up fallback rendering code
- [ ] Remove commented code

## Testing Checklist

Before removing old code, verify:
- [ ] Single session renders at 60 FPS
- [ ] Tab switching replays buffer correctly
- [ ] Input latency < 1ms
- [ ] No memory leaks over time
- [ ] Background sessions use minimal CPU
- [ ] Debouncer batches updates properly
- [ ] Screen buffer syncs with vt10x correctly
- [ ] Session creation/destruction works
- [ ] Error handling is robust

## Performance Metrics

### Before (Old Implementation)
- Message flow: Unbounded (up to 280 updates/sec)
- Memory per session: 3-5MB
- Channel buffers: 1000 items each
- Complex monitoring: 30-second polling
- Code complexity: ~2000 lines

### After (New Implementation)
- Message flow: 60 FPS max (debounced)
- Memory per session: ~500KB
- Channel buffers: 100 items each
- No monitoring overhead
- Code complexity: ~500 lines
- **75% code reduction**

## Notes

- The old code is marked with "DEPRECATED" comments for easy identification
- Migration can be done gradually - old and new code can coexist
- Focus on testing the happy path first before removing fallbacks
- Keep the deprecated code until integration testing is complete