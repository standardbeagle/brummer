#!/bin/bash
echo "Testing real implementation with component system..."

# Create a test directory with package.json
TESTDIR=$(mktemp -d)
cat > $TESTDIR/package.json << 'JSON'
{
  "name": "test",
  "scripts": {
    "hello": "echo 'Hello World from test!'"
  }
}
JSON

# Start brummer with component system in PTY
export BRUMMER_USE_COMPONENTS=true

echo "Starting brummer in test directory: $TESTDIR"
cd $TESTDIR

# Use script to run in a PTY
(
  sleep 2
  echo -e "\t"    # Tab key to scripts
  sleep 1
  echo -e "\r"    # Enter to run first script
  sleep 2
  echo "2"        # Switch to logs view
  sleep 2
  echo "q"        # Quit
) | script -q -c "/home/beagle/work/brummer/brum" /tmp/brummer_test.log

echo "Test output (searching for Hello or logs):"
cat /tmp/brummer_test.log | col -b | grep -i -E "(Hello|Logs|No logs)"

# Cleanup
rm -rf $TESTDIR
rm -f /tmp/brummer_test.log
