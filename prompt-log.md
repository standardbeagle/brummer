2025-09-21
Prompt: "do the two channels make sense, search everything for partially implemented duplicate components that could be getting things confused" followed by "yes go ahead".
Actions: Searched codebase for AIPTY* message handling and PTY channel consumers; identified duplicate/legacy AI components and managers.
Edits: (1) internal/tui/components/ai_pty_component.go — ignore PTYEventOutput in EventChan subscriber to avoid duplicate output signaling; rely on OutputChan for UI rendering. (2) internal/tui/components/ai_view_component.go — add DEPRECATED header comment to prevent accidental use.
Verification: make build; go test -run AIPTY ./internal/tui/components; make test-fast — all passed.
Rationale: Reduce routing/subscription confusion by using a single authoritative output stream and clearly deprecate old placeholder component.

