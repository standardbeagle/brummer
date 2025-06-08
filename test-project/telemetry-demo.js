const express = require('express');
const app = express();
const port = 3458;

// Serve a test page with various JavaScript features
app.get('/', (req, res) => {
  res.send(`
<!DOCTYPE html>
<html>
<head>
  <title>Telemetry Demo</title>
  <style>
    body { font-family: Arial, sans-serif; padding: 20px; }
    button { margin: 10px; padding: 10px 20px; font-size: 16px; }
    .result { margin: 20px 0; padding: 10px; background: #f0f0f0; }
  </style>
</head>
<body>
  <h1>Brummer Telemetry Demo</h1>
  
  <h2>Console Logging</h2>
  <button onclick="testConsole()">Test Console Messages</button>
  
  <h2>Error Generation</h2>
  <button onclick="testError()">Throw Error</button>
  <button onclick="testPromiseReject()">Reject Promise</button>
  
  <h2>Performance Test</h2>
  <button onclick="testPerformance()">Run Performance Test</button>
  
  <h2>Memory Usage</h2>
  <button onclick="testMemory()">Allocate Memory</button>
  
  <div id="result" class="result"></div>
  
  <script>
    // Page load timing
    window.addEventListener('load', () => {
      console.log('Page fully loaded');
      console.info('Performance timing:', performance.timing);
    });
    
    function testConsole() {
      console.log('This is a log message');
      console.info('This is an info message');
      console.warn('This is a warning');
      console.error('This is an error message');
      console.debug('This is a debug message');
      updateResult('Console messages sent - check the proxy telemetry!');
    }
    
    function testError() {
      try {
        throw new Error('This is a test error!');
      } catch (e) {
        console.error('Caught error:', e);
        // Also let one through uncaught
        setTimeout(() => {
          nonExistentFunction();
        }, 100);
      }
      updateResult('Error thrown - check telemetry for details!');
    }
    
    function testPromiseReject() {
      Promise.reject(new Error('Unhandled promise rejection'));
      updateResult('Promise rejected - check telemetry!');
    }
    
    function testPerformance() {
      const start = performance.now();
      let sum = 0;
      for (let i = 0; i < 10000000; i++) {
        sum += Math.sqrt(i);
      }
      const duration = performance.now() - start;
      updateResult(\`Performance test completed in \${duration.toFixed(2)}ms\`);
      
      // Mark custom performance entry
      performance.mark('custom-test-complete');
    }
    
    let memoryData = [];
    function testMemory() {
      // Allocate some memory
      const bigArray = new Array(1000000).fill(Math.random());
      memoryData.push(bigArray);
      
      if (performance.memory) {
        const mem = performance.memory;
        updateResult(\`Memory used: \${(mem.usedJSHeapSize / 1048576).toFixed(2)}MB / \${(mem.totalJSHeapSize / 1048576).toFixed(2)}MB\`);
      } else {
        updateResult('Memory API not available in this browser');
      }
    }
    
    function updateResult(text) {
      document.getElementById('result').textContent = text;
    }
    
    // Simulate some background activity
    setInterval(() => {
      console.debug('Background activity tick');
    }, 5000);
  </script>
</body>
</html>
  `);
});

app.listen(port, () => {
  console.log(`Telemetry demo server running at http://localhost:${port}/`);
  console.log('Configure your browser to use the Brummer proxy to see telemetry data');
});