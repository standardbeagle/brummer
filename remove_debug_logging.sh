#!/bin/bash

# Remove all debug logging from model.go

FILE="/home/beagle/work/brummer/internal/tui/model.go"

# Remove debug blocks (debugFile patterns)
# This removes:
# - debugFile variable declarations
# - debugFile.Close() calls
# - fmt.Fprintf(debugFile, ...) calls
# - if debugFile != nil blocks

perl -i -0pe 's/\t\/\/ DEBUG:.*?\n//g' "$FILE"
perl -i -0pe 's/\tdebugFile.*?os\.OpenFile.*?\n//g' "$FILE"
perl -i -0pe 's/\tif debugFile != nil \{.*?debugFile\.Close\(\)\n\t\}\n//gs' "$FILE"
perl -i -0pe 's/\tdebugFile\.Close\(\)\n//g' "$FILE"
perl -i -0pe 's/\tfmt\.Fprintf\(debugFile,.*?\n//g' "$FILE"

# Remove standalone debug blocks
perl -i -0pe 's/\tdebugFile, _ := .*?\n//g' "$FILE"
perl -i -0pe 's/\tdebugFile, _ = .*?\n//g' "$FILE"
perl -i -0pe 's/\tdebugFile2, _ := .*?\n//g' "$FILE"
perl -i -0pe 's/\tdebugFile3, _ := .*?\n//g' "$FILE"

echo "Debug logging removed from $FILE"
