# VHS script for error detection demonstration

Output ../static/img/screenshots/error-detection-demo.gif

Set FontSize 14
Set Width 1200
Set Height 800
Set Theme "Dracula"

Hide
Type "cd /tmp && mkdir error-demo && cd error-demo"
Enter

# Create package.json with scripts that will produce errors
Type "cat > package.json << 'EOF'"
Enter
Type '{'
Enter
Type '  "name": "error-demo",'
Enter
Type '  "scripts": {'
Enter
Type '    "dev": "node server.js",'
Enter
Type '    "test": "jest",'
Enter
Type '    "build": "tsc",'
Enter
Type '    "lint": "eslint ."'
Enter
Type '  }'
Enter
Type '}'
Enter
Type "EOF"
Enter

# Create a server file with an error
Type "cat > server.js << 'EOF'"
Enter
Type "console.log('Server starting...');"
Enter
Type "setTimeout(() => {"
Enter
Type "  console.error('ERROR: Database connection failed!');"
Enter
Type "  throw new Error('Unable to connect to PostgreSQL');"
Enter
Type "}, 2000);"
Enter
Type "EOF"
Enter
Show

# Start Brummer
Type "brum"
Enter
Sleep 2s

# Run the server (will error)
Enter
Sleep 3s

# Switch to Errors view to show error detection
Tab
Tab
Tab
Sleep 1s
Screenshot ../static/img/screenshots/error-detection.png

# Show TypeScript error example
Type "q"
Enter
Sleep 500ms

Hide
Type "cat > app.ts << 'EOF'"
Enter
Type "interface User {"
Enter
Type "  name: string;"
Enter
Type "  age: number;"
Enter
Type "}"
Enter
Type ""
Enter
Type "const user: User = {"
Enter
Type "  name: 'John',"
Enter
Type "  // Missing required property 'age'"
Enter
Type "};"
Enter
Type "EOF"
Enter
Show

Type "brum"
Enter
Sleep 2s

# Run build (will show TypeScript error)
Type "jj"  # Navigate to build
Enter
Sleep 3s

# Show error in Errors tab
Tab
Tab
Tab
Sleep 1s
Screenshot ../static/img/screenshots/typescript-error.png

Sleep 1s