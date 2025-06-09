Implementation Plan for MCP Streamable HTTP Integration in Brummer

  Based on my analysis of the Brummer codebase, here's my comprehensive plan for integrating the MCP Streamable
  HTTP solution with advanced tools:

  Phase 1: MCP Streamable HTTP Integration

  1. Replace current MCP server (internal/mcp/server.go):
    - Integrate the provided Streamable HTTP server code
    - Preserve existing endpoints for backward compatibility
    - Add JSON-RPC 2.0 support with SSE streaming
    - Maintain integration with EventBus for real-time updates
  2. Create new file structure:
    - internal/mcp/streamable_server.go - Main server implementation
    - internal/mcp/tools.go - Tool definitions and handlers
    - internal/mcp/resources.go - Resource handlers
    - internal/mcp/prompts.go - Prompt handlers

  Phase 2: MCP Tools Implementation

  Script Management Tools:

  1. scripts/list - List all available npm scripts
    - Chain of thought: "When you need to see what scripts are available to run"
    - Returns scripts from package.json with descriptions
  2. scripts/run - Start a script
    - Chain of thought: "Whenever you want to run 'npm run dev', use this tool to start it through brummer"
    - Supports streaming output
    - Auto-detects package manager (npm/yarn/pnpm/bun)
  3. scripts/stop - Stop a running script
    - Chain of thought: "When you need to stop a running process like a dev server"
    - Graceful shutdown with SIGTERM/SIGINT
  4. scripts/status - Check script status
    - Chain of thought: "Before starting a script, check if it's already running"
    - Returns process status, PID, uptime
  5. logs/stream - Stream real-time logs
    - Streaming tool for live log monitoring
    - Supports filtering by level, process, pattern
  6. logs/search - Search historical logs
    - Chain of thought: "When debugging, search for specific errors or patterns in logs"
    - Regex support, time-based filtering

  Proxy & Telemetry Tools:

  7. proxy/requests - Get HTTP requests
    - Chain of thought: "To see what API calls your app is making"
    - Shows status codes, timing, authentication info
  8. telemetry/sessions - Get browser telemetry
    - Chain of thought: "To monitor browser performance and errors"
    - Shows page load times, JS errors, memory usage
  9. telemetry/events - Stream telemetry events
    - Streaming tool for real-time browser monitoring
    - Includes console logs, errors, user interactions

  Browser Automation Tools:

  10. browser/open - Open URL in browser with proxy
    - Chain of thought: "To test your web app with automatic proxy configuration"
    - Cross-platform support (Windows/Mac/Linux/WSL2)
    - Auto-configures reverse proxy
    - Returns proxy URL for monitoring
  11. browser/refresh - Refresh browser tab
    - Uses telemetry WebSocket to send refresh command
    - Chain of thought: "After making changes, refresh the browser to see updates"
  12. browser/navigate - Navigate to URL
    - Chain of thought: "To test different pages or routes in your app"
    - Maintains proxy connection

  JavaScript REPL Tool:

  13. repl/execute - Execute JavaScript in browser context
    - Chain of thought: "To debug or interact with your running web app"
    - Uses telemetry WebSocket for execution
    - Returns execution result or errors
    - Supports async/await

  Phase 3: Implementation Details

  1. Tool Descriptions with Examples:
  tools["scripts/run"] = Tool{
      Name: "scripts/run",
      Description: `Start a package.json script through Brummer.
      
      Chain of thought: Whenever you want to run "npm run dev" or any other script,
      use this tool instead of running it directly. This ensures proper process
      management and log capturing.
      
      Example usage:
      - To start development server: {"name": "dev"}
      - To run tests: {"name": "test"}
      - To build project: {"name": "build"}`,
      InputSchema: {...},
      Streaming: true,
      StreamingHandler: ...,
  }
  2. Browser Opening Logic:
  func openBrowser(url string) error {
      var cmd string
      var args []string

      switch runtime.GOOS {
      case "windows":
          cmd = "cmd"
          args = []string{"/c", "start", url}
      case "darwin":
          cmd = "open"
          args = []string{url}
      case "linux":
          // Check if running in WSL
          if isWSL() {
              cmd = "cmd.exe"
              args = []string{"/c", "start", url}
          } else {
              cmd = "xdg-open"
              args = []string{url}
          }
      }

      return exec.Command(cmd, args...).Start()
  }
  3. JavaScript REPL Integration:
    - Inject REPL handler into monitor.js
    - Use existing WebSocket connection for bidirectional communication
    - Execute code in page context using eval (with security considerations)
  4. Resource Definitions:
    - logs://recent - Recent log entries
    - telemetry://sessions - Active telemetry sessions
    - proxy://requests - HTTP request history
    - Support subscription for real-time updates
  5. Prompt Templates:
    - "Debug Error" - Analyzes error logs and suggests fixes
    - "Performance Analysis" - Reviews telemetry data
    - "API Troubleshooting" - Examines proxy requests

  Phase 4: Integration Points

  1. Update main.go:
    - Replace MCP server initialization with streamable version
    - Maintain backward compatibility flag
  2. Update TUI model:
    - Add MCP status indicator
    - Show active MCP connections
  3. Enhance telemetry:
    - Add REPL command support
    - Browser control commands
  4. Testing:
    - Create test scripts for each tool
    - Ensure cross-platform browser opening
    - Validate streaming performance

  This implementation will provide a powerful MCP integration that allows AI assistants and other tools to fully
   control and monitor the development environment through Brummer.


