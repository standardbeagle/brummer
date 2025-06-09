# VHS script for tutorial first launch
# Output: tutorial-first-launch.png

Output ../static/img/screenshots/tutorial-first-launch.png

Set FontSize 14
Set Width 1200
Set Height 800
Set Theme "Dracula"

# Create a sample project
Type "mkdir my-fullstack-app && cd my-fullstack-app"
Enter
Sleep 1s

Type "npm init -y"
Enter
Sleep 2s

# Add some scripts to package.json
Type "cat > package.json << 'EOF'"
Enter
Type "{"
Enter
Type '  "name": "my-fullstack-app",'
Enter
Type '  "scripts": {'
Enter
Type '    "dev:server": "nodemon src/server.js",'
Enter
Type '    "dev:frontend": "echo '\''Frontend running on http://localhost:3001'\'' && sleep infinity",'
Enter
Type '    "test": "jest --watchAll",'
Enter
Type '    "lint": "eslint src/**/*.js",'
Enter
Type '    "build": "echo '\''Building project...'\'' && sleep 3 && echo '\''Build complete!'\''",'
Enter
Type '    "db:start": "echo '\''Starting database on port 5432...'\'' && sleep infinity"'
Enter
Type "  }"
Enter
Type "}"
Enter
Type "EOF"
Enter
Sleep 1s

# Start Brummer
Type "brum"
Enter
Sleep 2s

# Take screenshot showing available scripts
Screenshot

Sleep 1s