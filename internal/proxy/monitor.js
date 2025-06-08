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
        debugMode: true, // Enable visual debugging temporarily
        debugLevel: 'verbose', // verbose, normal, minimal
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
            if (!BRUMMER_CONFIG.debugMode) return;
            
            const timestamp = new Date().toLocaleTimeString();
            const icon = this.icons[level] || this.icons.debug;
            const categoryIcon = this.icons[category] || '📋';
            
            const style = `color: ${this.colors[level]}; font-weight: bold;`;
            
            console.groupCollapsed(`${icon} ${categoryIcon} BRUMMER: ${message} [${timestamp}]`);
            console.log(`%c${message}`, style);
            
            if (data) {
                console.log('📊 Data:', data);
            }
            
            // Add to debug timeline
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
            
            console.groupEnd();
        },
        
        status: function() {
            if (!window.__brummer) return;
            
            const stats = window.__brummer.stats;
            const buffer = telemetryBuffer;
            const now = Date.now();
            
            console.clear();
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
                
            default:
                DEBUG.log('info', 'network', `Unknown message type: ${message.type}`);
        }
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
        
        // Monitor Long Tasks (if supported)
        if ('PerformanceObserver' in window) {
            try {
                const longTaskObserver = new PerformanceObserver(function(list) {
                    for (const entry of list.getEntries()) {
                        sendTelemetry({
                            type: 'long_task',
                            data: {
                                duration: entry.duration,
                                startTime: entry.startTime,
                                name: entry.name
                            }
                        });
                    }
                });
                longTaskObserver.observe({ entryTypes: ['longtask'] });
            } catch (e) {
                // Long task observer not supported
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
        
        // Single startup message
        console.log('%c🐝 Brummer: Console monitoring active', 'color: #4b5563; font-style: italic;');
        
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
        
        // Track clicks
        document.addEventListener('click', function(event) {
            const target = event.target;
            const tagName = target.tagName.toLowerCase();
            
            // Build a selector path
            let selector = tagName;
            if (target.id) {
                selector += '#' + target.id;
            }
            if (target.className) {
                selector += '.' + target.className.split(' ').join('.');
            }
            
            sendTelemetry({
                type: 'user_interaction',
                data: {
                    action: 'click',
                    targetSelector: selector,
                    targetText: target.textContent ? target.textContent.substring(0, 100) : '',
                    x: event.clientX,
                    y: event.clientY,
                    timestamp: event.timeStamp
                }
            });
        }, true);
        
        // Track form submissions
        document.addEventListener('submit', function(event) {
            const form = event.target;
            sendTelemetry({
                type: 'user_interaction',
                data: {
                    action: 'form_submit',
                    formId: form.id,
                    formAction: form.action,
                    formMethod: form.method
                }
            });
        }, true);
        
        // Track focus events on input fields
        let focusStartTime = null;
        document.addEventListener('focus', function(event) {
            if (event.target.tagName === 'INPUT' || event.target.tagName === 'TEXTAREA') {
                focusStartTime = Date.now();
            }
        }, true);
        
        document.addEventListener('blur', function(event) {
            if ((event.target.tagName === 'INPUT' || event.target.tagName === 'TEXTAREA') && focusStartTime) {
                const focusDuration = Date.now() - focusStartTime;
                sendTelemetry({
                    type: 'user_interaction',
                    data: {
                        action: 'input_focus',
                        fieldType: event.target.type || 'text',
                        fieldName: event.target.name,
                        focusDuration: focusDuration
                    }
                });
                focusStartTime = null;
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
    
    // Initialize monitoring
    function initialize() {
        // Display startup banner
        if (BRUMMER_CONFIG.debugMode) {
            console.log('%c🚀 BRUMMER TELEMETRY INITIALIZED', 'font-size: 20px; font-weight: bold; color: #22c55e; background: #f0fdf4; padding: 10px;');
            console.log('%c═══════════════════════════════════════', 'color: #22c55e;');
            console.log(`${DEBUG.icons.performance} Process: ${BRUMMER_CONFIG.processName}`);
            console.log(`${DEBUG.icons.network} HTTP Endpoint: ${BRUMMER_CONFIG.telemetryEndpoint}`);
            console.log(`${DEBUG.icons.network} WebSocket Endpoint: ${BRUMMER_CONFIG.websocketEndpoint}`);
            console.log(`${DEBUG.icons.timer} Batch Interval: ${BRUMMER_CONFIG.batchInterval}ms`);
            console.log(`${DEBUG.icons.buffer} Max Batch Size: ${BRUMMER_CONFIG.maxBatchSize} events`);
            console.log(`🔌 Use WebSocket: ${BRUMMER_CONFIG.useWebSocket ? 'YES' : 'NO'}`);
            console.log('%c═══════════════════════════════════════', 'color: #22c55e;');
            console.log('%cDebug Commands:', 'font-weight: bold; color: #3b82f6;');
            console.log('  __brummer.debug.status() - Show telemetry dashboard');
            console.log('  __brummer.debug.timeline() - Show event timeline');
            console.log('  __brummer.debug.headers() - Show request context');
            console.log('  __brummer.debug.ping() - Test endpoint connectivity');
            console.log('  __brummer.debug.flush() - Force send buffered events');
            console.log('  __brummer.debug.clear() - Clear event buffer');
            console.log('  __brummer.debug.ws() - Get WebSocket connection');
            console.log('  __brummer.debug.reconnect() - Reconnect WebSocket');
            console.log('%c═══════════════════════════════════════', 'color: #22c55e;');
        }
        
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
        
        // Start monitors
        monitorDOMTiming();
        monitorPerformance();
        monitorMemory();
        monitorConsole();
        monitorInteractions();
        monitorNetworkActivity();
        
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
            console.log('🔧 BRUMMER: Forcing telemetry flush for testing...');
            flushTelemetry();
        }, 1000);
        
        // Flush telemetry on page unload
        window.addEventListener('beforeunload', () => {
            flushTelemetry();
        });
        window.addEventListener('pagehide', () => {
            flushTelemetry();
        });
        
        // Status display timer
        if (BRUMMER_CONFIG.debugMode && BRUMMER_CONFIG.debugLevel === 'verbose') {
            setInterval(() => {
                if (telemetryBuffer.length > 0 || stats.totalEvents > 0) {
                    DEBUG.status();
                }
            }, 10000); // Show status every 10 seconds if there's activity
        }
    }
    
    // Start monitoring when DOM is ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', initialize);
    } else {
        initialize();
    }
})();