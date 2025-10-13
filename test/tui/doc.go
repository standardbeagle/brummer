// Package tui provides integration testing harness for Brummer's terminal interface.
// It enables automated testing of TUI functionality without requiring TTY allocation
// by using PTY (pseudo-terminal) layer for realistic terminal simulation.
//
// The testing harness follows Test-First Development principles and integrates
// with existing Brummer architecture including InputService and NavigationService.
//
// Key components:
//   - harness: Core testing framework components
//   - pty: Pseudo-terminal layer for terminal emulation
//   - integration: Integration tests for real TUI code paths
//   - fixtures: Test data and mock configurations
//
// Usage:
//   th := harness.NewTestHarness()
//   config := harness.DefaultTestConfig()
//   th.Initialize(config)
//   defer th.Stop()
//
//   result := th.RunTest(testCase)
//   if result.Status != harness.TestStatusPass {
//       t.Errorf("Test failed: %s", result.Error)
//   }
package tui