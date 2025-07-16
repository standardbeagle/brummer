# Brummer: AI-Enhanced Development Environment Manager

**Brummer** is a comprehensive development environment manager designed specifically to enhance AI-assisted development workflows. It provides intelligent process monitoring, advanced debugging tools, and seamless integration with AI coding assistants like Claude.

## What is Brummer?

Brummer is a terminal-based development orchestrator that transforms how developers work with AI assistants. It automatically monitors your development processes, captures detailed logs and errors, provides browser automation, and exposes everything through a standardized MCP (Model Context Protocol) interface that AI assistants can use to help you debug, optimize, and build better software.

**Core Philosophy**: Bridge the gap between AI assistants and your development environment by providing rich, contextual information about your running processes, errors, network requests, and browser state.

## Key Capabilities

### 🔍 **Intelligent Error Detection & Analysis**
- **Automatic Error Clustering**: Groups related error messages (like multi-line stack traces) using time-based analysis
- **Contextual Error Information**: Captures process context, timing, and related log entries
- **Error Pattern Recognition**: Identifies common error patterns across different frameworks and languages
- **Structured Error Export**: Makes errors easily accessible to AI assistants for analysis and solutions

**Example Workflow:**

    Developer: "My React app is failing to start"
    AI (via Brummer): Uses logs_search to find startup errors, analyzes the stack trace, 
                      identifies missing dependencies, and suggests exact fix commands

### 📋 **Advanced Log Management & Search**
- **Real-time Log Streaming**: Monitor multiple processes simultaneously with filtering
- **Intelligent Log Search**: Regex-powered search across historical logs with context
- **Process-specific Filtering**: Focus on logs from specific services or processes
- **Time-based Analysis**: Find correlations between events across different processes

**Example Workflow:**

    Developer: "Why is my API responding slowly?"
    AI (via Brummer): Uses logs_search to find database timeout patterns, correlates with 
                      proxy_requests to identify slow endpoints, suggests optimization strategies

### 🌐 **Browser Automation & Debugging**
- **Live JavaScript REPL**: Execute code directly in your running application's browser context
- **Request Monitoring**: Capture and analyze all HTTP requests with detailed timing information
- **Browser State Inspection**: Access DOM, localStorage, global variables, and component state
- **Visual Testing**: Take screenshots and monitor visual changes during development

**Example Workflow:**

    Developer: "The login form isn't working properly"
    AI (via Brummer): Uses repl_execute to inspect form state, checks network requests 
                      via proxy_requests, identifies CORS issues, provides fix

### 🚀 **Multi-Project Coordination (Hub Mode)**
- **Instance Discovery**: Automatically detects and coordinates multiple project instances
- **Cross-Service Debugging**: Debug issues that span multiple microservices
- **Unified Tool Access**: Access tools from any project instance through a single interface
- **Distributed Development**: Perfect for microservice architectures and monorepos

**Example Workflow:**

    Developer: "My frontend can't connect to the backend"
    AI (via Brummer): Uses hub_proxy_requests to check API calls from frontend instance,
                      hub_logs_search to find backend errors, identifies port mismatch

## How Brummer Improves AI Development Workflows

### **1. Error Resolution Acceleration**
Traditional workflow:
- Error occurs → Developer copies error to AI → AI suggests generic solutions → Trial and error

**With Brummer:**
- Error occurs → AI automatically accesses full error context → AI provides specific, actionable solutions based on your exact environment

### **2. Real-time Debugging Assistance**
Traditional workflow:
- Issue occurs → Developer describes symptoms → AI asks clarifying questions → Slow back-and-forth

**With Brummer:**
- Issue occurs → AI directly inspects your application state, logs, and network traffic → Immediate, precise diagnosis

### **3. Contextual Code Suggestions**
Traditional workflow:
- AI suggests code → Developer tests manually → Multiple iterations to get it working

**With Brummer:**
- AI suggests code → AI tests directly in your browser via REPL → AI refines based on actual behavior → Working solution faster

### **4. Performance Optimization**
Traditional workflow:
- Performance issue → Developer manually gathers metrics → AI analyzes incomplete data

**With Brummer:**
- Performance issue → AI accesses real-time proxy data, log patterns, and browser metrics → Comprehensive optimization recommendations

## Practical Examples

### **Example 1: Database Connection Issues**

    # Developer notices app hanging
    Developer: "My app seems to be hanging on startup"

    # AI uses Brummer to investigate
    AI: logs_search({"query": "connect|database|timeout", "level": "error"})
    # Finds: "Connection timeout after 5000ms to database localhost:5432"

    AI: "I found a database connection timeout. Your app is trying to connect to 
        PostgreSQL on localhost:5432 but timing out. Is PostgreSQL running?"

    # AI provides specific solution
    AI: scripts_run({"name": "db:start"})  # Starts database if script exists

### **Example 2: API Integration Debugging**

    # API calls failing
    Developer: "My API calls are returning 401 errors"

    # AI investigates using proxy monitoring
    AI: proxy_requests({"status": "error"})
    # Finds requests missing Authorization header

    AI: repl_execute({"code": "localStorage.getItem('authToken')"})
    # Discovers token is null

    AI: "Your API calls are failing because the auth token isn't being stored. 
        The localStorage shows null for 'authToken'. Let me check your login flow..."

### **Example 3: Build Process Optimization**

    # Slow build times
    Developer: "My build is taking forever"

    # AI analyzes build logs and patterns
    AI: logs_search({"query": "webpack|build|compile", "since": "1h"})
    # Identifies large bundle sizes and unused imports

    AI: repl_execute({"code": "webpack.getStats().chunks.map(c => ({name: c.name, size: c.size}))"})
    # Gets actual bundle size data

    AI: "Your build is slow because webpack is processing large unused dependencies. 
        Based on your bundle analysis, consider lazy loading these modules..."

## Integration Examples

### **Claude Desktop Configuration**

    {
      "servers": {
        "brummer": {
          "command": "brum",
          "args": ["--no-tui", "--port", "7777"]
        }
      }
    }

### **Multi-Project Hub Setup**

    # Terminal 1: Frontend
    cd frontend/ && brum dev

    # Terminal 2: Backend  
    cd backend/ && brum -p 7778 start

    # Terminal 3: Hub coordinator
    brum --mcp  # Coordinates both instances

### **Continuous Integration**

    # GitHub Actions example
    - name: Start development environment
      run: brum --no-tui test &
      
    - name: Wait for services
      run: |
        # Use MCP tools to verify services are ready
        curl -X POST http://localhost:7777/mcp \
          -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"scripts_status"}}'

## Advanced Features

### **Time-based Error Clustering**
Brummer uses intelligent time-gap analysis to group related error messages:
- **Multi-line stack traces** are automatically assembled into single, coherent errors
- **Related warnings and errors** occurring within time windows are grouped
- **Context preservation** maintains the relationship between different error components

### **Automatic URL Detection & Proxy Management**
- **Smart URL discovery** from process logs (detects "Server running on http://localhost:3000")
- **Automatic proxy setup** for request monitoring without manual configuration
- **Shareable development URLs** for team collaboration and testing

### **Browser Integration**
- **Automatic browser launching** with proxy configuration for monitoring
- **Live JavaScript execution** for testing and debugging without leaving your AI assistant
- **Screenshot capabilities** for visual regression testing and bug reporting

### **Process Lifecycle Management**
- **Intelligent restart detection** prevents duplicate processes
- **Resource cleanup** automatically manages orphaned processes
- **Exit code analysis** provides detailed failure information

## Documentation & Resources

### **Full Documentation**
📖 **GitHub Repository**: [https://github.com/standardbeagle/brummer](https://github.com/standardbeagle/brummer)

### **Quick Start Guides**
- **Installation**: npm install -g @standardbeagle/brummer or go install github.com/standardbeagle/brummer/cmd/brum@latest
- **Basic Usage**: cd your-project && brum dev
- **MCP Integration**: Configure your AI assistant to connect to http://localhost:7777/mcp
- **Hub Mode**: Use brum --mcp for coordinating multiple project instances

### **Community & Support**
- **Issues & Feature Requests**: GitHub Issues
- **Documentation**: README.md and docs/ directory
- **Examples**: examples/ directory with real-world usage patterns

---

**Brummer transforms AI-assisted development from reactive problem-solving to proactive, context-aware development assistance. By providing AI assistants with direct access to your development environment, Brummer enables faster debugging, more accurate solutions, and enhanced development productivity.**

*Generated by Brummer about tool - bridging AI assistants and development environments*