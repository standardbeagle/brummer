// Content script for capturing browser logs, errors, and network events for Brummer
(function() {
    'use strict';
    
    let lastUrl = location.href;
    let brummerEnabled = false;
    let logBuffer = [];
    
    // Check if Brummer logging is enabled
    chrome.storage.local.get(['brummerLoggingEnabled'], (result) => {
        brummerEnabled = result.brummerLoggingEnabled || false;
        if (brummerEnabled) {
            initializeBrowserLogging();
        }
    });
    
    // Listen for enable/disable messages from extension
    chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
        if (message.type === 'brummer_toggle_logging') {
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
        
        chrome.runtime.sendMessage({
            type: 'brummer_browser_log',
            data: logData
        });
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
    
})();