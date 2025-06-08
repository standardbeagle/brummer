---
sidebar_position: 3
---

# Error Detection

Brummer's intelligent error detection helps you quickly identify and fix issues in your development workflow.

## How It Works

Brummer continuously monitors process output for error patterns and provides real-time notifications and highlighting.

### Detection Pipeline

```
Process Output → Pattern Matching → Classification → Notification → UI Update
```

## Built-in Patterns

### JavaScript/TypeScript Errors

Brummer detects common JavaScript and TypeScript errors:

```javascript
// Syntax errors
SyntaxError: Unexpected token '}'
  at Module._compile (internal/modules/cjs/loader.js:723:23)

// Runtime errors
TypeError: Cannot read property 'map' of undefined
  at App.render (/src/App.js:15:20)

// Reference errors
ReferenceError: myVariable is not defined
  at processData (index.js:42:5)
```

### Build Errors

Framework-specific build errors are automatically detected:

#### React
```
Failed to compile.

./src/App.js
Module not found: Can't resolve './components/Header'
```

#### Vue
```
ERROR  Failed to compile with 1 error

error in ./src/App.vue

Module Error (from ./node_modules/vue-loader/lib/index.js):
```

#### Next.js
```
Error: Page "/api/users/[id]" is missing exported function "default"
```

### Test Failures

Test framework failures are highlighted:

#### Jest
```
FAIL  src/utils.test.js
  ● utility functions › should format date correctly

    Expected: "2024-01-15"
    Received: "15-01-2024"
```

#### Mocha
```
  1) User API
       should create a new user:
     AssertionError: expected 201 to equal 200
```

### Linting Errors

ESLint and other linter errors:

```
/src/components/Button.js
  12:5  error  'PropTypes' is not defined  no-undef
  23:9  error  Missing semicolon           semi
```

## Error Levels

Brummer classifies errors into different severity levels:

### 🔴 Critical Errors
- Build failures
- Syntax errors
- Unhandled exceptions
- Process crashes

### 🟡 Warnings
- Deprecation notices
- Linting warnings
- Performance warnings
- Security advisories

### 🔵 Info
- Debug messages
- Build progress
- Server startup messages

## Smart Features

### 1. Error Grouping

Similar errors are grouped together:

```
┌─ Errors (3 similar) ────────────────────────┐
│ TypeError: Cannot read property 'x' of null │
│   at calculatePosition (layout.js:45)       │
│   at renderComponent (render.js:23)         │
│   at updateUI (app.js:156)                  │
│                                              │
│ First occurrence: 10:23:45                   │
│ Last occurrence: 10:24:12                    │
│ Count: 47                                    │
└──────────────────────────────────────────────┘
```

### 2. Stack Trace Parsing

Stack traces are parsed and made interactive:

```
Error: Connection timeout
    at Database.connect (/src/db/index.js:23:11)  [Click to open]
    at startServer (/src/server.js:45:20)         [Click to open]
    at main (/src/index.js:12:5)                  [Click to open]
```

### 3. Error Context

Brummer provides context around errors:

```
┌─ Error Context ─────────────────────────────┐
│ File: /src/api/users.js                     │
│ Line: 67                                    │
│ Column: 15                                  │
│                                             │
│ 65:   const user = await User.findById(id); │
│ 66:   if (!user) {                          │
│ 67:     throw new Error('User not found'); │
│       ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^  │
│ 68:   }                                     │
│ 69:   return user;                          │
└─────────────────────────────────────────────┘
```

### 4. Suggested Fixes

Common errors include suggested fixes:

```
Error: Module not found: 'express'

Suggested fix:
  npm install express
  # or
  yarn add express
```

## Custom Error Patterns

### Configuration File

Add custom patterns in `.brummer.yaml`:

```yaml
error_detection:
  patterns:
    - pattern: "CUSTOM_ERROR:(.*)"
      level: error
      group: "Custom Errors"
    
    - pattern: "DEPRECATED:(.*)"
      level: warning
      group: "Deprecations"
    
    - pattern: "TODO:(.*)"
      level: info
      group: "TODOs"
```

### Regular Expressions

Use regex for complex patterns:

```yaml
error_detection:
  patterns:
    - pattern: "\\[ERROR\\]\\s+\\[(\\d{4}-\\d{2}-\\d{2})\\]\\s+(.*)"
      level: error
      capture_groups:
        - name: date
        - name: message
```

### Pattern Priority

Higher priority patterns are matched first:

```yaml
error_detection:
  patterns:
    - pattern: "CRITICAL:.*"
      level: critical
      priority: 100
    
    - pattern: "ERROR:.*"
      level: error
      priority: 50
```

## Error Actions

### Quick Actions

Press `a` on an error for quick actions:

1. **Copy Error** - Copy full error to clipboard
2. **Open File** - Open error location in editor
3. **Search Similar** - Find similar errors
4. **Ignore Pattern** - Add to ignore list
5. **Report Issue** - Create GitHub issue

### Bulk Actions

Select multiple errors with `Space` then:

- **Mark as Resolved**
- **Export to File**
- **Create Issue**
- **Add to Ignore**

## Filtering and Search

### Filter by Level

```
Ctrl+F → Level: error
```

### Filter by Pattern

```
Ctrl+F → Pattern: "Cannot read property"
```

### Filter by Time

```
Ctrl+F → After: 10:30:00
```

### Complex Filters

```
level:error AND file:*.test.js AND after:"5 minutes ago"
```

## Error Statistics

View error statistics with `Ctrl+S`:

```
┌─ Error Statistics ──────────────────────────┐
│ Total Errors: 127                           │
│ Unique Errors: 12                           │
│ Error Rate: 2.1/min                         │
│                                             │
│ By Level:                                   │
│   Critical: 3                               │
│   Error: 45                                 │
│   Warning: 79                               │
│                                             │
│ Top Errors:                                 │
│   1. Cannot read property (34)              │
│   2. Module not found (23)                  │
│   3. Syntax error (15)                      │
└─────────────────────────────────────────────┘
```

## Integration with Tools

### VSCode Integration

Click on file paths to open in VSCode:

```bash
# Configure editor
brummer config set editor "code --goto"
```

### MCP Integration

External tools can query errors:

```javascript
// Get recent errors
const errors = await mcp.call('brummer.getErrors', {
  level: 'error',
  limit: 10
});
```

## Performance Considerations

### Pattern Matching Performance

- Patterns are compiled once at startup
- Most specific patterns first
- Use anchors in regex (`^` and `$`)

### Memory Usage

- Circular buffer for error storage
- Configurable retention period
- Automatic cleanup of old errors

```yaml
error_detection:
  max_errors: 1000
  retention_minutes: 60
```

## Troubleshooting

### Errors Not Detected

1. Check pattern configuration
2. Verify error format matches pattern
3. Check if pattern is too specific
4. Enable debug mode to see matching

### Too Many False Positives

1. Make patterns more specific
2. Add ignore patterns
3. Adjust detection sensitivity
4. Use negative lookahead in regex

### Performance Issues

1. Reduce number of patterns
2. Optimize regex patterns
3. Decrease retention period
4. Disable unused detectors

## Best Practices

1. **Start with built-in patterns** and add custom ones as needed
2. **Use specific patterns** to avoid false positives
3. **Group related errors** for better organization
4. **Set up ignore patterns** for known non-issues
5. **Monitor error rates** to catch regression
6. **Export critical errors** for post-mortem analysis
7. **Integrate with your workflow** using MCP