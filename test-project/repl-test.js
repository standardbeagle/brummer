const http = require('http');
const fs = require('fs');
const path = require('path');

const PORT = 3001;

const html = `
<!DOCTYPE html>
<html>
<head>
    <title>REPL Test Page</title>
    <script>
        // Some global variables for testing
        window.testValue = 42;
        window.testObject = {
            name: "Test Object",
            items: [1, 2, 3, 4, 5],
            nested: {
                deep: {
                    value: "Deep nested value"
                }
            }
        };
        
        window.testFunction = function() {
            return "Hello from test function!";
        };
        
        window.asyncFunction = async function() {
            await new Promise(resolve => setTimeout(resolve, 100));
            return "Async result after 100ms";
        };
    </script>
</head>
<body>
    <h1>REPL Test Page</h1>
    <p>This page has some test variables and functions for REPL testing.</p>
    <div id="testDiv">Test content in div</div>
</body>
</html>
`;

const server = http.createServer((req, res) => {
    console.log(`Request: ${req.method} ${req.url}`);
    
    if (req.url === '/') {
        res.writeHead(200, { 'Content-Type': 'text/html' });
        res.end(html);
    } else {
        res.writeHead(404);
        res.end('Not found');
    }
});

server.listen(PORT, () => {
    console.log(`REPL test server running on http://localhost:${PORT}`);
});