package proxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
	
	"github.com/beagle/brummer/internal/mcp"
)

type ProxyServer struct {
	targetURL    *url.URL
	proxyPort    int
	mcpServer    *mcp.Server
	processName  string
	injectorJS   string
	server       *http.Server
	shutdown     chan struct{}
}

func NewProxyServer(targetURL string, proxyPort int, mcpServer *mcp.Server, processName string) (*ProxyServer, error) {
	target, err := url.Parse(targetURL)
	if err != nil {
		return nil, err
	}
	
	// Load injector script (embedded as string for now)
	injectorJS := getInjectorJS()
	
	return &ProxyServer{
		targetURL:   target,
		proxyPort:   proxyPort,
		mcpServer:   mcpServer,
		processName: processName,
		injectorJS:  injectorJS,
		shutdown:    make(chan struct{}),
	}, nil
}

func (p *ProxyServer) Start() error {
	// Create reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(p.targetURL)
	
	// Modify response to inject script
	proxy.ModifyResponse = func(resp *http.Response) error {
		contentType := resp.Header.Get("Content-Type")
		
		// Only inject into HTML responses
		if strings.Contains(contentType, "text/html") {
			// Read the body
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			resp.Body.Close()
			
			// Generate token for this session
			token := p.mcpServer.GenerateURLToken(p.processName)
			
			// Prepare injection script with actual values
			script := strings.ReplaceAll(p.injectorJS, "{{BRUMMER_ENDPOINT}}", 
				fmt.Sprintf("http://localhost:%d/api/browser-log", p.mcpServer.GetPort()))
			script = strings.ReplaceAll(script, "{{BRUMMER_TOKEN}}", token)
			script = strings.ReplaceAll(script, "{{PROCESS_NAME}}", p.processName)
			
			// Inject before </body> or </html>
			injectionPoint := "</body>"
			if !strings.Contains(string(body), injectionPoint) {
				injectionPoint = "</html>"
			}
			
			if strings.Contains(string(body), injectionPoint) {
				injection := fmt.Sprintf("<script>%s</script>%s", script, injectionPoint)
				body = bytes.Replace(body, []byte(injectionPoint), []byte(injection), 1)
			} else {
				// No proper injection point, append to end
				body = append(body, []byte(fmt.Sprintf("<script>%s</script>", script))...)
			}
			
			// Update content length
			resp.Body = io.NopCloser(bytes.NewReader(body))
			resp.ContentLength = int64(len(body))
			resp.Header.Set("Content-Length", fmt.Sprintf("%d", len(body)))
			
			// Remove content security policy that might block inline scripts
			resp.Header.Del("Content-Security-Policy")
			resp.Header.Del("X-Frame-Options")
		}
		
		return nil
	}
	
	// Create HTTP server
	mux := http.NewServeMux()
	mux.Handle("/", proxy)
	
	p.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", p.proxyPort),
		Handler: mux,
	}
	
	// Start server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()
	
	// Wait for shutdown signal or error
	select {
	case <-p.shutdown:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return p.server.Shutdown(ctx)
	case err := <-errChan:
		return err
	}
}

func (p *ProxyServer) Stop() error {
	close(p.shutdown)
	if p.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return p.server.Shutdown(ctx)
	}
	return nil
}

func getInjectorJS() string {
	return `// JavaScript to inject into proxied pages
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
                'Authorization': 'Bearer ' + BRUMMER_TOKEN
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
            message: event.message + ' at ' + event.filename + ':' + event.lineno + ':' + event.colno,
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
                message: 'Navigated to ' + location.href
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
                    message: response.status + ' ' + response.statusText + ' - ' + url + ' (' + (Date.now() - startTime) + 'ms)'
                });
                return response;
            })
            .catch(error => {
                sendToBrummer({
                    type: 'network-error',
                    level: 'error',
                    message: 'Network error - ' + url + ': ' + error.message
                });
                throw error;
            });
    };
    
    console.log('🐝 Brummer proxy logging enabled');
})();`
}