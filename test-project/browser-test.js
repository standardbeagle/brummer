#!/usr/bin/env node

// Test script to verify browser extension logging and connection status

const http = require('http');

console.log('🚀 Starting browser test server...');

// Create a simple HTTP server
const server = http.createServer((req, res) => {
    console.log(`📥 ${req.method} ${req.url}`);
    
    // Send HTML response
    res.writeHead(200, { 'Content-Type': 'text/html' });
    res.end(`
        <!DOCTYPE html>
        <html>
        <head>
            <title>Brummer Browser Test</title>
            <style>
                body {
                    font-family: monospace;
                    padding: 20px;
                    background: #1e1e1e;
                    color: #fff;
                }
                button {
                    background: #4CAF50;
                    color: white;
                    border: none;
                    padding: 10px 20px;
                    margin: 5px;
                    cursor: pointer;
                    border-radius: 4px;
                }
                button:hover {
                    background: #45a049;
                }
                .error-btn {
                    background: #f44336;
                }
                .error-btn:hover {
                    background: #da190b;
                }
                pre {
                    background: #2d2d2d;
                    padding: 10px;
                    border-radius: 4px;
                    overflow-x: auto;
                }
            </style>
        </head>
        <body>
            <h1>🐝 Brummer Browser Test Page</h1>
            <p>This page tests the Brummer browser extension features.</p>
            
            <h2>Console Logging Tests</h2>
            <button onclick="testConsoleLog()">Test console.log</button>
            <button onclick="testConsoleWarn()">Test console.warn</button>
            <button onclick="testConsoleError()" class="error-btn">Test console.error</button>
            
            <h2>Error Tests</h2>
            <button onclick="testJSError()" class="error-btn">Throw JS Error</button>
            <button onclick="testPromiseRejection()" class="error-btn">Promise Rejection</button>
            
            <h2>Network Tests</h2>
            <button onclick="testFetch()">Test Fetch API</button>
            <button onclick="testFetchError()" class="error-btn">Test Fetch Error</button>
            
            <h2>Log Output</h2>
            <pre id="logOutput"></pre>
            
            <script>
                const log = document.getElementById('logOutput');
                
                function addLog(msg) {
                    log.textContent += new Date().toLocaleTimeString() + ' - ' + msg + '\\n';
                    console.log(msg);
                }
                
                function testConsoleLog() {
                    addLog('✅ This is a test console.log message');
                }
                
                function testConsoleWarn() {
                    console.warn('⚠️ This is a test warning message');
                    addLog('⚠️ Warning logged');
                }
                
                function testConsoleError() {
                    console.error('❌ This is a test error message');
                    addLog('❌ Error logged');
                }
                
                function testJSError() {
                    addLog('💥 Throwing JS error...');
                    throw new Error('This is a test JavaScript error');
                }
                
                function testPromiseRejection() {
                    addLog('🔥 Creating promise rejection...');
                    Promise.reject('This is a test promise rejection');
                }
                
                async function testFetch() {
                    addLog('🌐 Making fetch request...');
                    try {
                        const response = await fetch('/api/test');
                        addLog('✅ Fetch completed: ' + response.status);
                    } catch (e) {
                        addLog('❌ Fetch failed: ' + e.message);
                    }
                }
                
                async function testFetchError() {
                    addLog('🌐 Making fetch request to invalid URL...');
                    try {
                        const response = await fetch('http://invalid-domain-12345.test');
                        addLog('✅ Fetch completed: ' + response.status);
                    } catch (e) {
                        addLog('❌ Fetch failed: ' + e.message);
                    }
                }
                
                // Auto-run some tests
                setTimeout(() => {
                    addLog('🤖 Running automatic tests...');
                    testConsoleLog();
                }, 1000);
                
                setTimeout(() => {
                    testConsoleWarn();
                }, 2000);
                
                // Log page load
                addLog('📄 Page loaded successfully');
            </script>
        </body>
        </html>
    `);
});

const PORT = 8888;
server.listen(PORT, () => {
    console.log(`✅ Test server running at http://localhost:${PORT}`);
    console.log('');
    console.log('To test the browser extension:');
    console.log('1. Start Brummer TUI: ./brummer');
    console.log('2. Run this script from the Scripts view');
    console.log('3. The browser should open with Brummer parameters');
    console.log('4. Check the browser console for styled Brummer logs');
    console.log('5. Look for the connection status indicator in the bottom-right');
    console.log('');
});

// Keep the server running
process.on('SIGINT', () => {
    console.log('\n👋 Shutting down test server...');
    server.close();
    process.exit(0);
});