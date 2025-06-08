// Brummer Web Monitoring Script
// This script is injected into proxied web pages to collect telemetry data

(function() {
    'use strict';
    
    // Configuration
    const BRUMMER_CONFIG = {
        telemetryEndpoint: (function() {
            // Try to determine the proxy server URL
            // Default to localhost:8888 (standard Brummer proxy port)
            const proxyHost = window.__brummerProxyHost || 'localhost:8888';
            return 'http://' + proxyHost + '/__brummer_telemetry__';
        })(),
        batchInterval: 2000, // Send data every 2 seconds
        maxBatchSize: 100,
        collectInteractionMetrics: true,
        collectPerformanceMetrics: true,
        collectMemoryMetrics: true,
        collectConsoleMetrics: true,
        processName: window.__brummerProcessName || 'unknown'
    };
    
    // Telemetry buffer
    const telemetryBuffer = [];
    let batchTimer = null;
    
    // Page metadata
    const pageMetadata = {
        url: window.location.href,
        referrer: document.referrer,
        userAgent: navigator.userAgent,
        sessionId: generateSessionId(),
        pageLoadTime: Date.now()
    };
    
    // Generate a unique session ID
    function generateSessionId() {
        return 'brummer_' + Date.now() + '_' + Math.random().toString(36).substr(2, 9);
    }
    
    // Send telemetry data to the server
    function sendTelemetry(data) {
        telemetryBuffer.push({
            ...data,
            timestamp: Date.now(),
            sessionId: pageMetadata.sessionId,
            url: pageMetadata.url
        });
        
        // Start batch timer if not already running
        if (!batchTimer) {
            batchTimer = setTimeout(flushTelemetry, BRUMMER_CONFIG.batchInterval);
        }
        
        // Flush immediately if buffer is full
        if (telemetryBuffer.length >= BRUMMER_CONFIG.maxBatchSize) {
            flushTelemetry();
        }
    }
    
    // Flush telemetry buffer
    function flushTelemetry() {
        if (telemetryBuffer.length === 0) return;
        
        const batch = telemetryBuffer.splice(0, telemetryBuffer.length);
        
        // Use sendBeacon if available for reliability
        const payload = JSON.stringify({
            sessionId: pageMetadata.sessionId,
            events: batch
        });
        
        if (navigator.sendBeacon) {
            navigator.sendBeacon(BRUMMER_CONFIG.telemetryEndpoint, payload);
        } else {
            // Fallback to fetch
            fetch(BRUMMER_CONFIG.telemetryEndpoint, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: payload,
                keepalive: true
            }).catch(() => {
                // Silently fail - we don't want to disrupt the page
            });
        }
        
        // Clear the timer
        if (batchTimer) {
            clearTimeout(batchTimer);
            batchTimer = null;
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
        
        const originalMethods = {};
        const methodsToIntercept = ['log', 'info', 'warn', 'error', 'debug'];
        
        methodsToIntercept.forEach(method => {
            originalMethods[method] = console[method];
            console[method] = function(...args) {
                // Call original method
                originalMethods[method].apply(console, args);
                
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
        // Send initial telemetry
        sendTelemetry({
            type: 'monitor_initialized',
            data: {
                config: BRUMMER_CONFIG,
                pageMetadata: pageMetadata
            }
        });
        
        // Start monitors
        monitorDOMTiming();
        monitorPerformance();
        monitorMemory();
        monitorConsole();
        monitorInteractions();
        monitorNetworkActivity();
        
        // Flush telemetry on page unload
        window.addEventListener('beforeunload', flushTelemetry);
        window.addEventListener('pagehide', flushTelemetry);
    }
    
    // Start monitoring when DOM is ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', initialize);
    } else {
        initialize();
    }
})();