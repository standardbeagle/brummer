# Test: Running Scripts Filtering

This document describes the feature that filters out already running scripts from the `/run` command autocomplete.

## Feature Description

When using the `/run` command in the command palette (accessed via `/` key), the autocomplete suggestions will now exclude any scripts that are already running. This prevents users from accidentally trying to start a script that's already active.

## Implementation Details

1. **Updated `CommandAutocomplete` struct** to include a reference to the process manager
2. **Added new constructor** `NewCommandAutocompleteWithProcessManager` for creating autocomplete with process filtering
3. **Modified `getSuggestionsForCurrentPosition` method** to filter out running scripts when showing `/run` suggestions
4. **Updated validation** in `ValidateInput` to check if a script is already running and show an appropriate error message
5. **Applied same filtering** to the script selector view that appears when starting brum without arguments

## How It Works

1. When the user types `/run` in the command palette, the system:
   - Gets all available scripts from package.json
   - Checks which processes are currently running via the process manager
   - Filters out any scripts that have a running process with `StatusRunning`
   - Shows only the scripts that can be started

2. If a user tries to run a script that's already running (e.g., by typing the full command), they'll see an error: "Script 'scriptname' is already running"

## Testing

To test this feature:

1. Start brum in a project with multiple scripts
2. Run one or more scripts (e.g., start the 'dev' script)
3. Open the command palette with `/`
4. Type `/run` and observe that running scripts don't appear in the autocomplete
5. Try to manually type `/run dev` (if dev is running) and press Enter - you should see an error message

## Code Changes

- `/internal/tui/command_autocomplete.go`: Added process manager support and filtering logic
- `/internal/tui/script_selector.go`: Added process manager support for initial script selection
- `/internal/tui/model.go`: Updated to use new constructors with process manager