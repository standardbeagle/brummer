# VHS script for React development screenshots

Output ../static/img/screenshots/react-demo.gif

Set FontSize 14
Set Width 1200
Set Height 800
Set Theme "Dracula"

Hide
Type "cd /tmp && mkdir react-app-demo && cd react-app-demo"
Enter

# Create React project package.json
Type "cat > package.json << 'EOF'"
Enter
Type '{'
Enter
Type '  "name": "my-react-app",'
Enter
Type '  "scripts": {'
Enter
Type '    "dev": "vite",'
Enter
Type '    "build": "vite build",'
Enter
Type '    "preview": "vite preview",'
Enter
Type '    "test": "vitest",'
Enter
Type '    "test:ui": "vitest --ui",'
Enter
Type '    "lint": "eslint src --ext ts,tsx --report-unused-disable-directives",'
Enter
Type '    "type-check": "tsc --noEmit",'
Enter
Type '    "format": "prettier --write src/**/*.{ts,tsx,css}",'
Enter
Type '    "api": "json-server --watch db.json --port 3001"'
Enter
Type '  }'
Enter
Type '}'
Enter
Type "EOF"
Enter
Show

# React Scripts Screenshot
Type "brum"
Enter
Sleep 2s
Screenshot ../static/img/screenshots/react-scripts.png

# Start dev server
Enter
Sleep 2s

# Switch to logs to show dev server output
Tab
Tab
Sleep 1s
Screenshot ../static/img/screenshots/react-dev-server.png

# Go back and start tests
Tab
Tab
Tab
Tab
Tab
Sleep 500ms
Type "jjj"  # Navigate to test script
Enter
Sleep 2s

# Show processes view with both running
Tab
Sleep 1s
Screenshot ../static/img/screenshots/react-multiple-processes.png

# Show test output in logs
Tab
Sleep 1s
Screenshot ../static/img/screenshots/react-tests.png

Sleep 1s