// JavaScript to inject into proxied pages
(function() {
    'use strict';
    
    // Configuration from proxy
    const BRUMMER_ENDPOINT = '{{BRUMMER_ENDPOINT}}';
    const BRUMMER_TOKEN = '{{BRUMMER_TOKEN}}';
    const PROCESS_NAME = '{{PROCESS_NAME}}';
    
    // Override console methods
    const originalConsole = {
        log: console.log,
        error: console.error,
        warn: console.warn,
        info: console.info,
        debug: console.debug
    };
    
    function sendToBrummer(data) {
        fetch(BRUMMER_ENDPOINT, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${BRUMMER_TOKEN}`
            },
            body: JSON.stringify({
                logData: {
                    ...data,
                    url: window.location.href,
                    timestamp: new Date().toISOString(),
                    source: 'proxy-injection',
                    processName: PROCESS_NAME
                }
            })
        }).catch(() => {
            // Silently fail to avoid infinite loops
        });
    }
    
    // Override console methods
    Object.keys(originalConsole).forEach(method => {
        console[method] = function(...args) {
            originalConsole[method].apply(console, args);
            
            sendToBrummer({
                type: 'console',
                level: method,
                message: args.map(arg => 
                    typeof arg === 'object' ? JSON.stringify(arg) : String(arg)
                ).join(' ')
            });
        };
    });
    
    // Capture errors
    window.addEventListener('error', (event) => {
        sendToBrummer({
            type: 'error',
            level: 'error',
            message: `${event.message} at ${event.filename}:${event.lineno}:${event.colno}`,
            stack: event.error?.stack
        });
    });
    
    // Capture unhandled promise rejections
    window.addEventListener('unhandledrejection', (event) => {
        sendToBrummer({
            type: 'promise-rejection',
            level: 'error',
            message: String(event.reason),
            reason: event.reason
        });
    });
    
    // Track navigation
    let lastUrl = location.href;
    const checkUrlChange = () => {
        if (location.href !== lastUrl) {
            lastUrl = location.href;
            sendToBrummer({
                type: 'navigation',
                level: 'info',
                message: `Navigated to ${location.href}`
            });
        }
    };
    
    // Monitor URL changes
    setInterval(checkUrlChange, 100);
    
    // Override fetch
    const originalFetch = window.fetch;
    window.fetch = function(...args) {
        const url = args[0];
        const startTime = Date.now();
        
        return originalFetch.apply(this, args)
            .then(response => {
                sendToBrummer({
                    type: 'network',
                    level: response.ok ? 'info' : 'warn',
                    message: `${response.status} ${response.statusText} - ${url} (${Date.now() - startTime}ms)`
                });
                return response;
            })
            .catch(error => {
                sendToBrummer({
                    type: 'network-error',
                    level: 'error',
                    message: `Network error - ${url}: ${error.message}`
                });
                throw error;
            });
    };
    
    console.log('🐝 Brummer proxy logging enabled');
})();