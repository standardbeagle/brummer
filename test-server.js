const http = require('http');
const fs = require('fs');
const path = require('path');

const PORT = 3333;

const server = http.createServer((req, res) => {
    console.log(`${req.method} ${req.url}`);
    
    if (req.url === '/' || req.url === '/index.html') {
        // Serve the test HTML file
        const htmlPath = path.join(__dirname, 'test-telemetry.html');
        fs.readFile(htmlPath, 'utf8', (err, content) => {
            if (err) {
                res.writeHead(500, { 'Content-Type': 'text/plain' });
                res.end('Error loading test page');
                return;
            }
            res.writeHead(200, { 'Content-Type': 'text/html' });
            res.end(content);
        });
    } else if (req.url === '/api/test') {
        // Test API endpoint
        res.writeHead(200, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({
            message: 'Test API response',
            timestamp: new Date().toISOString()
        }));
    } else {
        res.writeHead(404, { 'Content-Type': 'text/plain' });
        res.end('Not found');
    }
});

server.listen(PORT, () => {
    console.log(`Test server running at http://localhost:${PORT}`);
    console.log('');
    console.log('To test the enhanced telemetry:');
    console.log('1. Start Brummer in another terminal: ./brum');
    console.log('2. Look for the proxy URL in the Brummer output');
    console.log('3. Open the proxy URL in your browser');
    console.log('4. Interact with the test page and watch the telemetry in Brummer');
});