const http = require('http');

const PORT = 3457;

// Create a test server with various pages to test telemetry
const server = http.createServer((req, res) => {
  if (req.url === '/') {
    res.writeHead(200, { 'Content-Type': 'text/html' });
    res.end(`
      <!DOCTYPE html>
      <html>
      <head>
        <title>Brummer Telemetry Test</title>
        <style>
          body { font-family: Arial, sans-serif; padding: 20px; }
          button { margin: 5px; padding: 10px; cursor: pointer; }
          .section { margin: 20px 0; padding: 15px; border: 1px solid #ccc; }
          #output { background: #f0f0f0; padding: 10px; margin-top: 10px; }
        </style>
      </head>
      <body>
        <h1>Brummer Telemetry Test Page</h1>
        <p>This page tests all telemetry features. Check the proxy telemetry data to verify.</p>
        
        <div class="section">
          <h2>Console Monitoring</h2>
          <button onclick="console.log('Test log message')">Console Log</button>
          <button onclick="console.info('Test info message')">Console Info</button>
          <button onclick="console.warn('Test warning message')">Console Warn</button>
          <button onclick="console.error('Test error message')">Console Error</button>
          <button onclick="console.debug('Test debug message')">Console Debug</button>
        </div>
        
        <div class="section">
          <h2>Error Monitoring</h2>
          <button onclick="throw new Error('Test JavaScript Error')">Throw Error</button>
          <button onclick="undefinedFunction()">Call Undefined Function</button>
          <button onclick="Promise.reject('Test unhandled rejection')">Unhandled Promise Rejection</button>
        </div>
        
        <div class="section">
          <h2>Performance Testing</h2>
          <button onclick="heavyComputation()">Heavy Computation (Long Task)</button>
          <button onclick="allocateMemory()">Allocate Memory</button>
        </div>
        
        <div class="section">
          <h2>Network Activity</h2>
          <button onclick="fetchData('/api/test')">Fetch API Data</button>
          <button onclick="fetchData('/api/slow')">Fetch Slow API (2s delay)</button>
          <button onclick="fetchData('/api/error')">Fetch Error API</button>
          <button onclick="loadImage()">Load Image</button>
        </div>
        
        <div class="section">
          <h2>User Interaction</h2>
          <form id="testForm" onsubmit="handleSubmit(event)">
            <input type="text" name="testInput" placeholder="Type something here">
            <input type="email" name="testEmail" placeholder="Email">
            <button type="submit">Submit Form</button>
          </form>
        </div>
        
        <div class="section">
          <h2>Output</h2>
          <div id="output">Telemetry events will be logged here...</div>
        </div>
        
        <script>
          // Log initial page load
          console.log('Telemetry test page loaded');
          
          // Output logging helper
          function log(message) {
            const output = document.getElementById('output');
            const timestamp = new Date().toLocaleTimeString();
            output.innerHTML += timestamp + ' - ' + message + '<br>';
            output.scrollTop = output.scrollHeight;
          }
          
          // Heavy computation function
          function heavyComputation() {
            log('Starting heavy computation...');
            const start = Date.now();
            let result = 0;
            for (let i = 0; i < 100000000; i++) {
              result += Math.sqrt(i);
            }
            const duration = Date.now() - start;
            log('Heavy computation completed in ' + duration + 'ms');
            console.log('Computation result:', result);
          }
          
          // Memory allocation function
          function allocateMemory() {
            log('Allocating memory...');
            const arrays = [];
            for (let i = 0; i < 10; i++) {
              arrays.push(new Array(1000000).fill(i));
            }
            log('Allocated ' + (arrays.length * 1000000 * 8 / 1024 / 1024).toFixed(2) + ' MB');
            console.log('Memory allocated');
          }
          
          // Fetch data function
          async function fetchData(url) {
            log('Fetching ' + url + '...');
            try {
              const response = await fetch(url);
              const data = await response.json();
              log('Fetch successful: ' + JSON.stringify(data));
              console.log('API response:', data);
            } catch (error) {
              log('Fetch error: ' + error.message);
              console.error('Fetch error:', error);
            }
          }
          
          // Load image function
          function loadImage() {
            log('Loading image...');
            const img = new Image();
            img.onload = () => {
              log('Image loaded successfully');
              console.log('Image loaded:', img.src);
            };
            img.onerror = () => {
              log('Image load error');
              console.error('Image load failed');
            };
            img.src = '/test-image.jpg?t=' + Date.now();
          }
          
          // Form submit handler
          function handleSubmit(event) {
            event.preventDefault();
            const formData = new FormData(event.target);
            const data = Object.fromEntries(formData);
            log('Form submitted: ' + JSON.stringify(data));
            console.log('Form data:', data);
          }
          
          // Test performance timing after load
          window.addEventListener('load', () => {
            setTimeout(() => {
              if (window.performance && window.performance.timing) {
                const timing = window.performance.timing;
                const pageLoadTime = timing.loadEventEnd - timing.navigationStart;
                log('Page load time: ' + pageLoadTime + 'ms');
                console.log('Performance timing:', timing);
              }
            }, 1000);
          });
          
          // Monitor visibility changes
          document.addEventListener('visibilitychange', () => {
            log('Visibility changed: ' + (document.hidden ? 'hidden' : 'visible'));
            console.log('Page visibility:', document.visibilityState);
          });
        </script>
      </body>
      </html>
    `);
  } else if (req.url === '/api/test') {
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ 
      message: 'Test API response',
      timestamp: new Date().toISOString()
    }));
  } else if (req.url === '/api/slow') {
    // Simulate slow API
    setTimeout(() => {
      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ 
        message: 'Slow API response',
        delay: '2 seconds'
      }));
    }, 2000);
  } else if (req.url === '/api/error') {
    res.writeHead(500, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ 
      error: 'Internal server error',
      message: 'This is a test error response'
    }));
  } else if (req.url.startsWith('/test-image.jpg')) {
    // Simulate image response
    res.writeHead(200, { 'Content-Type': 'image/jpeg' });
    res.end(Buffer.from('fake image data'));
  } else {
    res.writeHead(404);
    res.end('Not found');
  }
});

server.listen(PORT, () => {
  console.log(`Telemetry test server running at http://localhost:${PORT}/`);
  console.log('');
  console.log('To test telemetry:');
  console.log('1. Make sure Brummer proxy is running');
  console.log('2. Configure your browser to use the proxy');
  console.log('3. Visit http://localhost:' + PORT + '/');
  console.log('4. Interact with the page elements');
  console.log('5. Check Brummer telemetry data');
  console.log('');
  
  // Log server activity
  setInterval(() => {
    console.log(`[${new Date().toISOString()}] Server is running...`);
  }, 30000);
});

// Handle graceful shutdown
process.on('SIGTERM', () => {
  console.log('Shutting down telemetry test server...');
  server.close(() => {
    console.log('Server closed');
    process.exit(0);
  });
});