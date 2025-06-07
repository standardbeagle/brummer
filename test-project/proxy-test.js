const http = require('http');

const PORT = 3456;

// Create a simple HTTP server
const server = http.createServer((req, res) => {
  if (req.url === '/') {
    res.writeHead(200, { 'Content-Type': 'text/html' });
    res.end(`
      <!DOCTYPE html>
      <html>
      <head>
        <title>Proxy Test Server</title>
      </head>
      <body>
        <h1>Hello from Test Server!</h1>
        <p>This page should be proxied automatically by Brummer.</p>
        <button onclick="console.log('Button clicked!')">Test Console Log</button>
        <button onclick="console.error('Test error!')">Test Error</button>
        <button onclick="fetch('/api/test').then(r => console.log('API response:', r.status))">Test API Call</button>
        <script>
          console.log('Page loaded successfully');
          
          // Test navigation tracking
          setTimeout(() => {
            console.log('5 seconds have passed');
          }, 5000);
        </script>
      </body>
      </html>
    `);
  } else if (req.url === '/api/test') {
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ message: 'API response from test server' }));
  } else {
    res.writeHead(404);
    res.end('Not found');
  }
});

server.listen(PORT, () => {
  console.log(`Test server running at http://localhost:${PORT}/`);
  console.log('The proxy should automatically detect this URL and start proxying.');
  
  // Simulate some server activity
  setInterval(() => {
    console.log(`Server is still running... (${new Date().toISOString()})`);
  }, 10000);
});

// Handle graceful shutdown
process.on('SIGTERM', () => {
  console.log('Shutting down server...');
  server.close(() => {
    console.log('Server closed');
    process.exit(0);
  });
});