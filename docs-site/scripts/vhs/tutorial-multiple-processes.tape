# VHS script showing multiple processes
# Output: tutorial-multiple-processes.png

Output ../static/img/screenshots/tutorial-multiple-processes.png

Set FontSize 14
Set Width 1200
Set Height 800
Set Theme "Dracula"

# Assume Brummer is already running
Type "brum"
Enter
Sleep 2s

# Start first process (dev:server)
Down
Enter
Sleep 1s

# Start second process (db:start)
Down
Down
Enter
Sleep 1s

# Start third process (test)
Down
Enter
Sleep 1s

# Switch to Processes view
Tab
Sleep 1s

# Take screenshot showing multiple running processes
Screenshot

Sleep 1s