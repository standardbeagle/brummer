# VHS master script to generate all screenshots
# This demonstrates various Brummer features

Output ../static/img/screenshots/demo.gif

Set FontSize 14
Set Width 1200
Set Height 800
Set Theme "Dracula"
Set TypingSpeed 100ms

Hide
Type "cd /tmp && mkdir brummer-demo && cd brummer-demo"
Enter
Sleep 500ms

# Create a realistic package.json
Type "cat > package.json << 'EOF'"
Enter
Type '{'
Enter
Type '  "name": "my-fullstack-app",'
Enter
Type '  "version": "1.0.0",'
Enter
Type '  "scripts": {'
Enter
Type '    "dev": "concurrently \"npm:dev:*\"",'
Enter
Type '    "dev:frontend": "vite",'
Enter
Type '    "dev:backend": "nodemon server.js",'
Enter
Type '    "dev:workers": "node workers/process-jobs.js",'
Enter
Type '    "test": "jest --watch",'
Enter
Type '    "test:e2e": "playwright test",'
Enter
Type '    "lint": "eslint src --ext .js,.jsx,.ts,.tsx",'
Enter
Type '    "typecheck": "tsc --noEmit",'
Enter
Type '    "build": "vite build",'
Enter
Type '    "db:migrate": "prisma migrate dev",'
Enter
Type '    "db:seed": "node prisma/seed.js",'
Enter
Type '    "db:studio": "prisma studio"'
Enter
Type '  }'
Enter
Type '}'
Enter
Type "EOF"
Enter
Sleep 500ms
Show

# Main TUI Screenshot
Type "brum"
Enter
Sleep 2s
Screenshot ../static/img/screenshots/brummer-main-tui.png

# Navigate and start some processes
Type "j"  # Move down
Sleep 200ms
Enter     # Start dev:frontend
Sleep 1s

Type "j"  # Move down
Sleep 200ms
Enter     # Start dev:backend
Sleep 1s

Type "j"
Type "j"
Sleep 200ms
Enter     # Start test
Sleep 1s

# Switch to Processes view
Tab
Sleep 1s
Screenshot ../static/img/screenshots/processes-running.png

# Show process control
Type "j"  # Select a process
Sleep 500ms
Type "s"  # Stop it
Sleep 1s
Type "r"  # Restart it
Sleep 1s
Screenshot ../static/img/screenshots/process-control.png

# Switch to Logs view
Tab
Sleep 1s
Screenshot ../static/img/screenshots/logs-view.png

# Show log filtering
Type "/"
Sleep 500ms
Type "error"
Enter
Sleep 1s
Screenshot ../static/img/screenshots/logs-filtered.png

# Clear filter
Type "/"
Enter
Sleep 500ms

# Switch to Errors view
Tab
Sleep 1s
Screenshot ../static/img/screenshots/errors-view.png

# Switch to URLs view
Tab
Sleep 1s
Screenshot ../static/img/screenshots/urls-view.png

# Switch to Settings view
Tab
Sleep 1s
Screenshot ../static/img/screenshots/settings-view.png

# Exit
Type "q"
Sleep 500ms