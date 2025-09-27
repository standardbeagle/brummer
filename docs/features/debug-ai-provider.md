# Debug AI Provider ("/ai test")

Purpose: Provide a simple, deterministic streaming output source to validate the AI PTY component, routing, and tab persistence without relying on external AI CLIs.

How it works:
- Command: /ai test
- Behavior: Starts a PTY session that runs a bash loop printing `BRUMMER_TEST_COUNTER N` every 0.5 seconds
- Use cases:
  - Verify streaming output appears immediately
  - Confirm tab switches don’t blank the viewport
  - Check resubscription and scroll behavior

Notes:
- Implemented in `internal/tui/components/ai_pty_component.go` under `provider == "test"`
- Suggestions: The command palette includes `test` as a provider option
- Rendering: Still uses vt10x terminal emulator; ANSI is preserved like real providers

