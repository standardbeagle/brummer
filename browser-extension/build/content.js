// Content script for capturing browser logs, errors, and network events for Brummer
(function() {
    'use strict';
    
    let lastUrl = location.href;
    let brummerEnabled = false;
    let brummerToken = null;
    let brummerEndpoint = null;
    let logBuffer = [];
    
    // Check URL for brummer token
    const urlParams = new URLSearchParams(window.location.search);
    if (urlParams.has('brummer_token')) {
        brummerToken = urlParams.get('brummer_token');
        brummerEnabled = true;
        
        // Get the brummer base URL from parameters
        const brummerBase = urlParams.get('brummer_base') || 'http://localhost:7777';
        brummerEndpoint = brummerBase + '/api/browser-log';
        
        // Enhanced logging with styled console output
        console.log('%c🐝 Brummer Extension Activated', 'background: #FFD700; color: #000; padding: 5px 10px; font-weight: bold; border-radius: 3px;');
        console.log('%cURL Parameters Recognized:', 'color: #4CAF50; font-weight: bold;');
        console.log('  Token:', brummerToken);
        console.log('  Endpoint:', brummerEndpoint);
        console.log('  Process:', urlParams.get('brummer_process') || 'unknown');
        
        // Create connection status indicator
        createConnectionStatus();
        
        initializeBrowserLogging();
        
        // Start connection monitoring
        startConnectionMonitoring();
    }
    
    // Check if Brummer logging is enabled (skip if already enabled by token)
    if (!brummerToken) {
        chrome.storage.local.get(['brummerLoggingEnabled'], (result) => {
            brummerEnabled = result.brummerLoggingEnabled || false;
            if (brummerEnabled) {
                initializeBrowserLogging();
            }
        });
    }
    
    // Listen for enable/disable messages from extension (skip if using token)
    chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
        if (message.type === 'brummer_toggle_logging' && !brummerToken) {
            brummerEnabled = message.enabled;
            if (brummerEnabled) {
                initializeBrowserLogging();
            }
            sendResponse({success: true});
        }
    });
    
    function initializeBrowserLogging() {
        // Capture console logs
        interceptConsole();
        
        // Capture JavaScript errors
        captureErrors();
        
        // Capture network events
        captureNetworkEvents();
        
        // URL change detection
        setupUrlDetection();
    }
    
    function sendToBrummer(logData) {
        if (!brummerEnabled) return;
        
        // If we have a token, send directly to the API endpoint
        if (brummerToken && brummerEndpoint) {
            fetch(brummerEndpoint, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${brummerToken}`
                },
                body: JSON.stringify({
                    logData: logData
                })
            }).catch(error => {
                console.error('Failed to send log to Brummer:', error);
            });
        } else {
            // Fallback to extension messaging
            chrome.runtime.sendMessage({
                type: 'brummer_browser_log',
                data: logData
            });
        }
    }
    
    function interceptConsole() {
        const originalConsole = {
            log: console.log,
            warn: console.warn,
            error: console.error,
            info: console.info,
            debug: console.debug
        };
        
        function createLogInterceptor(level, originalMethod) {
            return function(...args) {
                // Call original console method
                originalMethod.apply(console, args);
                
                // Send to Brummer
                const message = args.map(arg => 
                    typeof arg === 'object' ? JSON.stringify(arg, null, 2) : String(arg)
                ).join(' ');
                
                sendToBrummer({
                    type: 'console',
                    level: level,
                    message: message,
                    url: location.href,
                    timestamp: new Date().toISOString(),
                    source: 'browser-console'
                });
            };
        }
        
        console.log = createLogInterceptor('info', originalConsole.log);
        console.warn = createLogInterceptor('warn', originalConsole.warn);
        console.error = createLogInterceptor('error', originalConsole.error);
        console.info = createLogInterceptor('info', originalConsole.info);
        console.debug = createLogInterceptor('debug', originalConsole.debug);
    }
    
    function captureErrors() {
        // Capture JavaScript errors
        window.addEventListener('error', (event) => {
            sendToBrummer({
                type: 'javascript-error',
                level: 'error',
                message: `${event.error?.name || 'Error'}: ${event.message}`,
                details: {
                    filename: event.filename,
                    lineno: event.lineno,
                    colno: event.colno,
                    stack: event.error?.stack
                },
                url: location.href,
                timestamp: new Date().toISOString(),
                source: 'browser-error'
            });
        });
        
        // Capture unhandled promise rejections
        window.addEventListener('unhandledrejection', (event) => {
            sendToBrummer({
                type: 'promise-rejection',
                level: 'error',
                message: `Unhandled Promise Rejection: ${event.reason}`,
                details: {
                    reason: event.reason,
                    promise: event.promise
                },
                url: location.href,
                timestamp: new Date().toISOString(),
                source: 'browser-error'
            });
        });
        
        // Capture resource loading errors
        window.addEventListener('error', (event) => {
            if (event.target !== window) {
                const element = event.target;
                const resourceType = element.tagName.toLowerCase();
                const src = element.src || element.href;
                
                if (src) {
                    sendToBrummer({
                        type: 'resource-error',
                        level: 'error',
                        message: `Failed to load ${resourceType}: ${src}`,
                        details: {
                            resourceType: resourceType,
                            src: src,
                            element: element.outerHTML
                        },
                        url: location.href,
                        timestamp: new Date().toISOString(),
                        source: 'browser-resource'
                    });
                }
            }
        }, true);
    }
    
    function captureNetworkEvents() {
        // Intercept fetch requests
        const originalFetch = window.fetch;
        window.fetch = function(...args) {
            const startTime = Date.now();
            const url = args[0];
            const options = args[1] || {};
            
            return originalFetch.apply(this, args)
                .then(response => {
                    const duration = Date.now() - startTime;
                    
                    sendToBrummer({
                        type: 'network-request',
                        level: response.ok ? 'info' : 'warn',
                        message: `${options.method || 'GET'} ${url} → ${response.status} ${response.statusText} (${duration}ms)`,
                        details: {
                            method: options.method || 'GET',
                            url: url,
                            status: response.status,
                            statusText: response.statusText,
                            duration: duration,
                            headers: Object.fromEntries(response.headers.entries()),
                            ok: response.ok
                        },
                        url: location.href,
                        timestamp: new Date().toISOString(),
                        source: 'browser-network'
                    });
                    
                    return response;
                })
                .catch(error => {
                    const duration = Date.now() - startTime;
                    
                    sendToBrummer({
                        type: 'network-error',
                        level: 'error',
                        message: `${options.method || 'GET'} ${url} → Network Error (${duration}ms): ${error.message}`,
                        details: {
                            method: options.method || 'GET',
                            url: url,
                            error: error.message,
                            duration: duration
                        },
                        url: location.href,
                        timestamp: new Date().toISOString(),
                        source: 'browser-network'
                    });
                    
                    throw error;
                });
        };
        
        // Intercept XMLHttpRequest
        const originalXHROpen = XMLHttpRequest.prototype.open;
        const originalXHRSend = XMLHttpRequest.prototype.send;
        
        XMLHttpRequest.prototype.open = function(method, url, ...args) {
            this._brummerMethod = method;
            this._brummerUrl = url;
            this._brummerStartTime = Date.now();
            
            return originalXHROpen.apply(this, [method, url, ...args]);
        };
        
        XMLHttpRequest.prototype.send = function(...args) {
            this.addEventListener('loadend', () => {
                const duration = Date.now() - this._brummerStartTime;
                
                sendToBrummer({
                    type: 'network-request',
                    level: this.status >= 200 && this.status < 400 ? 'info' : 'warn',
                    message: `${this._brummerMethod} ${this._brummerUrl} → ${this.status} ${this.statusText} (${duration}ms)`,
                    details: {
                        method: this._brummerMethod,
                        url: this._brummerUrl,
                        status: this.status,
                        statusText: this.statusText,
                        duration: duration,
                        responseURL: this.responseURL
                    },
                    url: location.href,
                    timestamp: new Date().toISOString(),
                    source: 'browser-network'
                });
            });
            
            this.addEventListener('error', () => {
                const duration = Date.now() - this._brummerStartTime;
                
                sendToBrummer({
                    type: 'network-error',
                    level: 'error',
                    message: `${this._brummerMethod} ${this._brummerUrl} → Network Error (${duration}ms)`,
                    details: {
                        method: this._brummerMethod,
                        url: this._brummerUrl,
                        duration: duration
                    },
                    url: location.href,
                    timestamp: new Date().toISOString(),
                    source: 'browser-network'
                });
            });
            
            return originalXHRSend.apply(this, args);
        };
    }
    
    function setupUrlDetection() {
        function urlChanged() {
            if (lastUrl !== location.href) {
                sendToBrummer({
                    type: 'navigation',
                    level: 'info',
                    message: `Navigation: ${lastUrl} → ${location.href}`,
                    details: {
                        from: lastUrl,
                        to: location.href,
                        title: document.title
                    },
                    url: location.href,
                    timestamp: new Date().toISOString(),
                    source: 'browser-navigation'
                });
                
                lastUrl = location.href;
            }
        }
        
        // Listen for navigation changes
        window.addEventListener('popstate', urlChanged);
        
        // For SPAs, also watch for pushState/replaceState
        const originalPushState = history.pushState;
        const originalReplaceState = history.replaceState;
        
        history.pushState = function() {
            originalPushState.apply(history, arguments);
            setTimeout(urlChanged, 0);
        };
        
        history.replaceState = function() {
            originalReplaceState.apply(history, arguments);
            setTimeout(urlChanged, 0);
        };
        
        // Initial check
        urlChanged();
    }
    
    // Connection status UI elements
    let statusIndicator = null;
    let connectionState = 'connecting';
    let lastPingTime = null;
    let pingInterval = null;
    
    function createConnectionStatus() {
        // Create status indicator element
        statusIndicator = document.createElement('div');
        statusIndicator.id = 'brummer-status';
        statusIndicator.style.cssText = `
            position: fixed;
            bottom: 20px;
            right: 20px;
            background: rgba(0, 0, 0, 0.8);
            color: white;
            padding: 10px 15px;
            border-radius: 8px;
            font-family: monospace;
            font-size: 12px;
            z-index: 999999;
            display: flex;
            align-items: center;
            gap: 8px;
            transition: all 0.3s ease;
            cursor: pointer;
            user-select: none;
        `;
        
        // Add hover effect
        statusIndicator.addEventListener('mouseenter', () => {
            statusIndicator.style.transform = 'scale(1.05)';
        });
        
        statusIndicator.addEventListener('mouseleave', () => {
            statusIndicator.style.transform = 'scale(1)';
        });
        
        // Click to hide/show
        statusIndicator.addEventListener('click', () => {
            if (statusIndicator.style.opacity === '0.3') {
                statusIndicator.style.opacity = '1';
            } else {
                statusIndicator.style.opacity = '0.3';
            }
        });
        
        updateConnectionStatus('connecting', 'Connecting to Brummer...');
        document.body.appendChild(statusIndicator);
    }
    
    function updateConnectionStatus(state, message) {
        if (!statusIndicator) return;
        
        connectionState = state;
        
        const stateColors = {
            'connected': '#4CAF50',
            'connecting': '#FFC107',
            'disconnected': '#F44336',
            'error': '#F44336'
        };
        
        const stateIcons = {
            'connected': '🟢',
            'connecting': '🟡',
            'disconnected': '🔴',
            'error': '⚠️'
        };
        
        const color = stateColors[state] || '#999';
        const icon = stateIcons[state] || '❓';
        
        statusIndicator.innerHTML = `
            <span style="font-size: 16px;">${icon}</span>
            <div>
                <div style="font-weight: bold;">Brummer</div>
                <div style="font-size: 10px; opacity: 0.8;">${message}</div>
            </div>
        `;
        
        statusIndicator.style.borderLeft = `3px solid ${color}`;
    }
    
    function startConnectionMonitoring() {
        // Initial ping
        sendPing();
        
        // Set up regular pings every 5 seconds
        pingInterval = setInterval(() => {
            sendPing();
        }, 5000);
        
        // Cleanup on page unload
        window.addEventListener('beforeunload', () => {
            if (pingInterval) {
                clearInterval(pingInterval);
            }
        });
    }
    
    async function sendPing() {
        if (!brummerEndpoint || !brummerToken) return;
        
        const startTime = Date.now();
        
        try {
            const response = await fetch(brummerEndpoint.replace('/api/browser-log', '/api/ping'), {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${brummerToken}`
                },
                body: JSON.stringify({
                    timestamp: new Date().toISOString()
                })
            });
            
            if (response.ok) {
                const latency = Date.now() - startTime;
                lastPingTime = Date.now();
                updateConnectionStatus('connected', `Connected (${latency}ms)`);
                
                // Log successful ping in console
                console.log(`%c✓ Brummer ping: ${latency}ms`, 'color: #4CAF50; font-size: 10px;');
            } else {
                throw new Error(`HTTP ${response.status}`);
            }
        } catch (error) {
            // Check if we've been disconnected for more than 10 seconds
            if (lastPingTime && Date.now() - lastPingTime > 10000) {
                updateConnectionStatus('disconnected', 'Connection lost');
            } else {
                updateConnectionStatus('error', `Error: ${error.message}`);
            }
            
            console.error('%c✗ Brummer ping failed:', 'color: #F44336; font-size: 10px;', error.message);
        }
    }
    
})();