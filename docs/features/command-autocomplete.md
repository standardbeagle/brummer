# Command Autocomplete Feature

## Overview

Brummer provides comprehensive command autocomplete for all slash commands in the command palette (triggered by pressing `/`). The autocomplete system offers:

- **Multi-level completion**: Command names and their arguments
- **Context-aware suggestions**: Different suggestions based on the command
- **Smart filtering**: Only shows relevant options (e.g., non-running scripts for `/run`)
- **Visual feedback**: Dropdown UI with arrow navigation
- **Tab completion**: Press Tab to apply the selected suggestion
- **Validation**: Real-time validation with helpful error messages

## How to Use

### Opening the Command Palette

Press `/` from any view (except when typing in an input field) to open the command palette.

### Navigating Suggestions

- **↑/↓ Arrow Keys**: Navigate through suggestions
- **Tab**: Apply the selected suggestion
- **Enter**: Execute the command
- **Esc**: Close the command palette

### Autocomplete Flow

1. Type `/` to open command palette
2. Start typing a command (e.g., `/doc`)
3. Autocomplete shows matching commands: `docker-up`, `docker-down`, `docker-build`
4. Use arrow keys to select or continue typing
5. Press Tab to complete the command
6. Add a space to see argument suggestions
7. Tab through arguments or type your own
8. Press Enter to execute

## Command Sources

Brummer provides commands from two sources:

### 1. Command Palette (Slash Commands)

Press `/` to open the command palette with autocomplete. These are built-in commands you can type:
- Process management: `/run`, `/restart`, `/stop`
- Log management: `/clear`, `/show`, `/hide`
- Docker: `/docker-up`, `/docker-down`, `/docker-build`
- Compose: `/compose-up`, `/compose-down`
- And more...

### 2. Run Dialog (Auto-Detected Commands)

The Run dialog (accessed via Ctrl+R or typing `/run`) shows commands auto-detected from your project:

**Automatically Detected**:
- **npm/yarn scripts** from `package.json`
- **Docker commands** when `Dockerfile` exists
  - `docker build .`
  - `docker build -t app .`
  - `docker run app`
  - `docker run -p 3000:3000 app`
- **Docker Compose commands** when `docker-compose.yml` exists
  - `docker-compose up`
  - `docker-compose up -d`
  - `docker-compose build`
  - `docker-compose down`
  - `docker-compose logs -f`
- **Go commands** when `go.mod` exists
- **Rust commands** when `Cargo.toml` exists
- **Python commands** when `requirements.txt` or `setup.py` exists
- **Wails commands** when `wails.json` exists
- **Electron commands** when electron is detected
- And more...

**Key Difference**:
- **Slash commands** (`/docker-up`) are generic - you specify container names manually
- **Auto-detected commands** (`docker-compose up`) use your actual project files automatically

## Supported Commands

### Process Management

#### `/run <script-name>`
- **Autocomplete**: Lists all available scripts from package.json
- **Filter**: Excludes scripts that are already running
- **Example**: `/run dev`

#### `/restart [process-name]`
- **Autocomplete**: Lists running processes + "all" option
- **Default**: Restarts all processes if no argument provided
- **Example**: `/restart dev` or `/restart all`

#### `/stop [process-name]`
- **Autocomplete**: Lists running processes + "all" option
- **Default**: Stops all processes if no argument provided
- **Example**: `/stop api` or `/stop all`

### Log Management

#### `/clear [target]`
- **Autocomplete**: `all`, `logs`, `errors`, `web`, or specific script names
- **Default**: Clears all logs if no argument provided
- **Example**: `/clear logs` or `/clear dev`

#### `/show <pattern>`
- **Autocomplete**: Common log patterns
  - `error`, `warn`, `info`, `debug`
  - Regex patterns: `^\\[`, `\\]$`, `|`
- **Example**: `/show error`

#### `/hide <pattern>`
- **Autocomplete**: Same patterns as `/show`
- **Example**: `/hide debug`

### Proxy Management

#### `/proxy <subcommand|url>`
- **Autocomplete**: `list` or common URL patterns
  - `http://localhost:3000`
  - `http://localhost:8080`
  - `https://example.com`
- **Example**: `/proxy list` or `/proxy http://localhost:3000`

#### `/toggle-proxy`
- **No arguments**: Toggles proxy on/off
- **Example**: `/toggle-proxy`

### Docker Commands

Brummer provides **two ways** to work with Docker:

1. **Slash Commands** (manual Docker commands):
   - `/docker-up`, `/docker-down`, `/docker-build`
   - Available via command palette autocomplete
   - Execute arbitrary Docker commands

2. **Auto-Detected Commands** (appears in Run dialog):
   - Automatically detected when `Dockerfile` exists in project
   - Commands like "docker build .", "docker run app"
   - Available in Run dialog (Ctrl+R or /run)

#### `/docker-up [container-name]`
- **Autocomplete**: `all` or container names
- **Default**: Starts all containers if no argument provided
- **Example**: `/docker-up web` or `/docker-up`
- **Note**: This is a manual command. For project-specific Docker commands, use the Run dialog.

#### `/docker-down [container-name]`
- **Autocomplete**: `all` or container names
- **Default**: Stops all containers if no argument provided
- **Example**: `/docker-down api` or `/docker-down all`

#### `/docker-build [tag]`
- **Autocomplete**: Common tag patterns
  - `latest`, `dev`, `prod`, `v1.0.0`
- **Default**: Uses `latest` tag if no argument provided
- **Example**: `/docker-build dev` or `/docker-build`

### Docker Compose Commands

Brummer provides **two ways** to work with Docker Compose:

1. **Slash Commands** (manual compose commands):
   - `/compose-up`, `/compose-down`
   - Available via command palette autocomplete
   - Execute arbitrary compose commands

2. **Auto-Detected Commands** (appears in Run dialog):
   - Automatically detected when `docker-compose.yml` or `compose.yml` exists
   - Commands like "docker-compose up", "docker-compose down"
   - Available in Run dialog
   - Uses your actual compose file automatically

#### `/compose-up [service-name]`
- **Autocomplete**: Common service names
  - `all`, `web`, `api`, `db`, `redis`, `worker`
- **Default**: Starts all services if no argument provided
- **Example**: `/compose-up web` or `/compose-up`
- **Note**: For project-specific compose files, use auto-detected commands from Run dialog.

#### `/compose-down [service-name]`
- **Autocomplete**: Same service names as `/compose-up`
- **Default**: Stops all services if no argument provided
- **Example**: `/compose-down db` or `/compose-down`

### AI Coders

#### `/ai <provider>`
- **Autocomplete**: Available AI providers
  - `claude`, `claude-3-7`, `claude-o3`
- **Example**: `/ai claude`

### Other Commands

#### `/help`
- **No arguments**: Shows help information
- **Example**: `/help`

#### `/exit` or `/quit`
- **No arguments**: Exits the application
- **Example**: `/exit`

## Smart Features

### Context-Aware Filtering

The autocomplete system intelligently filters suggestions based on the current state:

- **`/run`**: Only shows scripts that are NOT currently running
- **`/restart`** and **`/stop`**: Only shows processes that ARE running
- **`/clear`**: Shows both global targets (all, logs, errors, web) and specific script names

### Validation

Real-time validation provides helpful error messages:

- **Missing required arguments**: "Please specify a script name (e.g., /run dev)"
- **Invalid script names**: "Script 'foo' not found. Available: dev, test, build..."
- **Already running scripts**: "Script 'dev' is already running"
- **Invalid targets**: "Invalid clear target 'foo'. Use: all, logs, errors, or a script name"

### Partial Command Detection

If you type a partial command, autocomplete will suggest the full command:

- Type `/doc` → Suggests: `/docker-up`, `/docker-down`, `/docker-build`
- Type `/ru` → Suggests: `/run`
- Type `/comp` → Suggests: `/compose-up`, `/compose-down`

## Architecture

### Multi-Level Completion

The autocomplete system supports completion at multiple levels:

1. **Command Level**: Suggests slash commands (`/run`, `/docker-up`, etc.)
2. **Argument Level**: Suggests command-specific arguments based on context

### Prefix Matching

Current implementation uses prefix matching (case-insensitive):
- Type `doc` → Matches `docker-up`, `docker-down`, `docker-build`
- Type `comp` → Matches `compose-up`, `compose-down`

### Future Enhancements

Potential improvements for the autocomplete system:

1. **Fuzzy Matching**: Match commands with typos or out-of-order characters
2. **Dynamic Container Detection**: Parse `docker ps` output for actual container names
3. **Docker Compose Parsing**: Read `docker-compose.yml` to suggest actual service names
4. **Command History**: Suggest recently used commands
5. **Intelligent Ranking**: Show frequently used commands first
6. **Alias Support**: Create custom aliases for common commands

## Implementation Details

### Code Location

- **Autocomplete Logic**: `internal/tui/command_autocomplete.go`
- **Slash Commands**: `internal/tui/commands/slash_commands.go`
- **Command Validation**: Integrated into the `CommandAutocomplete` struct

### Key Methods

- `updateSuggestions()`: Parses input and generates context-aware suggestions
- `getSuggestionsForCurrentPosition()`: Returns suggestions based on command and argument position
- `filterSuggestions()`: Filters options by prefix matching (case-insensitive)
- `ValidateInput()`: Validates complete commands before execution
- `applySelectedSuggestion()`: Applies the selected suggestion via Tab key

### Testing

To test autocomplete functionality:

1. Start Brummer: `./brum`
2. Press `/` to open command palette
3. Try typing partial commands and observe suggestions
4. Test Tab completion at each level
5. Test validation by entering invalid commands

## User Experience Guidelines

### Best Practices

1. **Use Tab completion**: Don't type entire commands when autocomplete can help
2. **Check suggestions**: Autocomplete shows what options are valid
3. **Read validation messages**: Error messages explain what's wrong and what's available
4. **Navigate with arrows**: Faster than typing for selecting from suggestions

### Common Workflows

**Starting a development server:**
```
/ → run → Tab → dev → Tab → Enter
```

**Restarting all processes:**
```
/ → restart → Tab → all → Tab → Enter
```

**Building a Docker image:**
```
/ → docker-build → Tab → dev → Tab → Enter
```

**Starting Docker Compose services:**
```
/ → compose-up → Tab → web → Tab → Enter
```

## Troubleshooting

### Autocomplete not showing suggestions

- Ensure you're in a context where `/` opens the command palette (not in an input field)
- Check that you've typed at least one character after `/`
- Verify the command exists in the available commands list

### Tab not completing

- Ensure autocomplete dropdown is visible
- Check that a suggestion is selected (arrow indicator `▶`)
- Verify you're not at the end of a complete command

### Command validation failing

- Read the error message carefully - it explains what's wrong
- Check available options listed in the error message
- Ensure script names match those in package.json
- Verify processes are running before trying to restart/stop them
