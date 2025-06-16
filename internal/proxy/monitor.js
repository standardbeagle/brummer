// Brummer Web Monitoring Script
// This script is injected into proxied web pages to collect telemetry data

(function() {
    'use strict';
    
    // Check if already initialized to prevent multiple injections
    if (window.__brummerInitialized) {
        return;
    }
    
    // Mark as initialized immediately
    window.__brummerInitialized = true;
    
    // Configuration
    const BRUMMER_CONFIG = {
        telemetryEndpoint: (function() {
            // Try to determine the proxy server URL
            // Default to localhost:8888 (standard Brummer proxy port)
            const proxyHost = window.__brummerProxyHost || 'localhost:8888';
            return 'http://' + proxyHost + '/__brummer_telemetry__';
        })(),
        websocketEndpoint: (function() {
            // WebSocket endpoint for real-time telemetry
            const proxyHost = window.__brummerProxyHost || 'localhost:8888';
            return 'ws://' + proxyHost + '/__brummer_ws__';
        })(),
        batchInterval: 2000, // Send data every 2 seconds
        maxBatchSize: 100,
        collectInteractionMetrics: true,
        collectPerformanceMetrics: true,
        collectMemoryMetrics: true,
        collectConsoleMetrics: false,
        processName: window.__brummerProcessName || 'unknown',
        debugMode: false, // Disable debug output to keep console clean
        debugLevel: 'minimal', // verbose, normal, minimal
        useWebSocket: true // Use WebSocket instead of HTTP
    };
    
    // Debug utilities for visual output
    const DEBUG = {
        colors: {
            success: '#22c55e',
            warning: '#f59e0b', 
            error: '#ef4444',
            info: '#3b82f6',
            debug: '#8b5cf6'
        },
        icons: {
            success: '✅',
            warning: '⚠️',
            error: '❌',
            info: 'ℹ️',
            debug: '🐛',
            network: '🌐',
            timer: '⏱️',
            buffer: '📊',
            send: '📤',
            receive: '📥',
            user: '👤',
            performance: '🚀',
            memory: '🧠',
            console: '💬'
        },
        
        formatTime: function(timestamp) {
            const now = Date.now();
            const diff = now - timestamp;
            
            if (diff < 1000) return 'just now';
            if (diff < 60000) return `${Math.floor(diff/1000)}s ago`;
            if (diff < 3600000) return `${Math.floor(diff/60000)}m ago`;
            return `${Math.floor(diff/3600000)}h ago`;
        },
        
        formatDuration: function(ms) {
            if (ms < 100) return `${this.icons.timer} ${ms}ms`;
            if (ms < 1000) return `⏳ ${ms}ms`;
            if (ms < 5000) return `🐌 ${(ms/1000).toFixed(1)}s`;
            return `🚨 ${(ms/1000).toFixed(1)}s`;
        },
        
        formatSize: function(bytes) {
            if (bytes < 1024) return `${bytes}B`;
            if (bytes < 1024 * 1024) return `${(bytes/1024).toFixed(1)}KB`;
            return `${(bytes/(1024*1024)).toFixed(1)}MB`;
        },
        
        progressBar: function(current, max, width = 10) {
            const filled = Math.floor((current / max) * width);
            const empty = width - filled;
            const percentage = Math.floor((current / max) * 100);
            return `[${'█'.repeat(filled)}${'░'.repeat(empty)}] ${percentage}% (${current}/${max})`;
        },
        
        log: function(level, category, message, data = null) {
            // Silently add to debug timeline without console output
            if (window.__brummer && window.__brummer.debug && window.__brummer.debug.eventTimeline) {
                window.__brummer.debug.eventTimeline.push({
                    timestamp: Date.now(),
                    level,
                    category,
                    message,
                    data
                });
                
                // Keep only last 50 entries
                if (window.__brummer.debug.eventTimeline.length > 50) {
                    window.__brummer.debug.eventTimeline.shift();
                }
            }
        },
        
        status: function() {
            if (!window.__brummer) return;
            
            const stats = window.__brummer.stats;
            const buffer = telemetryBuffer;
            const now = Date.now();
            
            // Only show status if explicitly called by user
            console.log('%c🔍 BRUMMER TELEMETRY DEBUG DASHBOARD', 'font-size: 18px; font-weight: bold; color: #3b82f6;');
            console.log('═══════════════════════════════════════');
            console.log(`${this.icons.buffer} Buffer: ${this.progressBar(buffer.length, BRUMMER_CONFIG.maxBatchSize)}`);
            console.log(`${this.icons.network} Endpoint: ${BRUMMER_CONFIG.telemetryEndpoint} ${stats.lastPingSuccess ? '🟢 ONLINE' : '🔴 OFFLINE'} (${stats.lastPingTime || 'unknown'})`);
            console.log(`${this.icons.send} Last Send: ${stats.lastSendTime ? this.formatTime(stats.lastSendTime) : 'never'} ${stats.lastSendSuccess ? '✅ SUCCESS' : '❌ FAILED'}`);
            console.log(`${this.icons.timer} Next Flush: ${batchTimer ? `in ${Math.ceil((BRUMMER_CONFIG.batchInterval - (now - stats.lastBatchStart))/1000)}s` : 'not scheduled'}`);
            console.log(`${this.icons.receive} Total Sent: ${stats.totalEvents} events (${this.formatSize(stats.totalBytes)})`);
            console.log(`${this.icons.error} Errors: ${stats.errorCount} failures`);
            console.log(`${this.icons.performance} Session: ${this.formatTime(pageMetadata.pageLoadTime)} (${this.formatDuration(now - pageMetadata.pageLoadTime)})`);
        }
    };
    
    // Telemetry buffer
    const telemetryBuffer = [];
    let batchTimer = null;
    
    // WebSocket connection
    let websocket = null;
    let wsConnected = false;
    let wsReconnectAttempts = 0;
    const wsMaxReconnectAttempts = 5;
    const wsReconnectDelay = 2000;
    
    // Statistics tracking
    const stats = {
        totalEvents: 0,
        totalBytes: 0,
        errorCount: 0,
        lastSendTime: null,
        lastSendSuccess: null,
        lastBatchStart: null,
        lastPingTime: null,
        lastPingSuccess: null,
        eventCounts: {
            page_load: 0,
            user_interaction: 0,
            performance_metrics: 0,
            memory_usage: 0,
            console_output: 0,
            javascript_error: 0,
            resource_timing: 0
        },
        networkStats: {
            requestCount: 0,
            successCount: 0,
            failureCount: 0,
            totalLatency: 0
        }
    };
    
    // Page metadata
    const pageMetadata = {
        url: window.location.href,
        referrer: document.referrer,
        userAgent: navigator.userAgent,
        sessionId: generateSessionId(),
        pageLoadTime: Date.now(),
        cookies: document.cookie ? document.cookie.split(';').length : 0,
        localStorage: typeof(Storage) !== "undefined" && localStorage.length || 0,
        sessionStorage: typeof(Storage) !== "undefined" && sessionStorage.length || 0
    };
    
    // Global debug interface
    window.__brummer = {
        config: BRUMMER_CONFIG,
        stats: stats,
        metadata: pageMetadata,
        buffer: telemetryBuffer,
        debug: {
            eventTimeline: [],
            status: DEBUG.status.bind(DEBUG),
            flush: () => flushTelemetry(),
            ping: () => BRUMMER_CONFIG.useWebSocket ? pingWebSocket() : pingEndpoint(),
            clear: () => {
                telemetryBuffer.length = 0;
                DEBUG.log('info', 'debug', 'Buffer cleared manually');
            },
            ws: () => websocket,
            reconnect: () => connectWebSocket(),
            timeline: () => {
                console.log('%c⏰ TELEMETRY TIMELINE (last 30 events)', 'font-size: 16px; font-weight: bold; color: #8b5cf6;');
                console.log('════════════════════════════════════');
                window.__brummer.debug.eventTimeline.slice(-30).forEach(entry => {
                    console.log(`${DEBUG.formatTime(entry.timestamp)} │ ${DEBUG.icons[entry.level]} ${DEBUG.icons[entry.category]} ${entry.message}`);
                });
            },
            headers: () => {
                console.log('%c📡 REQUEST HEADERS & CONTEXT', 'font-size: 16px; font-weight: bold; color: #3b82f6;');
                console.log('════════════════════════════════════');
                console.log(`🔐 Authorization: ${document.cookie.includes('auth') ? 'Present' : 'Not found'}`);
                console.log(`🍪 Cookies: ${pageMetadata.cookies} items`);
                console.log(`💾 Local Storage: ${pageMetadata.localStorage} items`);
                console.log(`💾 Session Storage: ${pageMetadata.sessionStorage} items`);
                console.log(`🌐 Origin: ${window.location.origin}`);
                console.log(`🔗 Referrer: ${pageMetadata.referrer || 'Direct'}`);
                console.log(`📱 User Agent: ${navigator.userAgent.substring(0, 80)}...`);
            }
        }
    };
    
    // Custom Metrics API (similar to mcpTelemetry)
    window.brummerTelemetry = {
        // Track custom events
        track: function(eventName, data) {
            sendTelemetry({
                type: 'custom_event',
                data: {
                    eventName: eventName,
                    data: data,
                    timestamp: Date.now()
                }
            });
            
            DEBUG.log('info', 'custom', `Custom event tracked: ${eventName}`, data);
        },
        
        // Mark performance points
        mark: function(name) {
            if (window.performance && window.performance.mark) {
                performance.mark(name);
                
                sendTelemetry({
                    type: 'performance_mark',
                    data: {
                        name: name,
                        timestamp: performance.now()
                    }
                });
                
                DEBUG.log('info', 'performance', `Performance mark: ${name}`);
            }
        },
        
        // Measure between marks
        measure: function(name, startMark, endMark) {
            if (window.performance && window.performance.measure) {
                try {
                    performance.measure(name, startMark, endMark);
                    const entries = performance.getEntriesByName(name, 'measure');
                    const measure = entries[entries.length - 1];
                    
                    if (measure) {
                        sendTelemetry({
                            type: 'performance_measure',
                            data: {
                                name: name,
                                duration: measure.duration,
                                startTime: measure.startTime
                            }
                        });
                        
                        DEBUG.log('info', 'performance', `Performance measure: ${name} = ${measure.duration.toFixed(2)}ms`);
                    }
                } catch (e) {
                    DEBUG.log('error', 'performance', `Failed to measure ${name}: ${e.message}`);
                }
            }
        },
        
        // Log custom error
        error: function(message, details) {
            sendTelemetry({
                type: 'custom_error',
                data: {
                    message: message,
                    details: details,
                    stack: (new Error()).stack,
                    timestamp: Date.now()
                }
            });
            
            DEBUG.log('error', 'custom', `Custom error: ${message}`, details);
        },
        
        // Track user actions
        action: function(action, target, metadata) {
            sendTelemetry({
                type: 'user_action',
                data: {
                    action: action,
                    target: target,
                    metadata: metadata,
                    timestamp: Date.now()
                }
            });
            
            DEBUG.log('info', 'user', `User action: ${action} on ${target}`, metadata);
        },
        
        // Track feature usage
        feature: function(featureName, metadata) {
            sendTelemetry({
                type: 'feature_usage',
                data: {
                    feature: featureName,
                    metadata: metadata,
                    timestamp: Date.now()
                }
            });
            
            DEBUG.log('info', 'custom', `Feature used: ${featureName}`, metadata);
        }
    };
    
    // Alias for compatibility
    window.mcpTelemetry = window.brummerTelemetry;
    
    // Generate a unique session ID
    function generateSessionId() {
        return 'brummer_' + Date.now() + '_' + Math.random().toString(36).substr(2, 9);
    }
    
    // Connect to WebSocket for real-time telemetry
    function connectWebSocket() {
        if (!BRUMMER_CONFIG.useWebSocket) return;
        
        if (websocket && (websocket.readyState === WebSocket.CONNECTING || websocket.readyState === WebSocket.OPEN)) {
            DEBUG.log('info', 'network', 'WebSocket already connected or connecting');
            return;
        }
        
        DEBUG.log('info', 'network', `🔌 Connecting to WebSocket: ${BRUMMER_CONFIG.websocketEndpoint}`);
        
        try {
            websocket = new WebSocket(BRUMMER_CONFIG.websocketEndpoint);
            
            websocket.onopen = function(event) {
                wsConnected = true;
                wsReconnectAttempts = 0;
                stats.lastPingSuccess = true;
                stats.lastPingTime = DEBUG.formatTime(Date.now());
                
                DEBUG.log('success', 'network', '🔗 WebSocket connected successfully', {
                    readyState: websocket.readyState,
                    url: BRUMMER_CONFIG.websocketEndpoint
                });
            };
            
            websocket.onmessage = function(event) {
                try {
                    const message = JSON.parse(event.data);
                    handleWebSocketMessage(message);
                } catch (error) {
                    DEBUG.log('error', 'network', 'Failed to parse WebSocket message', {
                        error: error.message,
                        data: event.data
                    });
                }
            };
            
            websocket.onclose = function(event) {
                wsConnected = false;
                stats.lastPingSuccess = false;
                
                DEBUG.log('warning', 'network', `WebSocket connection closed`, {
                    code: event.code,
                    reason: event.reason,
                    wasClean: event.wasClean
                });
                
                // Attempt to reconnect if not intentionally closed
                if (event.code !== 1000 && wsReconnectAttempts < wsMaxReconnectAttempts) {
                    wsReconnectAttempts++;
                    DEBUG.log('info', 'network', `Attempting reconnect ${wsReconnectAttempts}/${wsMaxReconnectAttempts} in ${wsReconnectDelay}ms`);
                    setTimeout(connectWebSocket, wsReconnectDelay);
                }
            };
            
            websocket.onerror = function(error) {
                wsConnected = false;
                stats.lastPingSuccess = false;
                stats.errorCount++;
                
                DEBUG.log('error', 'network', 'WebSocket error occurred', {
                    error: error,
                    readyState: websocket ? websocket.readyState : 'null'
                });
            };
            
        } catch (error) {
            DEBUG.log('error', 'network', 'Failed to create WebSocket connection', {
                error: error.message,
                endpoint: BRUMMER_CONFIG.websocketEndpoint
            });
        }
    }
    
    // Handle incoming WebSocket messages
    function handleWebSocketMessage(message) {
        DEBUG.log('info', 'network', `📥 WebSocket message: ${message.type}`, message);
        
        switch (message.type) {
            case 'connected':
                DEBUG.log('success', 'network', '🎉 WebSocket welcome received', message.data);
                break;
                
            case 'command_response':
                DEBUG.log('info', 'debug', `Command response: ${JSON.stringify(message.data).substring(0, 200)}...`);
                break;
                
            case 'telemetry':
                DEBUG.log('info', 'network', 'Telemetry broadcast received from another client');
                break;
                
            case 'command':
                // Handle REPL commands
                if (message.data && message.data.action === 'repl') {
                    handleREPLCommand(message.data);
                }
                break;
                
            default:
                DEBUG.log('info', 'network', `Unknown message type: ${message.type}`);
        }
    }
    
    // Handle REPL command from server
    function handleREPLCommand(data) {
        const { code, responseId, sessionId } = data;
        
        DEBUG.log('info', 'repl', `📝 Executing REPL command: ${code.substring(0, 100)}...`, {
            responseId,
            sessionId,
            codeLength: code.length
        });
        
        // Execute the code and capture result
        let result;
        let error = null;
        
        try {
            // Create a function that returns the result of the code
            // This allows us to handle both expressions and statements
            const AsyncFunction = Object.getPrototypeOf(async function(){}).constructor;
            const fn = new AsyncFunction('return (' + code + ')');
            
            // Execute the function and get the result
            Promise.resolve(fn()).then(res => {
                result = res;
                sendREPLResponse(responseId, result, null);
            }).catch(err => {
                // If it fails as an expression, try as statements
                try {
                    const fn2 = new AsyncFunction(code);
                    Promise.resolve(fn2()).then(res => {
                        result = res;
                        sendREPLResponse(responseId, result, null);
                    }).catch(err2 => {
                        error = err2.toString();
                        sendREPLResponse(responseId, null, error);
                    });
                } catch (err2) {
                    error = err2.toString();
                    sendREPLResponse(responseId, null, error);
                }
            });
        } catch (err) {
            // Try executing as statements if expression fails
            try {
                const AsyncFunction = Object.getPrototypeOf(async function(){}).constructor;
                const fn = new AsyncFunction(code);
                
                Promise.resolve(fn()).then(res => {
                    result = res;
                    sendREPLResponse(responseId, result, null);
                }).catch(err2 => {
                    error = err2.toString();
                    sendREPLResponse(responseId, null, error);
                });
            } catch (err2) {
                error = err2.toString();
                sendREPLResponse(responseId, null, error);
            }
        }
    }
    
    // Send REPL response back to server
    function sendREPLResponse(responseId, result, error) {
        const responseData = {
            responseId: responseId,
            result: result !== undefined ? result : null,
            error: error
        };
        
        // Try to stringify the result safely
        try {
            // Handle circular references and functions
            const seen = new WeakSet();
            responseData.result = JSON.parse(JSON.stringify(result, (key, value) => {
                if (typeof value === 'object' && value !== null) {
                    if (seen.has(value)) {
                        return '[Circular]';
                    }
                    seen.add(value);
                }
                if (typeof value === 'function') {
                    return value.toString();
                }
                return value;
            }));
        } catch (e) {
            // If serialization fails, convert to string
            responseData.result = String(result);
        }
        
        DEBUG.log('info', 'repl', `📤 Sending REPL response`, {
            responseId,
            hasResult: result !== null && result !== undefined,
            hasError: error !== null
        });
        
        sendWebSocketCommand('repl_response', responseData);
    }
    
    // Send command via WebSocket
    function sendWebSocketCommand(type, data = {}) {
        if (!websocket || websocket.readyState !== WebSocket.OPEN) {
            DEBUG.log('error', 'network', 'WebSocket not connected, cannot send command');
            return false;
        }
        
        const message = {
            type: type,
            data: data,
            timestamp: Date.now()
        };
        
        try {
            websocket.send(JSON.stringify(message));
            DEBUG.log('info', 'network', `📤 Sent WebSocket command: ${type}`, data);
            return true;
        } catch (error) {
            DEBUG.log('error', 'network', `Failed to send WebSocket command: ${error.message}`);
            return false;
        }
    }
    
    // Ping via WebSocket
    function pingWebSocket() {
        const startTime = Date.now();
        DEBUG.log('info', 'network', '🏓 Pinging via WebSocket...');
        
        if (!sendWebSocketCommand('ping')) {
            // Fallback to HTTP ping if WebSocket fails
            DEBUG.log('warning', 'network', 'WebSocket ping failed, falling back to HTTP');
            pingEndpoint();
        }
    }
    
    // Ping endpoint to test connectivity
    function pingEndpoint() {
        const startTime = Date.now();
        DEBUG.log('info', 'network', '🏓 Pinging telemetry endpoint...');
        
        fetch(BRUMMER_CONFIG.telemetryEndpoint, {
            method: 'OPTIONS',
            headers: {
                'Content-Type': 'application/json'
            }
        })
        .then(response => {
            const latency = Date.now() - startTime;
            stats.lastPingTime = DEBUG.formatDuration(latency);
            stats.lastPingSuccess = response.ok;
            
            DEBUG.log('success', 'network', `Endpoint reachable`, {
                status: response.status,
                latency: `${latency}ms`,
                headers: Object.fromEntries(response.headers.entries())
            });
        })
        .catch(error => {
            const latency = Date.now() - startTime;
            stats.lastPingTime = DEBUG.formatDuration(latency);
            stats.lastPingSuccess = false;
            stats.errorCount++;
            
            DEBUG.log('error', 'network', `Endpoint unreachable: ${error.message}`, {
                error: error.name,
                latency: `${latency}ms`,
                suggestion: 'Check if proxy server is running on the correct port'
            });
        });
    }
    
    // Send telemetry data to the server
    function sendTelemetry(data) {
        const eventData = {
            ...data,
            timestamp: Date.now(),
            sessionId: pageMetadata.sessionId,
            url: pageMetadata.url
        };
        
        telemetryBuffer.push(eventData);
        
        // Update stats
        stats.totalEvents++;
        if (stats.eventCounts[data.type]) {
            stats.eventCounts[data.type]++;
        }
        
        const bufferStatus = DEBUG.progressBar(telemetryBuffer.length, BRUMMER_CONFIG.maxBatchSize);
        DEBUG.log('info', 'buffer', `Event buffered: ${data.type}`, {
            eventType: data.type,
            bufferSize: `${telemetryBuffer.length}/${BRUMMER_CONFIG.maxBatchSize}`,
            bufferStatus: bufferStatus,
            estimatedSize: DEBUG.formatSize(JSON.stringify(eventData).length)
        });
        
        // Start batch timer if not already running
        if (!batchTimer) {
            stats.lastBatchStart = Date.now();
            batchTimer = setTimeout(flushTelemetry, BRUMMER_CONFIG.batchInterval);
            DEBUG.log('info', 'timer', `Batch timer started (${BRUMMER_CONFIG.batchInterval}ms)`);
        }
        
        // Flush immediately if buffer is full
        if (telemetryBuffer.length >= BRUMMER_CONFIG.maxBatchSize) {
            DEBUG.log('warning', 'buffer', 'Buffer full, flushing immediately');
            flushTelemetry();
        }
    }
    
    // Flush telemetry buffer
    function flushTelemetry() {
        if (telemetryBuffer.length === 0) {
            DEBUG.log('info', 'send', 'No events to flush');
            return;
        }
        
        const batch = telemetryBuffer.splice(0, telemetryBuffer.length);
        const startTime = Date.now();
        
        // Try WebSocket first if enabled and connected
        if (BRUMMER_CONFIG.useWebSocket && wsConnected && websocket && websocket.readyState === WebSocket.OPEN) {
            flushTelemetryViaWebSocket(batch, startTime);
            return;
        }
        
        // Fallback to HTTP if WebSocket not available
        flushTelemetryViaHTTP(batch, startTime);
    }
    
    // Flush telemetry via WebSocket
    function flushTelemetryViaWebSocket(batch, startTime) {
        const payload = {
            sessionId: pageMetadata.sessionId,
            events: batch,
            metadata: {
                url: pageMetadata.url,
                referrer: pageMetadata.referrer,
                userAgent: navigator.userAgent,
                timestamp: Date.now(),
                cookies: document.cookie,
                viewport: {
                    width: window.innerWidth,
                    height: window.innerHeight
                },
                connection: navigator.connection ? {
                    effectiveType: navigator.connection.effectiveType,
                    downlink: navigator.connection.downlink
                } : null
            }
        };
        
        const payloadSize = JSON.stringify(payload).length;
        stats.totalBytes += payloadSize;
        stats.networkStats.requestCount++;
        
        DEBUG.log('info', 'send', `🚀 Flushing ${batch.length} events via WebSocket`, {
            batchSize: batch.length,
            payloadSize: DEBUG.formatSize(payloadSize),
            method: 'WebSocket',
            eventTypes: batch.reduce((acc, event) => {
                acc[event.type] = (acc[event.type] || 0) + 1;
                return acc;
            }, {})
        });
        
        // Send via WebSocket
        const message = {
            type: 'telemetry',
            data: payload,
            timestamp: Date.now()
        };
        
        try {
            websocket.send(JSON.stringify(message));
            const duration = Date.now() - startTime;
            
            stats.lastSendTime = Date.now();
            stats.lastSendSuccess = true;
            stats.networkStats.successCount++;
            stats.networkStats.totalLatency += duration;
            
            DEBUG.log('success', 'send', `WebSocket telemetry sent successfully`, {
                duration: DEBUG.formatDuration(duration),
                success: true
            });
        } catch (error) {
            const duration = Date.now() - startTime;
            stats.lastSendTime = Date.now();
            stats.lastSendSuccess = false;
            stats.networkStats.failureCount++;
            stats.errorCount++;
            
            DEBUG.log('error', 'send', `WebSocket send failed: ${error.message}`, {
                error: error.name,
                duration: DEBUG.formatDuration(duration),
                suggestion: 'Falling back to HTTP'
            });
            
            // Fallback to HTTP
            flushTelemetryViaHTTP(batch, startTime);
        }
        
        // Clear the timer
        if (batchTimer) {
            clearTimeout(batchTimer);
            batchTimer = null;
            DEBUG.log('info', 'timer', 'Batch timer cleared');
        }
    }
    
    // Flush telemetry via HTTP (fallback)
    function flushTelemetryViaHTTP(batch, startTime) {
        // Use sendBeacon if available for reliability
        const payload = JSON.stringify({
            sessionId: pageMetadata.sessionId,
            events: batch,
            metadata: {
                url: pageMetadata.url,
                referrer: pageMetadata.referrer,
                userAgent: navigator.userAgent,
                timestamp: Date.now(),
                cookies: document.cookie,
                viewport: {
                    width: window.innerWidth,
                    height: window.innerHeight
                },
                connection: navigator.connection ? {
                    effectiveType: navigator.connection.effectiveType,
                    downlink: navigator.connection.downlink
                } : null
            }
        });
        
        const payloadSize = payload.length;
        stats.totalBytes += payloadSize;
        stats.networkStats.requestCount++;
        
        DEBUG.log('info', 'send', `Flushing ${batch.length} events`, {
            batchSize: batch.length,
            payloadSize: DEBUG.formatSize(payloadSize),
            method: navigator.sendBeacon ? 'sendBeacon' : 'fetch',
            endpoint: BRUMMER_CONFIG.telemetryEndpoint,
            eventTypes: batch.reduce((acc, event) => {
                acc[event.type] = (acc[event.type] || 0) + 1;
                return acc;
            }, {})
        });
        
        if (navigator.sendBeacon) {
            const success = navigator.sendBeacon(BRUMMER_CONFIG.telemetryEndpoint, payload);
            const duration = Date.now() - startTime;
            
            stats.lastSendTime = Date.now();
            stats.lastSendSuccess = success;
            
            if (success) {
                stats.networkStats.successCount++;
                stats.networkStats.totalLatency += duration;
                DEBUG.log('success', 'send', `Beacon sent successfully`, {
                    duration: DEBUG.formatDuration(duration),
                    success: true
                });
            } else {
                stats.networkStats.failureCount++;
                stats.errorCount++;
                DEBUG.log('error', 'send', 'Beacon failed to send', {
                    duration: DEBUG.formatDuration(duration),
                    suggestion: 'Browser may have blocked the beacon or endpoint is unreachable'
                });
            }
        } else {
            // Fallback to fetch with detailed error handling
            fetch(BRUMMER_CONFIG.telemetryEndpoint, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Brummer-Session': pageMetadata.sessionId,
                    'X-Brummer-Process': BRUMMER_CONFIG.processName
                },
                body: payload,
                keepalive: true
            })
            .then(response => {
                const duration = Date.now() - startTime;
                stats.lastSendTime = Date.now();
                stats.lastSendSuccess = response.ok;
                stats.networkStats.totalLatency += duration;
                
                if (response.ok) {
                    stats.networkStats.successCount++;
                    DEBUG.log('success', 'send', `Fetch completed successfully`, {
                        status: response.status,
                        statusText: response.statusText,
                        duration: DEBUG.formatDuration(duration),
                        headers: Object.fromEntries(response.headers.entries())
                    });
                } else {
                    stats.networkStats.failureCount++;
                    stats.errorCount++;
                    DEBUG.log('error', 'send', `Server returned error: ${response.status}`, {
                        status: response.status,
                        statusText: response.statusText,
                        duration: DEBUG.formatDuration(duration),
                        headers: Object.fromEntries(response.headers.entries()),
                        suggestion: response.status === 404 ? 'Telemetry endpoint not found' : 
                                   response.status === 403 ? 'Permission denied' :
                                   response.status === 500 ? 'Server error' : 'Check server logs'
                    });
                }
            })
            .catch(error => {
                const duration = Date.now() - startTime;
                stats.lastSendTime = Date.now();
                stats.lastSendSuccess = false;
                stats.networkStats.failureCount++;
                stats.errorCount++;
                
                DEBUG.log('error', 'send', `Network error: ${error.message}`, {
                    error: error.name,
                    duration: DEBUG.formatDuration(duration),
                    suggestion: error.name === 'TypeError' ? 'Network connection failed' :
                               error.name === 'AbortError' ? 'Request was aborted' :
                               'Check network connectivity and CORS settings'
                });
            });
        }
        
        // Clear the timer
        if (batchTimer) {
            clearTimeout(batchTimer);
            batchTimer = null;
            DEBUG.log('info', 'timer', 'Batch timer cleared');
        }
    }
    
    // Monitor DOM readiness and timing
    function monitorDOMTiming() {
        // Initial page state
        sendTelemetry({
            type: 'page_load',
            data: {
                readyState: document.readyState,
                referrer: pageMetadata.referrer
            }
        });
        
        // Monitor DOM readiness changes
        document.addEventListener('readystatechange', function() {
            sendTelemetry({
                type: 'dom_state',
                data: {
                    readyState: document.readyState,
                    elapsedTime: Date.now() - pageMetadata.pageLoadTime
                }
            });
        });
        
        // Monitor page visibility changes
        document.addEventListener('visibilitychange', function() {
            sendTelemetry({
                type: 'visibility_change',
                data: {
                    hidden: document.hidden,
                    visibilityState: document.visibilityState
                }
            });
        });
        
        // Monitor scroll events with debouncing
        let scrollTimer = null;
        let lastScrollPosition = { x: 0, y: 0 };
        let scrollStartTime = null;
        let totalScrollDistance = { x: 0, y: 0 };
        
        function handleScroll() {
            const currentPosition = {
                x: window.scrollX || window.pageXOffset,
                y: window.scrollY || window.pageYOffset
            };
            
            // Track scroll distance
            if (scrollStartTime) {
                totalScrollDistance.x += Math.abs(currentPosition.x - lastScrollPosition.x);
                totalScrollDistance.y += Math.abs(currentPosition.y - lastScrollPosition.y);
            } else {
                scrollStartTime = Date.now();
            }
            
            lastScrollPosition = currentPosition;
            
            // Clear existing timer
            clearTimeout(scrollTimer);
            
            // Set new timer for debounced telemetry
            scrollTimer = setTimeout(() => {
                const scrollData = {
                    type: 'user_interaction',
                    data: {
                        action: 'scroll',
                        scrollPosition: currentPosition,
                        scrollDistance: totalScrollDistance,
                        scrollDuration: Date.now() - scrollStartTime,
                        viewport: {
                            width: window.innerWidth,
                            height: window.innerHeight
                        },
                        document: {
                            width: document.documentElement.scrollWidth,
                            height: document.documentElement.scrollHeight
                        },
                        scrollPercentage: {
                            x: (currentPosition.x / (document.documentElement.scrollWidth - window.innerWidth)) * 100 || 0,
                            y: (currentPosition.y / (document.documentElement.scrollHeight - window.innerHeight)) * 100 || 0
                        },
                        timestamp: Date.now()
                    }
                };
                
                sendTelemetry(scrollData);
                
                // Reset tracking variables
                scrollStartTime = null;
                totalScrollDistance = { x: 0, y: 0 };
            }, 150); // Send after 150ms of no scrolling
        }
        
        // Add scroll listener with passive flag for performance
        window.addEventListener('scroll', handleScroll, { passive: true });
    }
    
    // Monitor Performance Metrics
    function monitorPerformance() {
        if (!BRUMMER_CONFIG.collectPerformanceMetrics || !window.performance) return;
        
        // Wait for page load to complete
        if (document.readyState === 'complete') {
            collectPerformanceMetrics();
        } else {
            window.addEventListener('load', collectPerformanceMetrics);
        }
        
        // Enhanced Performance Observer
        if ('PerformanceObserver' in window) {
            // Create observers for different entry types
            const observerConfigs = [
                {
                    entryTypes: ['longtask'],
                    handler: (entries) => {
                        entries.forEach(entry => {
                            sendTelemetry({
                                type: 'long_task',
                                data: {
                                    duration: entry.duration,
                                    startTime: entry.startTime,
                                    name: entry.name,
                                    attribution: entry.attribution
                                }
                            });
                        });
                    }
                },
                {
                    entryTypes: ['paint'],
                    handler: (entries) => {
                        entries.forEach(entry => {
                            sendTelemetry({
                                type: 'paint_timing',
                                data: {
                                    name: entry.name,
                                    startTime: entry.startTime,
                                    duration: entry.duration
                                }
                            });
                        });
                    }
                },
                {
                    entryTypes: ['largest-contentful-paint'],
                    handler: (entries) => {
                        entries.forEach(entry => {
                            sendTelemetry({
                                type: 'largest_contentful_paint',
                                data: {
                                    startTime: entry.startTime,
                                    size: entry.size,
                                    element: entry.element?.tagName,
                                    elementId: entry.element?.id,
                                    elementClass: entry.element?.className,
                                    url: entry.url
                                }
                            });
                        });
                    }
                },
                {
                    entryTypes: ['layout-shift'],
                    handler: (entries) => {
                        entries.forEach(entry => {
                            if (!entry.hadRecentInput) { // Only track shifts not caused by user input
                                sendTelemetry({
                                    type: 'layout_shift',
                                    data: {
                                        value: entry.value,
                                        startTime: entry.startTime,
                                        sources: entry.sources?.map(source => ({
                                            node: source.node?.tagName,
                                            previousRect: source.previousRect,
                                            currentRect: source.currentRect
                                        }))
                                    }
                                });
                            }
                        });
                    }
                },
                {
                    entryTypes: ['first-input'],
                    handler: (entries) => {
                        entries.forEach(entry => {
                            sendTelemetry({
                                type: 'first_input_delay',
                                data: {
                                    delay: entry.processingStart - entry.startTime,
                                    duration: entry.duration,
                                    startTime: entry.startTime,
                                    name: entry.name,
                                    target: entry.target?.tagName
                                }
                            });
                        });
                    }
                },
                {
                    entryTypes: ['navigation'],
                    handler: (entries) => {
                        entries.forEach(entry => {
                            sendTelemetry({
                                type: 'navigation_timing',
                                data: {
                                    domContentLoaded: entry.domContentLoadedEventEnd - entry.domContentLoadedEventStart,
                                    domComplete: entry.domComplete,
                                    domInteractive: entry.domInteractive,
                                    loadComplete: entry.loadEventEnd - entry.loadEventStart,
                                    type: entry.type,
                                    redirectCount: entry.redirectCount,
                                    transferSize: entry.transferSize,
                                    encodedBodySize: entry.encodedBodySize,
                                    decodedBodySize: entry.decodedBodySize,
                                    serverTiming: entry.serverTiming
                                }
                            });
                        });
                    }
                }
            ];
            
            // Try to observe each entry type
            observerConfigs.forEach(config => {
                try {
                    const observer = new PerformanceObserver(list => {
                        config.handler(list.getEntries());
                    });
                    observer.observe({ entryTypes: config.entryTypes });
                } catch (e) {
                    // Entry type not supported in this browser
                    DEBUG.log('warning', 'performance', `Performance entry type not supported: ${config.entryTypes.join(', ')}`);
                }
            });
            
            // Monitor all supported entry types
            try {
                const allTypesObserver = new PerformanceObserver(list => {
                    const entries = list.getEntries();
                    
                    // Track Web Vitals
                    entries.forEach(entry => {
                        if (entry.entryType === 'measure' && entry.name.startsWith('CLS')) {
                            sendTelemetry({
                                type: 'web_vital',
                                data: {
                                    metric: 'CLS',
                                    value: entry.duration,
                                    timestamp: Date.now()
                                }
                            });
                        }
                    });
                });
                
                if (PerformanceObserver.supportedEntryTypes) {
                    allTypesObserver.observe({ 
                        entryTypes: PerformanceObserver.supportedEntryTypes.filter(type => 
                            !['longtask', 'paint', 'largest-contentful-paint', 'layout-shift', 'first-input', 'navigation'].includes(type)
                        )
                    });
                }
            } catch (e) {
                // Fallback for older browsers
            }
        }
    }
    
    function collectPerformanceMetrics() {
        const perfData = window.performance.timing;
        const navigation = window.performance.navigation;
        
        // Calculate key metrics
        const metrics = {
            // Navigation timing
            navigationStart: perfData.navigationStart,
            redirectTime: perfData.redirectEnd - perfData.redirectStart,
            dnsTime: perfData.domainLookupEnd - perfData.domainLookupStart,
            connectTime: perfData.connectEnd - perfData.connectStart,
            requestTime: perfData.responseStart - perfData.requestStart,
            responseTime: perfData.responseEnd - perfData.responseStart,
            domProcessingTime: perfData.domComplete - perfData.domLoading,
            domContentLoadedTime: perfData.domContentLoadedEventEnd - perfData.navigationStart,
            loadCompleteTime: perfData.loadEventEnd - perfData.navigationStart,
            
            // Navigation type
            navigationType: ['navigate', 'reload', 'back_forward', 'reserved'][navigation.type] || 'unknown',
            redirectCount: navigation.redirectCount
        };
        
        // Paint timing (if available)
        if (window.performance.getEntriesByType) {
            const paintEntries = window.performance.getEntriesByType('paint');
            paintEntries.forEach(entry => {
                metrics[entry.name.replace('-', '_')] = entry.startTime;
            });
        }
        
        sendTelemetry({
            type: 'performance_metrics',
            data: metrics
        });
    }
    
    // Monitor Memory Usage
    function monitorMemory() {
        if (!BRUMMER_CONFIG.collectMemoryMetrics || !performance.memory) return;
        
        // Collect memory stats periodically
        const collectMemoryStats = () => {
            sendTelemetry({
                type: 'memory_usage',
                data: {
                    usedJSHeapSize: performance.memory.usedJSHeapSize,
                    totalJSHeapSize: performance.memory.totalJSHeapSize,
                    jsHeapSizeLimit: performance.memory.jsHeapSizeLimit,
                    percentUsed: (performance.memory.usedJSHeapSize / performance.memory.jsHeapSizeLimit) * 100
                }
            });
        };
        
        // Initial collection
        collectMemoryStats();
        
        // Collect every 10 seconds
        setInterval(collectMemoryStats, 10000);
    }
    
    // Monitor Console Output
    function monitorConsole() {
        if (!BRUMMER_CONFIG.collectConsoleMetrics) return;
        
        // Store original console methods globally for debug use
        if (!window.__brummer_originalConsole) {
            window.__brummer_originalConsole = {
                log: console.log.bind(console),
                info: console.info.bind(console),
                warn: console.warn.bind(console),
                error: console.error.bind(console),
                debug: console.debug.bind(console),
                groupCollapsed: console.groupCollapsed.bind(console),
                groupEnd: console.groupEnd.bind(console)
            };
        }
        
        // No startup message to keep console clean
        
        const originalMethods = {};
        const methodsToIntercept = ['log', 'info', 'warn', 'error', 'debug'];
        
        methodsToIntercept.forEach(method => {
            originalMethods[method] = console[method];
            console[method] = function(...args) {
                // Call original method
                originalMethods[method].apply(console, args);
                
                // Skip telemetry for Brummer's own debug messages and telemetry data
                const message = args.map(arg => String(arg)).join(' ');
                if (message.includes('BRUMMER:') || 
                    message.includes('📊 Data:') || 
                    message.includes('brummer.debug') ||
                    message.includes('BRUMMER TELEMETRY DEBUG') ||
                    message.includes('bufferSize:') ||
                    message.includes('bufferStatus:') ||
                    message.includes('eventType: "console_output"')) {
                    return;
                }
                
                // Also skip if the first argument is an object with eventType
                if (args.length > 0 && typeof args[0] === 'object' && args[0] !== null) {
                    if (args[0].eventType === 'console_output' || 
                        (args[0].bufferSize && args[0].bufferStatus)) {
                        return;
                    }
                }
                
                // Send telemetry
                try {
                    sendTelemetry({
                        type: 'console_output',
                        data: {
                            level: method,
                            message: args.map(arg => {
                                if (typeof arg === 'object') {
                                    try {
                                        return JSON.stringify(arg);
                                    } catch (e) {
                                        return String(arg);
                                    }
                                }
                                return String(arg);
                            }).join(' '),
                            stack: (new Error()).stack
                        }
                    });
                } catch (e) {
                    // Fail silently
                }
            };
        });
        
        // Monitor unhandled errors
        window.addEventListener('error', function(event) {
            sendTelemetry({
                type: 'javascript_error',
                data: {
                    message: event.message,
                    filename: event.filename,
                    lineno: event.lineno,
                    colno: event.colno,
                    stack: event.error ? event.error.stack : null
                }
            });
        });
        
        // Monitor unhandled promise rejections
        window.addEventListener('unhandledrejection', function(event) {
            sendTelemetry({
                type: 'unhandled_rejection',
                data: {
                    reason: event.reason ? String(event.reason) : 'Unknown',
                    promise: String(event.promise)
                }
            });
        });
    }
    
    // Monitor UI Interactions
    function monitorInteractions() {
        if (!BRUMMER_CONFIG.collectInteractionMetrics) return;
        
        // Throttling settings
        const interactionThrottle = 100; // ms
        const lastInteractionTime = {};
        
        // Helper to build selector path
        function buildSelectorPath(element) {
            const path = [];
            while (element && element !== document.body) {
                let selector = element.tagName.toLowerCase();
                if (element.id) {
                    selector += '#' + element.id;
                    path.unshift(selector);
                    break; // ID is unique, stop here
                } else {
                    if (element.className && typeof element.className === 'string') {
                        selector += '.' + element.className.trim().split(/\s+/).join('.');
                    }
                    // Add index if there are siblings with same tag
                    const siblings = element.parentNode ? Array.from(element.parentNode.children) : [];
                    const sameTagSiblings = siblings.filter(sibling => sibling.tagName === element.tagName);
                    if (sameTagSiblings.length > 1) {
                        const index = sameTagSiblings.indexOf(element);
                        selector += `:nth-of-type(${index + 1})`;
                    }
                    path.unshift(selector);
                }
                element = element.parentElement;
            }
            return path.join(' > ');
        }
        
        // Enhanced click tracking with throttling
        document.addEventListener('click', function(event) {
            const now = Date.now();
            const eventKey = 'click';
            
            if (lastInteractionTime[eventKey] && now - lastInteractionTime[eventKey] < interactionThrottle) {
                return; // Throttle rapid clicks
            }
            lastInteractionTime[eventKey] = now;
            
            const target = event.target;
            const rect = target.getBoundingClientRect();
            
            sendTelemetry({
                type: 'user_interaction',
                data: {
                    action: 'click',
                    targetSelector: buildSelectorPath(target),
                    targetTag: target.tagName.toLowerCase(),
                    targetId: target.id || null,
                    targetClass: target.className || null,
                    targetText: target.textContent ? target.textContent.substring(0, 100).trim() : '',
                    targetHref: target.href || null,
                    coordinates: {
                        client: { x: event.clientX, y: event.clientY },
                        page: { x: event.pageX, y: event.pageY },
                        screen: { x: event.screenX, y: event.screenY },
                        element: { 
                            x: event.clientX - rect.left, 
                            y: event.clientY - rect.top,
                            width: rect.width,
                            height: rect.height
                        }
                    },
                    viewport: {
                        width: window.innerWidth,
                        height: window.innerHeight,
                        scrollX: window.scrollX,
                        scrollY: window.scrollY
                    },
                    modifiers: {
                        alt: event.altKey,
                        ctrl: event.ctrlKey,
                        meta: event.metaKey,
                        shift: event.shiftKey
                    },
                    button: event.button,
                    timestamp: event.timeStamp
                }
            });
        }, true);
        
        // Double click tracking
        document.addEventListener('dblclick', function(event) {
            const target = event.target;
            
            sendTelemetry({
                type: 'user_interaction',
                data: {
                    action: 'double_click',
                    targetSelector: buildSelectorPath(target),
                    targetTag: target.tagName.toLowerCase(),
                    coordinates: {
                        client: { x: event.clientX, y: event.clientY },
                        page: { x: event.pageX, y: event.pageY }
                    },
                    timestamp: event.timeStamp
                }
            });
        }, true);
        
        // Context menu (right-click) tracking
        document.addEventListener('contextmenu', function(event) {
            const target = event.target;
            
            sendTelemetry({
                type: 'user_interaction',
                data: {
                    action: 'context_menu',
                    targetSelector: buildSelectorPath(target),
                    targetTag: target.tagName.toLowerCase(),
                    coordinates: {
                        client: { x: event.clientX, y: event.clientY },
                        page: { x: event.pageX, y: event.pageY }
                    },
                    timestamp: event.timeStamp
                }
            });
        }, true);
        
        // Enhanced form tracking
        document.addEventListener('submit', function(event) {
            const form = event.target;
            const formData = new FormData(form);
            const fields = [];
            
            // Collect form field metadata (not values for privacy)
            for (const element of form.elements) {
                if (element.name) {
                    fields.push({
                        name: element.name,
                        type: element.type,
                        tag: element.tagName.toLowerCase(),
                        required: element.required || false,
                        hasValue: !!element.value
                    });
                }
            }
            
            sendTelemetry({
                type: 'user_interaction',
                data: {
                    action: 'form_submit',
                    formId: form.id || null,
                    formName: form.name || null,
                    formAction: form.action,
                    formMethod: form.method,
                    formTarget: form.target || null,
                    fieldCount: fields.length,
                    fields: fields,
                    timestamp: Date.now()
                }
            });
        }, true);
        
        // Enhanced input tracking with debouncing
        let inputTimer = null;
        const inputBuffer = new Map();
        
        document.addEventListener('input', function(event) {
            const target = event.target;
            if (target.tagName !== 'INPUT' && target.tagName !== 'TEXTAREA') return;
            
            const key = buildSelectorPath(target);
            
            if (!inputBuffer.has(key)) {
                inputBuffer.set(key, {
                    startTime: Date.now(),
                    changeCount: 0,
                    fieldType: target.type || 'text',
                    fieldName: target.name || null,
                    hasValue: false
                });
            }
            
            const bufferEntry = inputBuffer.get(key);
            bufferEntry.changeCount++;
            bufferEntry.hasValue = !!target.value;
            
            // Debounce input events
            clearTimeout(inputTimer);
            inputTimer = setTimeout(() => {
                // Send all buffered input events
                inputBuffer.forEach((data, selector) => {
                    sendTelemetry({
                        type: 'user_interaction',
                        data: {
                            action: 'input_change',
                            targetSelector: selector,
                            fieldType: data.fieldType,
                            fieldName: data.fieldName,
                            changeCount: data.changeCount,
                            duration: Date.now() - data.startTime,
                            hasValue: data.hasValue,
                            timestamp: Date.now()
                        }
                    });
                });
                inputBuffer.clear();
            }, 1000); // Send after 1 second of inactivity
        }, true);
        
        // Focus/blur tracking
        let focusData = null;
        
        document.addEventListener('focus', function(event) {
            const target = event.target;
            if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.tagName === 'SELECT') {
                focusData = {
                    startTime: Date.now(),
                    selector: buildSelectorPath(target),
                    fieldType: target.type || target.tagName.toLowerCase(),
                    fieldName: target.name || null
                };
            }
        }, true);
        
        document.addEventListener('blur', function(event) {
            if (focusData && (event.target.tagName === 'INPUT' || event.target.tagName === 'TEXTAREA' || event.target.tagName === 'SELECT')) {
                const focusDuration = Date.now() - focusData.startTime;
                
                sendTelemetry({
                    type: 'user_interaction',
                    data: {
                        action: 'field_interaction',
                        targetSelector: focusData.selector,
                        fieldType: focusData.fieldType,
                        fieldName: focusData.fieldName,
                        focusDuration: focusDuration,
                        hasValue: !!event.target.value,
                        timestamp: Date.now()
                    }
                });
                
                focusData = null;
            }
        }, true);
        
        // Mouse movement tracking (heavily throttled)
        let lastMouseMove = 0;
        const mouseMoveThrottle = 5000; // Only track every 5 seconds
        
        document.addEventListener('mousemove', function(event) {
            const now = Date.now();
            if (now - lastMouseMove < mouseMoveThrottle) return;
            lastMouseMove = now;
            
            sendTelemetry({
                type: 'user_interaction',
                data: {
                    action: 'mouse_movement',
                    coordinates: {
                        client: { x: event.clientX, y: event.clientY },
                        page: { x: event.pageX, y: event.pageY }
                    },
                    viewport: {
                        width: window.innerWidth,
                        height: window.innerHeight
                    },
                    timestamp: event.timeStamp
                }
            });
        }, true);
        
        // Keyboard shortcut tracking (only special combinations)
        document.addEventListener('keydown', function(event) {
            // Only track if modifier keys are pressed
            if (event.ctrlKey || event.metaKey || event.altKey) {
                // Don't track sensitive keys like passwords
                if (event.key.length === 1 || ['Enter', 'Escape', 'Tab', 'Delete', 'Backspace'].includes(event.key)) {
                    sendTelemetry({
                        type: 'user_interaction',
                        data: {
                            action: 'keyboard_shortcut',
                            key: event.key,
                            code: event.code,
                            modifiers: {
                                alt: event.altKey,
                                ctrl: event.ctrlKey,
                                meta: event.metaKey,
                                shift: event.shiftKey
                            },
                            timestamp: event.timeStamp
                        }
                    });
                }
            }
        }, true);
    }
    
    // Monitor Network Activity (via Resource Timing API)
    function monitorNetworkActivity() {
        if (!window.performance || !window.performance.getEntriesByType) return;
        
        const checkResources = () => {
            const resources = window.performance.getEntriesByType('resource');
            const lastCheckTime = window.__brummerLastResourceCheck || 0;
            
            resources.forEach(resource => {
                if (resource.startTime > lastCheckTime) {
                    sendTelemetry({
                        type: 'resource_timing',
                        data: {
                            name: resource.name,
                            initiatorType: resource.initiatorType,
                            duration: resource.duration,
                            transferSize: resource.transferSize || 0,
                            encodedBodySize: resource.encodedBodySize || 0,
                            decodedBodySize: resource.decodedBodySize || 0,
                            startTime: resource.startTime
                        }
                    });
                }
            });
            
            window.__brummerLastResourceCheck = Date.now();
        };
        
        // Check resources periodically
        setInterval(checkResources, 5000);
    }
    
    // Advanced Network Interception (Fetch and XHR)
    function interceptNetworkRequests() {
        // Intercept Fetch API
        const originalFetch = window.fetch;
        window.fetch = async function(...args) {
            const [resource, init] = args;
            const method = init?.method || 'GET';
            const startTime = performance.now();
            const requestId = Math.random().toString(36).substr(2, 9);
            
            // Capture request details
            const requestData = {
                type: 'network_request',
                data: {
                    requestId: requestId,
                    method: method,
                    url: resource.toString(),
                    headers: init?.headers || {},
                    body: init?.body ? String(init.body).substring(0, 1000) : null,
                    timestamp: Date.now(),
                    initiator: 'fetch'
                }
            };
            
            sendTelemetry(requestData);
            
            try {
                const response = await originalFetch.apply(this, args);
                const duration = performance.now() - startTime;
                
                // Clone response to read it
                const clone = response.clone();
                
                // Capture response details
                const responseData = {
                    type: 'network_response',
                    data: {
                        requestId: requestId,
                        status: response.status,
                        statusText: response.statusText,
                        headers: Object.fromEntries(response.headers.entries()),
                        duration: duration,
                        size: response.headers.get('content-length') || 'unknown',
                        timestamp: Date.now()
                    }
                };
                
                sendTelemetry(responseData);
                
                // Optionally capture response body for JSON responses
                if (response.headers.get('content-type')?.includes('application/json')) {
                    clone.json().then(body => {
                        sendTelemetry({
                            type: 'network_response_body',
                            data: {
                                requestId: requestId,
                                body: JSON.stringify(body).substring(0, 5000),
                                bodySize: JSON.stringify(body).length
                            }
                        });
                    }).catch(() => {
                        // Ignore parsing errors
                    });
                }
                
                return response;
            } catch (error) {
                const duration = performance.now() - startTime;
                
                sendTelemetry({
                    type: 'network_error',
                    data: {
                        requestId: requestId,
                        error: error.message,
                        errorType: error.name,
                        duration: duration,
                        timestamp: Date.now()
                    }
                });
                
                throw error;
            }
        };
        
        // Intercept XMLHttpRequest
        const XHR = XMLHttpRequest.prototype;
        const originalOpen = XHR.open;
        const originalSend = XHR.send;
        const originalSetRequestHeader = XHR.setRequestHeader;
        
        XHR.open = function(method, url) {
            this._brummer = {
                method: method,
                url: url,
                headers: {},
                startTime: null,
                requestId: Math.random().toString(36).substr(2, 9)
            };
            return originalOpen.apply(this, arguments);
        };
        
        XHR.setRequestHeader = function(header, value) {
            if (this._brummer) {
                this._brummer.headers[header] = value;
            }
            return originalSetRequestHeader.apply(this, arguments);
        };
        
        XHR.send = function(body) {
            if (this._brummer) {
                this._brummer.startTime = performance.now();
                this._brummer.body = body ? String(body).substring(0, 1000) : null;
                
                sendTelemetry({
                    type: 'network_request',
                    data: {
                        requestId: this._brummer.requestId,
                        method: this._brummer.method,
                        url: this._brummer.url,
                        headers: this._brummer.headers,
                        body: this._brummer.body,
                        timestamp: Date.now(),
                        initiator: 'xhr'
                    }
                });
                
                // Monitor state changes
                this.addEventListener('readystatechange', function() {
                    if (this.readyState === 4) { // Request complete
                        const duration = performance.now() - this._brummer.startTime;
                        
                        sendTelemetry({
                            type: 'network_response',
                            data: {
                                requestId: this._brummer.requestId,
                                status: this.status,
                                statusText: this.statusText,
                                duration: duration,
                                responseSize: this.responseText?.length || 0,
                                responseHeaders: this.getAllResponseHeaders(),
                                timestamp: Date.now()
                            }
                        });
                        
                        // Capture response body for JSON responses
                        const contentType = this.getResponseHeader('content-type');
                        if (contentType?.includes('application/json') && this.responseText) {
                            try {
                                const responseBody = JSON.parse(this.responseText);
                                sendTelemetry({
                                    type: 'network_response_body',
                                    data: {
                                        requestId: this._brummer.requestId,
                                        body: JSON.stringify(responseBody).substring(0, 5000),
                                        bodySize: this.responseText.length
                                    }
                                });
                            } catch (e) {
                                // Ignore JSON parsing errors
                            }
                        }
                    }
                });
                
                this.addEventListener('error', function() {
                    const duration = performance.now() - this._brummer.startTime;
                    sendTelemetry({
                        type: 'network_error',
                        data: {
                            requestId: this._brummer.requestId,
                            error: 'Network request failed',
                            errorType: 'NetworkError',
                            duration: duration,
                            timestamp: Date.now()
                        }
                    });
                });
                
                this.addEventListener('abort', function() {
                    const duration = performance.now() - this._brummer.startTime;
                    sendTelemetry({
                        type: 'network_error',
                        data: {
                            requestId: this._brummer.requestId,
                            error: 'Request aborted',
                            errorType: 'AbortError',
                            duration: duration,
                            timestamp: Date.now()
                        }
                    });
                });
            }
            
            return originalSend.apply(this, arguments);
        };
    }
    
    // Monitor Storage Events
    function monitorStorageEvents() {
        ['localStorage', 'sessionStorage'].forEach(storageType => {
            const storage = window[storageType];
            if (!storage) return;
            
            const originalSetItem = storage.setItem;
            const originalRemoveItem = storage.removeItem;
            const originalClear = storage.clear;
            
            storage.setItem = function(key, value) {
                sendTelemetry({
                    type: 'storage_event',
                    data: {
                        storageType: storageType,
                        action: 'setItem',
                        key: key,
                        valueSize: value?.length || 0,
                        timestamp: Date.now()
                    }
                });
                
                return originalSetItem.apply(storage, arguments);
            };
            
            storage.removeItem = function(key) {
                sendTelemetry({
                    type: 'storage_event',
                    data: {
                        storageType: storageType,
                        action: 'removeItem',
                        key: key,
                        timestamp: Date.now()
                    }
                });
                
                return originalRemoveItem.apply(storage, arguments);
            };
            
            storage.clear = function() {
                sendTelemetry({
                    type: 'storage_event',
                    data: {
                        storageType: storageType,
                        action: 'clear',
                        timestamp: Date.now()
                    }
                });
                
                return originalClear.apply(storage, arguments);
            };
        });
        
        // Monitor storage events from other windows/tabs
        window.addEventListener('storage', (event) => {
            sendTelemetry({
                type: 'storage_event',
                data: {
                    storageType: event.storageArea === localStorage ? 'localStorage' : 'sessionStorage',
                    action: 'external_change',
                    key: event.key,
                    oldValue: event.oldValue?.substring(0, 100),
                    newValue: event.newValue?.substring(0, 100),
                    url: event.url,
                    timestamp: Date.now()
                }
            });
        });
        
        DEBUG.log('info', 'storage', 'Storage monitoring initialized');
    }
    
    // Monitor DOM Mutations
    function monitorDOMMutations() {
        if (!window.MutationObserver) return;
        
        const mutationSummary = {
            added: 0,
            removed: 0,
            attributeChanged: 0,
            textChanged: 0
        };
        
        let mutationTimer = null;
        const MUTATION_BATCH_DELAY = 1000; // Batch mutations for 1 second
        
        const sendMutationSummary = () => {
            if (Object.values(mutationSummary).some(v => v > 0)) {
                sendTelemetry({
                    type: 'dom_mutation',
                    data: {
                        summary: {...mutationSummary},
                        timestamp: Date.now()
                    }
                });
                
                // Reset counters
                mutationSummary.added = 0;
                mutationSummary.removed = 0;
                mutationSummary.attributeChanged = 0;
                mutationSummary.textChanged = 0;
            }
        };
        
        const mutationObserver = new MutationObserver((mutations) => {
            mutations.forEach(mutation => {
                switch (mutation.type) {
                    case 'childList':
                        mutationSummary.added += mutation.addedNodes.length;
                        mutationSummary.removed += mutation.removedNodes.length;
                        
                        // Track significant additions (e.g., new sections, scripts)
                        mutation.addedNodes.forEach(node => {
                            if (node.nodeType === Node.ELEMENT_NODE) {
                                const element = node;
                                if (element.tagName === 'SCRIPT' || element.tagName === 'LINK' || element.tagName === 'STYLE') {
                                    sendTelemetry({
                                        type: 'dom_resource_added',
                                        data: {
                                            tagName: element.tagName,
                                            src: element.src || element.href || 'inline',
                                            timestamp: Date.now()
                                        }
                                    });
                                }
                            }
                        });
                        break;
                        
                    case 'attributes':
                        mutationSummary.attributeChanged++;
                        
                        // Track specific attribute changes
                        if (mutation.target.nodeType === Node.ELEMENT_NODE) {
                            const element = mutation.target;
                            if (mutation.attributeName === 'class' || mutation.attributeName === 'style') {
                                // Track style/class changes on body or major containers
                                if (element === document.body || element.tagName === 'HTML') {
                                    sendTelemetry({
                                        type: 'dom_style_change',
                                        data: {
                                            element: element.tagName,
                                            attribute: mutation.attributeName,
                                            oldValue: mutation.oldValue,
                                            newValue: element.getAttribute(mutation.attributeName),
                                            timestamp: Date.now()
                                        }
                                    });
                                }
                            }
                        }
                        break;
                        
                    case 'characterData':
                        mutationSummary.textChanged++;
                        break;
                }
            });
            
            // Batch mutations
            clearTimeout(mutationTimer);
            mutationTimer = setTimeout(sendMutationSummary, MUTATION_BATCH_DELAY);
        });
        
        // Start observing
        mutationObserver.observe(document.body, {
            childList: true,
            attributes: true,
            characterData: true,
            subtree: true,
            attributeOldValue: true,
            characterDataOldValue: false // Don't track old text values to save memory
        });
        
        DEBUG.log('info', 'dom', 'DOM mutation observer started');
    }
    
    // Create floating badge UI
    function createFloatingBadge() {
        if (!document.body) return;
        
        // Create badge container
        const badge = document.createElement('div');
        badge.id = '__brummer_badge';
        badge.style.cssText = `
            position: fixed;
            bottom: 20px;
            right: 20px;
            background: #1e293b;
            color: #e2e8f0;
            padding: 8px 12px;
            border-radius: 6px;
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            font-size: 12px;
            font-weight: 500;
            box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
            z-index: 999999;
            cursor: pointer;
            transition: all 0.2s ease;
            display: flex;
            align-items: center;
            gap: 6px;
            user-select: none;
        `;
        
        // Create icon
        const icon = document.createElement('span');
        icon.innerHTML = '🐝';
        icon.style.fontSize = '14px';
        
        // Create text
        const text = document.createElement('span');
        text.textContent = 'Brummer';
        
        // Create status indicator
        const status = document.createElement('span');
        status.id = '__brummer_status';
        status.style.cssText = `
            width: 8px;
            height: 8px;
            background: #22c55e;
            border-radius: 50%;
            display: inline-block;
        `;
        
        badge.appendChild(icon);
        badge.appendChild(text);
        badge.appendChild(status);
        
        // Add hover effect
        badge.addEventListener('mouseenter', () => {
            badge.style.transform = 'translateY(-2px)';
            badge.style.boxShadow = '0 6px 12px rgba(0, 0, 0, 0.15)';
        });
        
        badge.addEventListener('mouseleave', () => {
            badge.style.transform = 'translateY(0)';
            badge.style.boxShadow = '0 4px 6px rgba(0, 0, 0, 0.1)';
        });
        
        // Toggle debug panel on click
        let debugPanelVisible = false;
        badge.addEventListener('click', () => {
            if (debugPanelVisible) {
                const panel = document.getElementById('__brummer_debug_panel');
                if (panel) panel.remove();
                debugPanelVisible = false;
            } else {
                createDebugPanel();
                debugPanelVisible = true;
            }
        });
        
        document.body.appendChild(badge);
        
        // Update status indicator based on connection
        setInterval(() => {
            const statusEl = document.getElementById('__brummer_status');
            if (statusEl) {
                if (wsConnected || stats.lastPingSuccess) {
                    statusEl.style.background = '#22c55e'; // Green for connected
                } else {
                    statusEl.style.background = '#ef4444'; // Red for disconnected
                }
            }
        }, 1000);
    }
    
    // Create debug panel (hidden by default)
    function createDebugPanel() {
        const panel = document.createElement('div');
        panel.id = '__brummer_debug_panel';
        panel.style.cssText = `
            position: fixed;
            bottom: 60px;
            right: 20px;
            width: 300px;
            max-height: 400px;
            background: #1e293b;
            color: #e2e8f0;
            border-radius: 8px;
            box-shadow: 0 8px 16px rgba(0, 0, 0, 0.2);
            z-index: 999998;
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            font-size: 12px;
            overflow: hidden;
            display: flex;
            flex-direction: column;
        `;
        
        // Header
        const header = document.createElement('div');
        header.style.cssText = `
            padding: 12px 16px;
            background: #0f172a;
            border-bottom: 1px solid #334155;
            font-weight: 600;
            display: flex;
            justify-content: space-between;
            align-items: center;
        `;
        header.innerHTML = `
            <span>Brummer Debug Panel</span>
            <span style="cursor: pointer; font-size: 16px;" onclick="document.getElementById('__brummer_debug_panel').remove()">×</span>
        `;
        
        // Content
        const content = document.createElement('div');
        content.style.cssText = `
            padding: 16px;
            overflow-y: auto;
            flex: 1;
        `;
        
        // Add stats
        const updateContent = () => {
            content.innerHTML = `
                <div style="margin-bottom: 12px;">
                    <strong>Connection:</strong> ${wsConnected ? '🟢 WebSocket' : (stats.lastPingSuccess ? '🟡 HTTP' : '🔴 Offline')}
                </div>
                <div style="margin-bottom: 12px;">
                    <strong>Events Sent:</strong> ${stats.totalEvents}
                </div>
                <div style="margin-bottom: 12px;">
                    <strong>Buffer:</strong> ${telemetryBuffer.length}/${BRUMMER_CONFIG.maxBatchSize}
                </div>
                <div style="margin-bottom: 12px;">
                    <strong>Errors:</strong> ${stats.errorCount}
                </div>
                <hr style="border: none; border-top: 1px solid #334155; margin: 12px 0;">
                <div style="margin-bottom: 8px;">
                    <strong>Debug Commands:</strong>
                </div>
                <code style="display: block; background: #0f172a; padding: 8px; border-radius: 4px; font-size: 11px;">
                    __brummer.debug.status()<br>
                    __brummer.debug.timeline()<br>
                    __brummer.debug.flush()<br>
                    __brummer.debug.clear()
                </code>
            `;
        };
        
        updateContent();
        panel.appendChild(header);
        panel.appendChild(content);
        document.body.appendChild(panel);
        
        // Update stats every second
        const updateInterval = setInterval(() => {
            if (document.getElementById('__brummer_debug_panel')) {
                updateContent();
            } else {
                clearInterval(updateInterval);
            }
        }, 1000);
    }
    
    // Initialize monitoring
    function initialize() {
        // No startup banner to keep console clean
        
        // Connect WebSocket if enabled
        if (BRUMMER_CONFIG.useWebSocket) {
            setTimeout(() => {
                connectWebSocket();
            }, 500);
        }
        
        // Ping endpoint to test connectivity
        setTimeout(() => {
            if (BRUMMER_CONFIG.useWebSocket) {
                // WebSocket ping will happen after connection is established
                DEBUG.log('info', 'network', 'WebSocket ping will be performed after connection');
            } else {
                pingEndpoint();
            }
        }, 1000);
        
        // Send initial telemetry
        sendTelemetry({
            type: 'monitor_initialized',
            data: {
                config: BRUMMER_CONFIG,
                pageMetadata: pageMetadata,
                capabilities: {
                    sendBeacon: !!navigator.sendBeacon,
                    performanceObserver: 'PerformanceObserver' in window,
                    memoryAPI: !!performance.memory,
                    connectionAPI: !!navigator.connection,
                    storageAPI: typeof(Storage) !== "undefined"
                },
                environment: {
                    language: navigator.language,
                    platform: navigator.platform,
                    cookieEnabled: navigator.cookieEnabled,
                    onLine: navigator.onLine,
                    viewport: {
                        width: window.innerWidth,
                        height: window.innerHeight,
                        devicePixelRatio: window.devicePixelRatio
                    }
                }
            }
        });
        
        // Create floating badge UI
        setTimeout(() => {
            createFloatingBadge();
        }, 100);
        
        // Start monitors
        monitorDOMTiming();
        monitorPerformance();
        monitorMemory();
        monitorConsole();
        monitorInteractions();
        monitorNetworkActivity();
        interceptNetworkRequests();
        monitorStorageEvents();
        monitorDOMMutations();
        
        // Send a test event to verify the system is working
        sendTelemetry({
            type: 'brummer_init',
            data: {
                message: 'Brummer monitoring initialized successfully',
                url: window.location.href,
                timestamp: Date.now()
            }
        });
        
        // Force immediate flush for testing
        setTimeout(() => {
            flushTelemetry();
        }, 1000);
        
        // Flush telemetry on page unload
        window.addEventListener('beforeunload', () => {
            flushTelemetry();
        });
        window.addEventListener('pagehide', () => {
            flushTelemetry();
        });
        
        // No automatic status display to keep console clean
    }
    
    // Start monitoring when DOM is ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', initialize);
    } else {
        initialize();
    }
})();