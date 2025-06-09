# VHS script to capture Brummer main TUI
# Output: brummer-tui.png

Output ../static/img/brummer-tui.png

Set FontSize 14
Set Width 1200
Set Height 800
Set Theme "Dracula"

# Show Brummer starting up
Type "brum"
Enter
Sleep 2s

# Take screenshot of main scripts view
Screenshot

Sleep 1s